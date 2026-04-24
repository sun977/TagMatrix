package aiengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"TagMatrix/internal/model"
)

// setupTestDB 创建一个临时的 SQLite DB 用于测试 Schema 提取
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

func TestAIEngineService_GetSchema(t *testing.T) {
	dbPath := setupTestDB(t, "test_aiengine_schema.db")
	defer teardownTestEnv(dbPath)

	// 不传真实 API Key，这里只测试 Schema 获取逻辑
	svc := NewAIEngineService()

	schema, err := svc.getSchema()
	if err != nil {
		t.Fatalf("getSchema failed: %v", err)
	}

	// 验证 schema 字符串中是否包含关键表的建表语句
	expectedTables := []string{
		"CREATE TABLE `raw_data_records`",
		"CREATE TABLE `sys_tags`",
		"CREATE TABLE `tag_task_batches`",
	}

	for _, expected := range expectedTables {
		if !strings.Contains(schema, expected) {
			t.Errorf("Expected schema to contain '%s', but it didn't. Schema output:\n%s", expected, schema)
		}
	}
}

// 模拟 AI 通信 (跳过真实的 API 请求)
// 在实际业务中可能需要 Mock client 或者跳过此测试
func TestAIEngineService_ChatWithAI_Mock(t *testing.T) {
	t.Skip("Skipping real API call test. Run this manually with a valid API key if needed.")

	// dbPath := setupTestDB(t, "test_aiengine_chat.db")
	// defer teardownTestEnv(dbPath)
	//
	// apiKey := os.Getenv("OPENAI_API_KEY")
	// if apiKey == "" {
	// 	t.Skip("No OPENAI_API_KEY set")
	// }
	//
	// svc := NewAIEngineService(apiKey, "")
	// resp, err := svc.ChatWithAI(context.Background(), "帮我写一个查询标签数量的 SQL", "")
	// if err != nil {
	// 	t.Fatalf("ChatWithAI failed: %v", err)
	// }
	//
	// t.Logf("AI Response:\n%s", resp)
}
