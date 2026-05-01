package model

import (
	"fmt"
	"time"

	"TagMatrix/internal/pkg/logger"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB
	gormZapLog  *logger.GormZapLogger
)

// InitDB 初始化本地 SQLite 数据库并进行模型迁移
func InitDB(dbPath string) error {
	var err error

	// 使用我们的自定义 Zap Logger 接管 GORM
	gormZapLog = logger.NewGormLogger(logger.Log, 200*time.Millisecond)

	// 配置 GORM 日志
	gormConfig := &gorm.Config{
		Logger: gormZapLog,
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

	logger.Info(fmt.Sprintf("Successfully connected to SQLite at %s and migrated models", dbPath))
	return nil
}

// UpdateDBLoggerLevel 用于在界面修改 Debug 模式后同步更新数据库日志级别
func UpdateDBLoggerLevel(isDebug bool) {
	if gormZapLog != nil {
		gormZapLog.UpdateLevel(isDebug)
	}
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&SysDataset{},
		&RawDataRecord{},
		&SysTag{},
		&SysMatchRule{},
		&TagTaskBatch{},
		&TagTaskLog{},
		&SysEntityTag{},
		&SysSqlTemplate{},
	)
}
