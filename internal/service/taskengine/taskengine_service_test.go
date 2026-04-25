package taskengine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"TagMatrix/internal/model"
)

func setupTestDB(t *testing.T, dbName string) string {
	dbPath := filepath.Join(os.TempDir(), dbName)
	_ = os.Remove(dbPath)

	err := model.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	return dbPath
}

func teardownTestEnv(dbPath string) {
	_ = os.Remove(dbPath)
}

func TestTaskEngine_RunAndRollback(t *testing.T) {
	dbPath := setupTestDB(t, "test_taskengine.db")
	defer teardownTestEnv(dbPath)

	ctx := context.WithValue(context.Background(), CtxKeyIsTest, true)
	svc := NewTaskEngineService(ctx)

	// 1. 准备测试数据 (3条)
	records := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25},
		{"id": 2, "name": "Bob", "age": 15},
		{"id": 3, "name": "Charlie", "age": 30},
	}
	for _, rec := range records {
		data, _ := json.Marshal(rec)
		model.DB.Create(&model.RawDataRecord{Data: string(data)})
	}

	// 2. 准备规则 (age > 18)
	ruleJSON := `{"field":"age","operator":"greater_than","value":18}`
	rule := model.SysMatchRule{
		TagID:    100, // 假设 TagID 100
		Name:     "Adults",
		RuleJSON: ruleJSON,
	}
	model.DB.Create(&rule)

	// 3. 执行打标任务 (异步)
	batchID, err := svc.RunTaggingTask([]uint64{rule.ID}, "TestBatch", true, "multiple", "")
	if err != nil {
		t.Fatalf("RunTaggingTask failed: %v", err)
	}

	// 由于是异步执行，我们稍微等一下 (在测试中不太优雅，但这里简单处理)
	time.Sleep(1 * time.Second)

	// 4. 验证打标结果
	var tags []model.SysEntityTag
	model.DB.Where("batch_id = ?", batchID).Find(&tags)

	if len(tags) != 2 {
		t.Fatalf("Expected 2 tags generated, got %d", len(tags))
	}

	// 验证日志是否生成
	var logs []model.TagTaskLog
	model.DB.Where("batch_id = ?", batchID).Find(&logs)
	if len(logs) != 2 {
		t.Fatalf("Expected 2 logs generated, got %d", len(logs))
	}

	// 5. 测试任务回退 (Rollback)
	err = svc.RollbackTask(context.Background(), batchID)
	if err != nil {
		t.Fatalf("RollbackTask failed: %v", err)
	}

	// 验证标签是否被删除
	var tagsAfterRollback []model.SysEntityTag
	model.DB.Where("batch_id = ?", batchID).Find(&tagsAfterRollback)
	if len(tagsAfterRollback) != 0 {
		t.Fatalf("Expected tags to be 0 after rollback, got %d", len(tagsAfterRollback))
	}

	// 验证批次状态是否变更为 rolled_back
	var batch model.TagTaskBatch
	model.DB.First(&batch, batchID)
	if batch.Status != "rolled_back" {
		t.Fatalf("Expected batch status 'rolled_back', got '%s'", batch.Status)
	}
}
