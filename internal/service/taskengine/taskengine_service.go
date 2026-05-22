// ============================================================================
//  ___________              _____          __         .__
//  \__    ___/____    ____ /     \ _____ _/  |________|__|__  ___
//    |    |  \__  \  / ___/  \ /  \\__  \\   __\_  __ \  \  \/  /
//    |    |   / __ \/ /_/  >  Y    \/ __ \|  |  |  | \/  |>    <
//    |____|  (____  /\___  /____|__  (____  /__|  |__|  |__/__/\_ \
//                 \//_____/        \/     \/                     \/
// ============================================================================
// ⚡️ TagMatrix :: Task Execution Engine
//
// 👤 SYSTEM_ARCHITECT : sun977 (SunHaobo)
// 🌐 GITHUB_REF       : https://github.com/sun977
// 📧 CONTACT_MAIL     : jiuwei977@foxmail.com
// 📅 INIT_YEAR        : 2026
//
// 📝 [DESC] 负责海量数据的打标任务流转、状态追踪、批量高性能入库与事务级回退 (Rollback) 的核心调度器。
//
// 💡 "A somewhat obsessive developer in cybersecurity & AI scenarios."
// ============================================================================

package taskengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/pkg/logger"
	"TagMatrix/internal/pkg/matcher"

	"github.com/gen2brain/beeep"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// TaskEngineService 处理打标任务的异步调度与回退逻辑
type TaskEngineService struct {
	db  *gorm.DB
	ctx context.Context
}

type contextKey string

// CtxKeyIsTest 用于在测试时禁用 Wails 事件推送
const CtxKeyIsTest contextKey = "isTest"

// NewTaskEngineService 创建 TaskEngineService 实例
func NewTaskEngineService(ctx context.Context) *TaskEngineService {
	return &TaskEngineService{
		db:  model.DB,
		ctx: ctx,
	}
}

// emitProgress 封装进度推送逻辑，便于统一管理和测试环境规避
func (s *TaskEngineService) emitProgress(batchID uint64, progress int, processed int, total int64, status string) {
	// 如果是测试环境 (ctx 中包含 CtxKeyIsTest) 或 ctx 无效，则跳过推送
	if s.ctx == nil || s.ctx.Err() != nil || s.ctx.Value(CtxKeyIsTest) != nil {
		return
	}
	runtime.EventsEmit(s.ctx, "taskProgress", map[string]interface{}{
		"batchID":   batchID,
		"progress":  progress,
		"processed": processed,
		"total":     total,
		"status":    status,
	})
}

// tagTaskContext 用于在 Goroutine 中传递任务上下文
type tagTaskContext struct {
	BatchID     uint64
	Rule        *model.SysMatchRule
	MRule       matcher.MatchRule
	Records     []model.RawDataRecord
	IsOverwrite bool   // 是否为覆盖模式
	TagMode     string // 打标模式：single, multiple, mixed
}

// RunTaggingTask 异步执行规则打标任务
// datasetID: 任务针对的数据集
// ruleIDs: 需要执行的规则ID列表
// batchName: 此次打标任务的自定义名称
// desc: 任务描述
// isOverwrite: 是否为覆盖模式（清除原有标签）
// tagMode: 打标模式（single: 单标签, multiple: 多标签, mixed: 混合模式）
func (s *TaskEngineService) RunTaggingTask(datasetID uint64, ruleIDs []uint64, batchName string, desc string, isOverwrite bool, tagMode string, sourceFile string) (uint64, error) {
	if datasetID == 0 {
		return 0, fmt.Errorf("dataset_id cannot be empty")
	}
	if len(ruleIDs) == 0 {
		return 0, fmt.Errorf("no rules provided")
	}

	// 1. 获取规则并解析 (确保规则属于该数据集)
	var rules []model.SysMatchRule
	if err := s.db.Where("id IN ? AND dataset_id = ?", ruleIDs, datasetID).Find(&rules).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch rules: %w", err)
	}
	if len(rules) == 0 {
		return 0, fmt.Errorf("no valid rules found for this dataset")
	}

	// 提取规则名称用于展示
	var ruleNames []string
	for _, r := range rules {
		ruleNames = append(ruleNames, r.Name)
	}
	rulesStr := strings.Join(ruleNames, ", ")
	if len(rulesStr) > 500 {
		rulesStr = rulesStr[:497] + "..." // 截断防止过长
	}

	execStrategy := "append"
	if isOverwrite {
		execStrategy = "overwrite"
	}

	// 2. 生成任务批次
	batchID := uint64(time.Now().UnixMilli())
	if batchName == "" {
		batchName = fmt.Sprintf("AutoTag_%d", batchID)
	}

	batch := model.TagTaskBatch{
		BaseModel:    model.BaseModel{ID: batchID},
		DatasetID:    datasetID,
		Name:         batchName,
		Desc:         desc,
		Status:       "running",
		TagMode:      tagMode,
		SourceFile:   sourceFile,
		ExecStrategy: execStrategy,
		Rules:        rulesStr,
	}
	if err := s.db.Create(&batch).Error; err != nil {
		return 0, fmt.Errorf("failed to create task batch: %w", err)
	}

	// 3. 异步启动打标引擎
	go s.executeTask(batchID, datasetID, rules, isOverwrite, tagMode, sourceFile)

	return batchID, nil
}

// GetAvailableSourceFiles 获取可用的来源文件列表
func (s *TaskEngineService) GetAvailableSourceFiles(ctx context.Context, datasetID uint64) ([]model.SourceFileOption, error) {
	var results []model.SourceFileOption

	query := s.db.WithContext(ctx).Table("raw_data_records").
		Select("json_extract(data, '$.\"TagM_sourceFile\"') as source_name, count(id) as count").
		Where("deleted_at IS NULL AND json_extract(data, '$.\"TagM_sourceFile\"') IS NOT NULL")

	if datasetID > 0 {
		query = query.Where("dataset_id = ?", datasetID)
	}

	err := query.Group("json_extract(data, '$.\"TagM_sourceFile\"')").
		Order("source_name ASC").
		Find(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch source files: %w", err)
	}
	return results, nil
}

// parsedRule 用于预解析的规则结构体
type parsedRule struct {
	model *model.SysMatchRule
	tag   *model.SysTag
	mRule matcher.MatchRule
}

// executeTask 核心调度引擎，使用 Worker Pool 模式流式处理海量数据
func (s *TaskEngineService) executeTask(batchID uint64, datasetID uint64, rules []model.SysMatchRule, isOverwrite bool, tagMode string, sourceFile string) {
	log.Printf("[TaskEngine] Starting batch %d, datasetID: %d, sourceFile: %s", batchID, datasetID, sourceFile)

	// 提取所有涉及的 TagID
	// [优化]: 此处提前批量提取并查询相关的 Tag 信息，是为了避免在海量数据执行 MDCT 仲裁时引发 N+1 查库导致性能崩溃。
	// 同时，提前将 Tag 的定义描述 (Description) 挂载到内存结构中，以便为后续大模型的冲突裁决提供精准的业务上下文。
	var tagIDs []uint64
	for _, r := range rules {
		tagIDs = append(tagIDs, r.TagID)
	}

	// 批量查询 Tags
	var tags []model.SysTag
	tagMap := make(map[uint64]*model.SysTag)
	if len(tagIDs) > 0 {
		s.db.Where("id IN ?", tagIDs).Find(&tags)
		for i := range tags {
			tagMap[tags[i].ID] = &tags[i]
		}
	}

	// 预解析规则
	var pRules []parsedRule
	for i := range rules {
		var mr matcher.MatchRule
		if err := json.Unmarshal([]byte(rules[i].RuleJSON), &mr); err == nil {
			pRules = append(pRules, parsedRule{
				model: &rules[i],
				tag:   tagMap[rules[i].TagID],
				mRule: mr,
			})
		}
	}

	// 初始化 MDCT 打分器
	appConfig := config.GetConfig()
	mdctScorer := NewMDCTScorer(appConfig.MDCT)

	// 初始化 Worker Pool
	workerCount := appConfig.Adv.TaskConcurrency
	if workerCount <= 0 {
		workerCount = 20
	}
	jobsChan := make(chan []model.RawDataRecord, 100)
	var wg sync.WaitGroup
	var totalProcessed int
	var mu sync.Mutex // 保护 totalProcessed

	// 初始化 GlobalCounter 实例
	gc := matcher.NewGlobalCounter()

	// 先获取总记录数用于进度计算
	var totalRecords int64
	query := s.db.Model(&model.RawDataRecord{}).Where("dataset_id = ?", datasetID)
	if sourceFile != "" && sourceFile != "all" {
		query = query.Where("json_extract(data, '$.\"TagM_sourceFile\"') = ?", sourceFile)
	}
	query.Count(&totalRecords)

	// 发送初始进度事件
	s.emitProgress(batchID, 0, 0, totalRecords, "running")

	// 启动 Workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for records := range jobsChan {
				s.processRecords(s.ctx, gc, batchID, records, pRules, isOverwrite, tagMode, mdctScorer)

				mu.Lock()
				totalProcessed += len(records)
				currentProcessed := totalProcessed
				mu.Unlock()

				// 计算并推送进度
				if totalRecords > 0 {
					progress := float64(currentProcessed) / float64(totalRecords) * 100.0
					if progress > 100 {
						progress = 100
					}
					s.emitProgress(batchID, int(progress), currentProcessed, totalRecords, "running")
				}
			}
		}()
	}

	// 流式读取数据库 (FindInBatches) 丢入 Channel
	batchSize := 1000
	var results []model.RawDataRecord
	query = s.db.Model(&model.RawDataRecord{}).Where("dataset_id = ?", datasetID)
	if sourceFile != "" && sourceFile != "all" {
		query = query.Where("json_extract(data, '$.\"TagM_sourceFile\"') = ?", sourceFile)
	}
	err := query.FindInBatches(&results, batchSize, func(tx *gorm.DB, batch int) error {
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

	progress := 100
	if status == "failed" {
		progress = 0 // 或者保持原有进度，或者根据需要设置
	}
	s.emitProgress(batchID, progress, totalProcessed, totalRecords, status)

	logger.Info(fmt.Sprintf("[TaskEngine] Finished batch %d. Processed: %d", batchID, totalProcessed))

	// 发送系统通知 (根据配置)
	appConfig = config.GetConfig()
	if appConfig.System.TaskNotification {
		title := "TagMatrix"
		msg := fmt.Sprintf("打标任务 (批次: %d) 已完成！共处理数据: %d 条", batchID, totalProcessed)
		if status == "failed" {
			msg = fmt.Sprintf("打标任务 (批次: %d) 执行失败！", batchID)
		}
		// 可以忽略错误，因为不同系统环境下的通知权限或环境支持可能不同，不应该阻塞核心业务
		_ = beeep.Notify(title, msg, "")
	}
}

// processRecords 处理一小批数据（由 Worker 执行）
func (s *TaskEngineService) processRecords(ctx context.Context, gc *matcher.GlobalCounter, batchID uint64, records []model.RawDataRecord, pRules []parsedRule, isOverwrite bool, tagMode string, scorer *MDCTScorer) {
	var logs []model.TagTaskLog
	var tags []model.SysEntityTag
	var recordIDsToClear []uint64 // 记录需要清除原有 auto_rule 标签的 recordID

	for _, record := range records {
		var dataMap map[string]interface{}
		if err := json.Unmarshal([]byte(record.Data), &dataMap); err != nil {
			continue
		}

		// 为当前行初始化行级计数器，并构建行级别的上下文
		rc := matcher.NewRowCounter()
		rowCtx := matcher.WithGlobalCounter(ctx, gc)
		rowCtx = matcher.WithRowCounter(rowCtx, rc)

		// 收集该条记录命中的所有规则
		var matchedRules []parsedRule

		// 对这行数据应用所有规则
		for _, pr := range pRules {
			// 在调用 matcher.Match 前，将当前标签名称注入上下文
			tagName := ""
			if pr.tag != nil {
				tagName = pr.tag.Name
			}
			if tagName == "" && pr.model != nil {
				// 兜底处理，如果有异常没有拿到 Name，尝试从其他地方获取或者使用 RuleName 兜底
				tagName = pr.model.Name
			}
			tagCtx := matcher.WithCurrentTag(rowCtx, tagName)

			matched, err := matcher.Match(tagCtx, dataMap, pr.mRule)
			if err == nil && matched {
				matchedRules = append(matchedRules, pr)
			}
		}

		if len(matchedRules) > 0 {
			// 如果是覆盖模式，则将该记录ID加入待清理列表
			if isOverwrite {
				recordIDsToClear = append(recordIDsToClear, record.ID)
			}

			// 接入 MDCT 多维共识打标算法引擎进行打分、仲裁与排序
			scoredRules := scorer.EvaluateAndSort(ctx, dataMap, matchedRules, tagMode)

			// 根据打标模式 (tagMode) 处理命中的规则 (多标签模式不参与AI裁决)
			if tagMode == "single" && len(scoredRules) > 0 {
				// 单标签模式：只取 MDCT 综合打分最高的第一个
				scoredRules = scoredRules[:1]
			}

			// 提取所有命中的算子频次统计数据
			rowHits := rc.GetAll()

			for i, sr := range scoredRules {
				isPrimary := false
				if tagMode == "mixed" && i == 0 {
					// 混合模式：综合得分最高的第一个作为主标签
					isPrimary = true
				}

				// 提取针对当前标签的 Hits 次数（保底至少为1）
				hitCount := 1
				if sr.ParsedRule.tag != nil {
					hitCount = rowHits[sr.ParsedRule.tag.Name]
				} else if sr.ParsedRule.model != nil {
					hitCount = rowHits[sr.ParsedRule.model.Name]
				}
				if hitCount < 1 {
					hitCount = 1
				}

				// 命中规则，生成结果与日志
				tags = append(tags, model.SysEntityTag{
					RecordID:            record.ID,
					TagID:               sr.ParsedRule.model.TagID,
					Source:              "auto_rule",
					IsPrimary:           isPrimary,
					BatchID:             batchID,
					RuleID:              sr.ParsedRule.model.ID,
					IsAiIntervened:      sr.IsAiIntervened,
					AiArbitrationReason: sr.AiArbitrationReason,
					Confidence:          sr.Confidence,
					Hits:                hitCount, // 将算子上下文的计数落地到 DB
				})

				logReason := fmt.Sprintf("Matched rule: %s (MDCT FinalScore: %.2f, Confidence: %.2f%%)", sr.ParsedRule.model.Name, sr.FinalScore, sr.Confidence)
				if sr.IsAiIntervened {
					logReason += fmt.Sprintf(" [AI Intervened: %s]", sr.AiArbitrationReason)
				}

				logs = append(logs, model.TagTaskLog{
					BatchID:  batchID,
					RecordID: record.ID,
					TagID:    sr.ParsedRule.model.TagID,
					RuleID:   sr.ParsedRule.model.ID,
					Action:   "add",
					Reason:   logReason,
				})
			}
		}
	}

	// 批量插入数据库
	if len(recordIDsToClear) > 0 {
		// 在覆盖模式下，清除这些记录之前打上的自动规则标签
		s.db.Unscoped().Where("record_id IN ? AND source = 'auto_rule'", recordIDsToClear).Delete(&model.SysEntityTag{})
	}
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
		if err := tx.Unscoped().Where("batch_id = ?", batchID).Delete(&model.SysEntityTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete entity tags: %w", err)
		}

		// 2. 标记日志状态 (由于是物理删除标签，日志可以保留作为审计，或者也删掉。这里我们保留日志，但标记 Batch 状态)

		// 3. 更新 Batch 状态
		if err := tx.Model(&batch).Update("status", "rolled_back").Error; err != nil {
			return fmt.Errorf("failed to update batch status: %w", err)
		}

		s.emitProgress(batchID, 100, batch.TotalProcessed, int64(batch.TotalProcessed), "rolled_back")

		log.Printf("[TaskEngine] Rolled back batch %d", batchID)
		return nil
	})
}

// GetTaskLogs 获取某个批次的打标日志
func (s *TaskEngineService) GetTaskLogs(batchID uint64) ([]model.TagTaskLogDto, error) {
	var logs []model.TagTaskLog
	if err := s.db.Where("batch_id = ?", batchID).Find(&logs).Error; err != nil {
		return nil, err
	}

	// 批量查询相关 Tag 和 Rule 以减少查询次数
	tagMap := make(map[uint64]string)
	ruleMap := make(map[uint64]string)

	var tagIDs []uint64
	var ruleIDs []uint64
	for _, l := range logs {
		tagIDs = append(tagIDs, l.TagID)
		if l.RuleID > 0 {
			ruleIDs = append(ruleIDs, l.RuleID)
		}
	}

	if len(tagIDs) > 0 {
		var tags []model.SysTag
		s.db.Where("id IN ?", tagIDs).Find(&tags)
		for _, t := range tags {
			if t.Path != "" {
				tagMap[t.ID] = t.Path
			} else {
				tagMap[t.ID] = t.Name
			}
		}
	}

	if len(ruleIDs) > 0 {
		var rules []model.SysMatchRule
		s.db.Where("id IN ?", ruleIDs).Find(&rules)
		for _, r := range rules {
			ruleMap[r.ID] = r.Name
		}
	}

	var dtos []model.TagTaskLogDto
	for _, l := range logs {
		dtos = append(dtos, model.TagTaskLogDto{
			ID:        l.ID,
			RecordID:  l.RecordID,
			TagName:   tagMap[l.TagID],
			RuleName:  ruleMap[l.RuleID],
			Action:    l.Action,
			Reason:    l.Reason,
			CreatedAt: l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return dtos, nil
}

// GetTaskLogsPaged 分页获取某个批次的打标日志
func (s *TaskEngineService) GetTaskLogsPaged(batchID uint64, page int, pageSize int) (*model.PagedTaskLogs, error) {
	var total int64
	if err := s.db.Model(&model.TagTaskLog{}).Where("batch_id = ?", batchID).Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []model.TagTaskLog
	offset := (page - 1) * pageSize
	if err := s.db.Where("batch_id = ?", batchID).Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	// 批量查询相关 Tag 和 Rule 以减少查询次数
	tagMap := make(map[uint64]string)
	ruleMap := make(map[uint64]string)

	var tagIDs []uint64
	var ruleIDs []uint64
	for _, l := range logs {
		tagIDs = append(tagIDs, l.TagID)
		if l.RuleID > 0 {
			ruleIDs = append(ruleIDs, l.RuleID)
		}
	}

	if len(tagIDs) > 0 {
		var tags []model.SysTag
		s.db.Where("id IN ?", tagIDs).Find(&tags)
		for _, t := range tags {
			if t.Path != "" {
				tagMap[t.ID] = t.Path
			} else {
				tagMap[t.ID] = t.Name
			}
		}
	}

	if len(ruleIDs) > 0 {
		var rules []model.SysMatchRule
		s.db.Where("id IN ?", ruleIDs).Find(&rules)
		for _, r := range rules {
			ruleMap[r.ID] = r.Name
		}
	}

	var dtos []model.TagTaskLogDto
	for _, l := range logs {
		dtos = append(dtos, model.TagTaskLogDto{
			ID:        l.ID,
			RecordID:  l.RecordID,
			TagName:   tagMap[l.TagID],
			RuleName:  ruleMap[l.RuleID],
			Action:    l.Action,
			Reason:    l.Reason,
			CreatedAt: l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// If dtos is nil, make it an empty slice instead of null in JSON
	if dtos == nil {
		dtos = []model.TagTaskLogDto{}
	}

	return &model.PagedTaskLogs{
		Total: total,
		Logs:  dtos,
	}, nil
}

// DeleteTaskBatches 硬删除指定的打标批次及其关联的日志和标签
func (s *TaskEngineService) DeleteTaskBatches(ctx context.Context, batchIDs []uint64) error {
	if len(batchIDs) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 删除批次产生的标签关联记录
		if err := tx.Unscoped().Where("batch_id IN ?", batchIDs).Delete(&model.SysEntityTag{}).Error; err != nil {
			return fmt.Errorf("failed to hard delete entity tags: %w", err)
		}

		// 2. 删除批次日志
		if err := tx.Unscoped().Where("batch_id IN ?", batchIDs).Delete(&model.TagTaskLog{}).Error; err != nil {
			return fmt.Errorf("failed to hard delete task logs: %w", err)
		}

		// 3. 删除批次记录本身
		if err := tx.Unscoped().Where("id IN ?", batchIDs).Delete(&model.TagTaskBatch{}).Error; err != nil {
			return fmt.Errorf("failed to hard delete task batches: %w", err)
		}

		log.Printf("[TaskEngine] Hard deleted batches: %v", batchIDs)
		return nil
	})
}
