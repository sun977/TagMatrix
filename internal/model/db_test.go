package model

import (
	"os"
	"testing"
)

func TestInitDB(t *testing.T) {
	// 使用内存数据库或临时文件测试
	testDBPath := "test_data.db"

	// 确保测试前没有遗留的 db
	_ = os.Remove(testDBPath)

	err := InitDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	if DB == nil {
		t.Fatal("Expected DB to be initialized, got nil")
	}

	// 插入一条标签数据测试 CRUD
	tag := SysTag{
		Name:        "TestTag",
		Color:       "#FFFFFF",
		Description: "A test tag",
	}

	result := DB.Create(&tag)
	if result.Error != nil {
		t.Fatalf("Failed to create tag: %v", result.Error)
	}

	var fetchedTag SysTag
	DB.First(&fetchedTag, tag.ID)

	if fetchedTag.Name != "TestTag" {
		t.Errorf("Expected tag name 'TestTag', got '%s'", fetchedTag.Name)
	}

	// 清理测试数据库
	_ = os.Remove(testDBPath)
}
