package dataset

import (
	"fmt"
	"TagMatrix/internal/model"

	"gorm.io/gorm"
)

// DatasetService 数据集管理服务
type DatasetService struct {
	db *gorm.DB
}

// NewDatasetService 创建 DatasetService 实例
func NewDatasetService() *DatasetService {
	return &DatasetService{
		db: model.DB,
	}
}

// CreateDataset 创建新的数据集
func (s *DatasetService) CreateDataset(name, description string) (*model.SysDataset, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	dataset := &model.SysDataset{
		Name:        name,
		Description: description,
		SchemaKeys:  "[]", // 默认空 JSON 数组
	}

	if err := s.db.Create(dataset).Error; err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	return dataset, nil
}

// UpdateDataset 更新数据集信息
func (s *DatasetService) UpdateDataset(id uint64, name, description string) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	updates := map[string]interface{}{
		"name":        name,
		"description": description,
	}

	if err := s.db.Model(&model.SysDataset{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update dataset: %w", err)
	}

	return nil
}

// DeleteDataset 删除数据集及关联数据
func (s *DatasetService) DeleteDataset(id uint64) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 开启事务，删除数据集本身及其关联的所有数据
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除数据集
		if err := tx.Delete(&model.SysDataset{}, id).Error; err != nil {
			return err
		}
		// 2. 删除原始数据
		if err := tx.Where("dataset_id = ?", id).Delete(&model.RawDataRecord{}).Error; err != nil {
			return err
		}
		// 3. 删除匹配规则
		if err := tx.Where("dataset_id = ?", id).Delete(&model.SysMatchRule{}).Error; err != nil {
			return err
		}
		// 4. 删除打标批次
		if err := tx.Where("dataset_id = ?", id).Delete(&model.TagTaskBatch{}).Error; err != nil {
			return err
		}
		// (可选) 更深度的清理比如 sys_entity_tags 和 tag_task_logs
		// 因为这些表跟 RecordID 和 BatchID 关联，如果要彻底清理，可以通过子查询或让后续软删除清理脚本处理。
		// 这里暂作基础删除。

		return nil
	})
}

// ListDatasets 查询所有数据集列表
func (s *DatasetService) ListDatasets() ([]model.SysDataset, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var datasets []model.SysDataset
	if err := s.db.Order("id desc").Find(&datasets).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch datasets: %w", err)
	}

	return datasets, nil
}

// GetDataset 获取单个数据集详情
func (s *DatasetService) GetDataset(id uint64) (*model.SysDataset, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var dataset model.SysDataset
	if err := s.db.First(&dataset, id).Error; err != nil {
		return nil, fmt.Errorf("dataset not found: %w", err)
	}

	return &dataset, nil
}
