package model

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化本地 SQLite 数据库并进行模型迁移
func InitDB(dbPath string) error {
	var err error

	// 配置 GORM 日志
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 开发时可打印SQL
	}

	DB, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to sqlite database: %w", err)
	}

	// 自动迁移所有模型表
	err = autoMigrate()
	if err != nil {
		return fmt.Errorf("failed to auto migrate database schemas: %w", err)
	}

	log.Printf("Successfully connected to SQLite at %s and migrated models", dbPath)
	return nil
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&RawDataRecord{},
		&SysTag{},
		&SysMatchRule{},
		&TagTaskBatch{},
		&TagTaskLog{},
		&SysEntityTag{},
	)
}
