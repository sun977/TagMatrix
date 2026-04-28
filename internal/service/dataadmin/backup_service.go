package dataadmin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BackupService struct {
	db        *gorm.DB
	dbPath    string
	backupDir string
}

func NewBackupService(db *gorm.DB, appDataDir string) *BackupService {
	return &BackupService{
		db:        db,
		dbPath:    filepath.Join(appDataDir, "tagmatrix.db"),
		backupDir: filepath.Join(appDataDir, "backups"),
	}
}

type BackupInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	Note      string `json:"note"`
}

// ListBackups 列出所有备份
func (s *BackupService) ListBackups() ([]BackupInfo, error) {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Name:      entry.Name(),
			Path:      filepath.Join(s.backupDir, entry.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
			Note:      "", // Note can be extracted from filename if encoded, omitted for simplicity
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})

	return backups, nil
}

// CreateBackup 创建备份
func (s *BackupService) CreateBackup(note string) error {
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	safeNote := strings.ReplaceAll(note, " ", "_")
	if safeNote == "" {
		safeNote = "manual"
	}
	filename := fmt.Sprintf("backup_%s_%s.db", timestamp, safeNote)
	destPath := filepath.Join(s.backupDir, filename)

	src, err := os.Open(s.dbPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	return err
}

// DeleteBackup 删除备份
func (s *BackupService) DeleteBackup(backupPath string) error {
	return os.Remove(backupPath)
}

// RestoreDatabase 恢复数据库 (需高危谨慎使用，调用方需保证重启应用)
func (s *BackupService) RestoreDatabase(backupPath string) error {
	// 断开现有数据库连接
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	// 必须主动关闭，释放文件锁
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %v", err)
	}

	// 使用 io.Copy 覆盖当前数据库文件
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %v", err)
	}
	defer src.Close()

	dest, err := os.OpenFile(s.dbPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to open current database file for writing: %v", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return fmt.Errorf("failed to overwrite database file: %v", err)
	}

	// 这里的 db 已经被关闭，无法再使用，需前端执行 WindowReload 以重新启动整个应用进程和后端的 App.startup
	return nil
}
