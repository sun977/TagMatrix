package taskengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"TagMatrix/internal/model"
	"TagMatrix/internal/pkg/matcher"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// TaskEngineService 处理打标任务的异步调度与回退逻辑
type TaskEngineService struct {
	db  *gorm.DB
	ctx context.Context
}

// NewTaskEngineService 创建 TaskEngineService 实例
func NewTaskEngineService(ctx context.Context) *TaskEngineService {
	return &TaskEngineService{
		db:  model.DB,
		ctx: ctx,
	}
}

// tagTaskContext 用于在 Goroutine 中传递任务上下文
type tagTaskContext struct {
	BatchID   uint64
	Rule      *model.SysMatchRule
	MRule     matcher.MatchRule
	Records   []model.RawDataRecord
	IsPrimary bool // 是否为主标签（针对混合模式）
}

// RunTaggingTask 异步执行规则打标任务
// ruleIDs: 需要执行的规则ID列表
// batchName: 此次打标任务的自定义名称
// isPrimary: 是否将这些规则命中的标签设为主标签
func (s *TaskEngineService) RunTaggingTask(ruleIDs []uint64, batchName string, isPrimary bool) (uint64, error) {
	if len(ruleIDs) == 0 {
		return 0, fmt.Errorf("no rules provided")
	}

	// 1. 获取规则并解析
	var rules []model.SysMatchRule
	if err := s.db.Where("id IN ?", ruleIDs).Find(&rules).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch rules: %w", err)
	}
	if len(rules) == 0 {
		return 0, fmt.Errorf("no valid rules found")
	}

	// 2. 生成任务批次
	batchID := uint64(time.Now().UnixNano())
	if batchName == "" {
		batchName = fmt.Sprintf("AutoTag_%d", batchID)
	}

	batch := model.TagTaskBatch{
		BaseModel: model.BaseModel{ID: batchID},
		Name:      batchName,
		Status:    "running",
	}
	if err := s.db.Create(&batch).Error; err != nil {
		return 0, fmt.Errorf("failed to create task batch: %w", err)
	}

	// 3. 异步启动打标引擎
	go s.executeTask(batchID, rules, isPrimary)

	return batchID, nil
}

// parsedRule 用于预解析的规则结构体
type parsedRule struct {
	model *model.SysMatchRule
	mRule matcher.MatchRule
}

// executeTask 核心调度引擎，使用 Worker Pool 模式流式处理海量数据
func (s *TaskEngineService) executeTask(batchID uint64, rules []model.SysMatchRule, isPrimary bool) {
	log.Printf("[TaskEngine] Starting batch %d", batchID)

	// 预解析规则
	var pRules []parsedRule
	for i := range rules {
		var mr matcher.MatchRule
		if err := json.Unmarshal([]byte(rules[i].RuleJSON), &mr); err == nil {
			pRules = append(pRules, parsedRule{
				model: &rules[i],
				mRule: mr,
			})
		}
	}

	// 初始化 Worker Pool
	workerCount := 5 // 启动 5 个协程并发处理
	jobsChan := make(chan []model.RawDataRecord, 100)
	var wg sync.WaitGroup
	var totalProcessed int
	var mu sync.Mutex // 保护 totalProcessed

	// 先获取总记录数用于进度计算
	var totalRecords int64
	s.db.Model(&model.RawDataRecord{}).Count(&totalRecords)

	// 发送初始进度事件
	if s.ctx != nil && s.ctx.Err() == nil {
		runtime.EventsEmit(s.ctx, "taskProgress", map[string]interface{}{
			"batchID":   batchID,
			"progress":  0,
			"processed": 0,
			"total":     totalRecords,
			"status":    "running",
		})
	}

	// 启动 Workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for records := range jobsChan {
				s.processRecords(batchID, records, pRules, isPrimary)

				mu.Lock()
				totalProcessed += len(records)
				currentProcessed := totalProcessed
				mu.Unlock()

				// 计算并推送进度
				if totalRecords > 0 && s.ctx != nil && s.ctx.Err() == nil {
					progress := float64(currentProcessed) / float64(totalRecords) * 100.0
					if progress > 100 {
						progress = 100
					}
					runtime.EventsEmit(s.ctx, "taskProgress", map[string]interface{}{
						"batchID":   batchID,
						"progress":  int(progress),
						"processed": currentProcessed,
						"total":     totalRecords,
						"status":    "running",
					})
				}
			}
		}()
	}

	// 流式读取数据库 (FindInBatches) 丢入 Channel
	batchSize := 1000
	var results []model.RawDataRecord
	err := s.db.Model(&model.RawDataRecord{}).FindInBatches(&results, batchSize, func(tx *gorm.DB, batch int) error {
		// 将当前的 records 深拷贝发送给 channel，避免并发修改
		recordsCopy := make([]model.RawDataRecord, len(results))
		copy(recordsCopy, results)
		jobsChan <- recordsCopy
		return nil
	}).Error

	close(jobsChan)
	wg.Wait() // 等待所有 Worker 完成

	// 4. 更新任务状态
	now := time.Now()
	status := "completed"
	if err != nil {
		log.Printf("[TaskEngine] Batch %d failed: %v", batchID, err)
		status = "failed"
	}

	s.db.Model(&model.TagTaskBatch{}).Where("id = ?", batchID).Updates(map[string]interface{}{
		"status":          status,
		"total_processed": totalProcessed,
		"finished_at":     &now,
	})

	if s.ctx != nil && s.ctx.Err() == nil {
		progress := 100
		if status == "failed" {
			progress = 0 // 或者保持原有进度，或者根据需要设置
		}
		runtime.EventsEmit(s.ctx, "taskProgress", map[string]interface{}{
			"batchID":   batchID,
			"progress":  progress,
			"processed": totalProcessed,
			"total":     totalRecords,
			"status":    status,
		})
	}

	log.Printf("[TaskEngine] Finished batch %d. Processed: %d", batchID, totalProcessed)
}

// processRecords 处理一小批数据（由 Worker 执行）
func (s *TaskEngineService) processRecords(batchID uint64, records []model.RawDataRecord, pRules []parsedRule, isPrimary bool) {
	var logs []model.TagTaskLog
	var tags []model.SysEntityTag

	for _, record := range records {
		var dataMap map[string]interface{}
		if err := json.Unmarshal([]byte(record.Data), &dataMap); err != nil {
			continue
		}

		// 对这行数据应用所有规则
		for _, pr := range pRules {
			matched, err := matcher.Match(dataMap, pr.mRule)
			if err == nil && matched {
				// 命中规则，生成结果与日志
				tags = append(tags, model.SysEntityTag{
					RecordID:  record.ID,
					TagID:     pr.model.TagID,
					Source:    "auto_rule",
					IsPrimary: isPrimary,
					BatchID:   batchID,
					RuleID:    pr.model.ID,
				})

				logs = append(logs, model.TagTaskLog{
					BatchID:  batchID,
					RecordID: record.ID,
					TagID:    pr.model.TagID,
					RuleID:   pr.model.ID,
					Action:   "add",
					Reason:   fmt.Sprintf("Matched rule: %s", pr.model.Name),
				})
			}
		}
	}

	// 批量插入数据库
	if len(tags) > 0 {
		// 为了防止主键冲突，可以使用事务或 OnConflict
		s.db.Create(&tags)
	}
	if len(logs) > 0 {
		s.db.Create(&logs)
	}
}

// RollbackTask 回退指定的打标批次
func (s *TaskEngineService) RollbackTask(ctx context.Context, batchID uint64) error {
	// 开启事务进行回退
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch model.TagTaskBatch
		if err := tx.First(&batch, batchID).Error; err != nil {
			return fmt.Errorf("task batch not found: %w", err)
		}

		if batch.Status == "rolled_back" {
			return fmt.Errorf("task batch is already rolled back")
		}

		// 1. 删除该批次打上的标签
		if err := tx.Where("batch_id = ?", batchID).Delete(&model.SysEntityTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete entity tags: %w", err)
		}

		// 2. 标记日志状态 (由于是物理删除标签，日志可以保留作为审计，或者也删掉。这里我们保留日志，但标记 Batch 状态)

		// 3. 更新 Batch 状态
		if err := tx.Model(&batch).Update("status", "rolled_back").Error; err != nil {
			return fmt.Errorf("failed to update batch status: %w", err)
		}

		if s.ctx != nil && s.ctx.Err() == nil {
			runtime.EventsEmit(s.ctx, "taskProgress", map[string]interface{}{
				"batchID":   batchID,
				"progress":  100, // 回退完成
				"processed": batch.TotalProcessed,
				"total":     batch.TotalProcessed,
				"status":    "rolled_back",
			})
		}

		log.Printf("[TaskEngine] Rolled back batch %d", batchID)
		return nil
	})
}
