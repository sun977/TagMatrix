package dataset

import (
	"encoding/json"
	"fmt"
	"os"

	"TagMatrix/internal/model"

	"gorm.io/gorm"
)

// ExportDatasetWithRules 导出数据集及专属规则为 JSON
func (s *DatasetService) ExportDatasetWithRules(datasetID uint64, exportPath string) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 1. 获取数据集
	var dataset model.SysDataset
	if err := s.db.First(&dataset, datasetID).Error; err != nil {
		return fmt.Errorf("failed to find dataset: %w", err)
	}

	// 2. 获取该数据集的所有规则，并联表获取 tag path
	type RuleWithTagPath struct {
		model.SysMatchRule
		TagPath string `gorm:"column:tag_path"`
	}
	var dbRules []RuleWithTagPath
	if err := s.db.Table("sys_match_rules").
		Select("sys_match_rules.*, sys_tags.path as tag_path").
		Joins("JOIN sys_tags ON sys_tags.id = sys_match_rules.tag_id").
		Where("sys_match_rules.dataset_id = ? AND sys_match_rules.deleted_at IS NULL", datasetID).
		Find(&dbRules).Error; err != nil {
		return fmt.Errorf("failed to fetch rules: %w", err)
	}

	// 3. 组装 Export 数据
	exportData := model.ExportDatasetWithRules{
		Version:     "1.0",
		Name:        dataset.Name,
		Description: dataset.Description,
		SchemaKeys:  dataset.SchemaKeys,
		Rules:       make([]model.ExportRule, 0, len(dbRules)),
	}

	for _, r := range dbRules {
		exportData.Rules = append(exportData.Rules, model.ExportRule{
			TagPath:   r.TagPath,
			Name:      r.Name,
			Priority:  r.Priority,
			RuleJSON:  r.RuleJSON,
			IsEnabled: r.IsEnabled,
		})
	}

	// 4. 序列化为 JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	// 5. 写入文件
	if err := os.WriteFile(exportPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportDatasetWithRules 导入业务资产 (数据集及专属规则)
func (s *DatasetService) ImportDatasetWithRules(importPath string) (*model.ImportResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// 1. 读取并解析 JSON 文件
	data, err := os.ReadFile(importPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}

	var importData model.ExportDatasetWithRules
	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("invalid json format: %w", err)
	}

	if importData.Name == "" {
		return nil, fmt.Errorf("dataset name is empty in import file")
	}

	result := &model.ImportResult{
		DatasetName:  importData.Name,
		RuleImported: 0,
		RuleSkipped:  0,
	}

	// 2. 事务执行导入
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 查询或创建数据集
		var dataset model.SysDataset
		err := tx.Where("name = ?", importData.Name).First(&dataset).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 创建新数据集
				dataset = model.SysDataset{
					Name:        importData.Name,
					Description: importData.Description,
					SchemaKeys:  importData.SchemaKeys,
				}
				if err := tx.Create(&dataset).Error; err != nil {
					return fmt.Errorf("failed to create dataset: %w", err)
				}
			} else {
				return fmt.Errorf("failed to query dataset: %w", err)
			}
		} else {
			// 更新现有数据集
			updates := map[string]interface{}{
				"description": importData.Description,
				"schema_keys": importData.SchemaKeys,
			}
			if err := tx.Model(&dataset).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update dataset: %w", err)
			}
		}

		// 遍历导入规则
		for _, rule := range importData.Rules {
			// 查找关联的全局标签
			var tag model.SysTag
			err := tx.Where("path = ?", rule.TagPath).First(&tag).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					// 标签不存在，跳过该规则的导入
					result.RuleSkipped++
					continue
				}
				return fmt.Errorf("failed to query tag for path %s: %w", rule.TagPath, err)
			}

			// 查找是否该数据集下已有针对此标签的规则
			var existingRule model.SysMatchRule
			err = tx.Where("dataset_id = ? AND tag_id = ?", dataset.ID, tag.ID).First(&existingRule).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					// 创建新规则
					newRule := model.SysMatchRule{
						DatasetID: dataset.ID,
						TagID:     tag.ID,
						Name:      rule.Name,
						Priority:  rule.Priority,
						RuleJSON:  rule.RuleJSON,
						IsEnabled: rule.IsEnabled,
					}
					if err := tx.Create(&newRule).Error; err != nil {
						return fmt.Errorf("failed to create rule for tag %s: %w", tag.Name, err)
					}
					result.RuleImported++
				} else {
					return fmt.Errorf("failed to query existing rule for tag %s: %w", tag.Name, err)
				}
			} else {
				// 更新已存在的规则
				updates := map[string]interface{}{
					"name":       rule.Name,
					"priority":   rule.Priority,
					"rule_json":  rule.RuleJSON,
					"is_enabled": rule.IsEnabled,
				}
				if err := tx.Model(&existingRule).Updates(updates).Error; err != nil {
					return fmt.Errorf("failed to update existing rule for tag %s: %w", tag.Name, err)
				}
				result.RuleImported++
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
