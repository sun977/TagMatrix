package taglogic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"TagMatrix/internal/model"
)

// 创建一个临时的 SQLite DB 用于测试
func setupTestDB(t *testing.T, dbName string) string {
	dbPath := filepath.Join(os.TempDir(), dbName)
	_ = os.Remove(dbPath)

	err := model.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	return dbPath
}

// 清理临时文件
func teardownTestEnv(dbPath string) {
	_ = os.Remove(dbPath)
}

func TestTagLogicService_CreateTag(t *testing.T) {
	dbPath := setupTestDB(t, "test_taglogic_create_tag.db")
	defer teardownTestEnv(dbPath)

	svc := NewTagLogicService()

	// 1. 创建根标签
	rootTag := &model.SysTag{
		Name:        "Root",
		Description: "Root tag",
	}
	err := svc.CreateTag(rootTag)
	if err != nil {
		t.Fatalf("Failed to create root tag: %v", err)
	}

	if rootTag.ID == 0 {
		t.Errorf("Expected root tag to have an ID, got 0")
	}
	if rootTag.Path != "/Root/" {
		t.Errorf("Expected root tag path to be '/Root/', got '%s'", rootTag.Path)
	}

	// 2. 创建子标签
	childTag := &model.SysTag{
		Name:     "Child",
		ParentID: rootTag.ID,
	}
	err = svc.CreateTag(childTag)
	if err != nil {
		t.Fatalf("Failed to create child tag: %v", err)
	}

	if childTag.Path != "/Root/Child/" {
		t.Errorf("Expected child tag path to be '/Root/Child/', got '%s'", childTag.Path)
	}
	if childTag.Level != 2 {
		t.Errorf("Expected child tag level to be 2, got %d", childTag.Level)
	}
}

func TestTagLogicService_SaveRule(t *testing.T) {
	dbPath := setupTestDB(t, "test_taglogic_save_rule.db")
	defer teardownTestEnv(dbPath)

	svc := NewTagLogicService()

	// 创建一个测试标签
	tag := &model.SysTag{Name: "RuleTest"}
	_ = svc.CreateTag(tag)

	// 1. 保存有效的规则
	validRuleJSON := `{"field":"age","operator":"greater_than","value":18}`
	rule := &model.SysMatchRule{
		TagID:    tag.ID,
		Name:     "Adults",
		RuleJSON: validRuleJSON,
	}
	err := svc.SaveRule(rule)
	if err != nil {
		t.Fatalf("Failed to save valid rule: %v", err)
	}

	// 2. 尝试保存无效的规则 (模拟前端传入错误的 JSON)
	invalidRuleJSON := `{"field":"age",` // 缺少括号
	invalidRule := &model.SysMatchRule{
		TagID:    tag.ID,
		Name:     "Invalid",
		RuleJSON: invalidRuleJSON,
	}
	err = svc.SaveRule(invalidRule)
	if err == nil {
		t.Errorf("Expected error when saving invalid rule JSON, but got nil")
	}
}

func TestTagLogicService_DryRunRule(t *testing.T) {
	dbPath := setupTestDB(t, "test_taglogic_dry_run.db")
	defer teardownTestEnv(dbPath)

	svc := NewTagLogicService()

	// 准备测试数据
	testRecords := []map[string]interface{}{
		{"id": 1, "name": "Alice", "age": 25},
		{"id": 2, "name": "Bob", "age": 15},
		{"id": 3, "name": "Charlie", "age": 30},
	}

	batchID := uint64(time.Now().UnixNano())
	for _, rec := range testRecords {
		jsonData, _ := json.Marshal(rec)
		model.DB.Create(&model.RawDataRecord{
			BatchID: batchID,
			Data:    string(jsonData),
		})
	}

	// 测试一个规则：age > 18
	ruleJSON := `{"field":"age","operator":"greater_than","value":18}`

	results, err := svc.DryRunRule(ruleJSON, 10)
	if err != nil {
		t.Fatalf("DryRunRule failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// 验证结果
	matchedCount := 0
	for _, res := range results {
		if res.Matched {
			matchedCount++
		}
	}

	if matchedCount != 2 {
		t.Errorf("Expected 2 matched records (Alice and Charlie), got %d", matchedCount)
	}
}
