package taglogic

import (
	"encoding/json"
	"fmt"
	"os"

	"TagMatrix/internal/model"
	"TagMatrix/internal/pkg/matcher"

	"gorm.io/gorm"
)

// TagLogicService 处理标签与规则的业务逻辑
type TagLogicService struct {
	db *gorm.DB
}

// NewTagLogicService 创建 TagLogicService 实例
func NewTagLogicService() *TagLogicService {
	return &TagLogicService{
		db: model.DB,
	}
}

// ----------------- 标签管理 (Tag Management) -----------------

// CreateTag 创建新标签
func (s *TagLogicService) CreateTag(tag *model.SysTag) error {
	if tag.Name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	// 简单计算 Path 和 Level (实际业务中可能需要更复杂的路径构建)
	if tag.ParentID == 0 {
		tag.Path = fmt.Sprintf("/%s/", tag.Name)
		tag.Level = 1
	} else {
		var parent model.SysTag
		if err := s.db.First(&parent, tag.ParentID).Error; err != nil {
			return fmt.Errorf("parent tag not found: %w", err)
		}
		tag.Path = fmt.Sprintf("%s%s/", parent.Path, tag.Name)
		tag.Level = parent.Level + 1
	}

	return s.db.Create(tag).Error
}

// GetTagTree 获取所有标签并组装为树形结构
func (s *TagLogicService) GetTagTree() ([]model.TagTreeNode, error) {
	var tags []model.SysTag
	err := s.db.Find(&tags).Error
	if err != nil {
		return nil, err
	}

	return buildTagTree(tags, 0), nil
}

// 递归构建标签树
func buildTagTree(tags []model.SysTag, parentID uint64) []model.TagTreeNode {
	var tree []model.TagTreeNode
	for _, tag := range tags {
		if tag.ParentID == parentID {
			node := model.TagTreeNode{
				SysTag: tag,
			}
			children := buildTagTree(tags, tag.ID)
			if len(children) > 0 {
				node.Children = children
			}
			tree = append(tree, node)
		}
	}
	return tree
}

// GetAllTags 获取所有标签 (平铺列表)
func (s *TagLogicService) GetAllTags() ([]model.SysTag, error) {
	var tags []model.SysTag
	err := s.db.Find(&tags).Error
	return tags, err
}

// DeleteTag 删除标签及其子标签
func (s *TagLogicService) DeleteTag(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 递归找到所有子孙节点 ID
		var allTags []model.SysTag
		if err := tx.Find(&allTags).Error; err != nil {
			return err
		}
		idsToDelete := getSubTagIDs(allTags, id)
		idsToDelete = append(idsToDelete, id) // 包含自己

		// 2. 删除相关的规则
		if err := tx.Where("tag_id IN ?", idsToDelete).Delete(&model.SysMatchRule{}).Error; err != nil {
			return err
		}

		// 3. 删除相关的打标结果关联 (sys_entity_tags)
		if err := tx.Where("tag_id IN ?", idsToDelete).Delete(&model.SysEntityTag{}).Error; err != nil {
			return err
		}

		// 4. 删除标签本身
		if err := tx.Where("id IN ?", idsToDelete).Delete(&model.SysTag{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func getSubTagIDs(tags []model.SysTag, parentID uint64) []uint64 {
	var ids []uint64
	for _, tag := range tags {
		if tag.ParentID == parentID {
			ids = append(ids, tag.ID)
			ids = append(ids, getSubTagIDs(tags, tag.ID)...)
		}
	}
	return ids
}

// ----------------- 标签导入导出 (Import/Export) -----------------

// ExportTags 导出标签树为 JSON 文件
func (s *TagLogicService) ExportTags(exportPath string) error {
	tree, err := s.GetTagTree()
	if err != nil {
		return fmt.Errorf("failed to get tag tree: %w", err)
	}

	exportTree := convertToExportNodes(tree)

	data, err := json.MarshalIndent(exportTree, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tag tree: %w", err)
	}

	return os.WriteFile(exportPath, data, 0644)
}

func convertToExportNodes(nodes []model.TagTreeNode) []model.ExportTagNode {
	var result []model.ExportTagNode
	for _, node := range nodes {
		exportNode := model.ExportTagNode{
			Name:        node.Name,
			ParentID:    node.ParentID,
			Path:        node.Path,
			Level:       node.Level,
			Color:       node.Color,
			Description: node.Description,
		}
		if len(node.Children) > 0 {
			exportNode.Children = convertToExportNodes(node.Children)
		}
		result = append(result, exportNode)
	}
	return result
}

// ImportTags 导入标签树 JSON 文件
func (s *TagLogicService) ImportTags(importPath string) error {
	data, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var importedTree []model.ExportTagNode
	if err := json.Unmarshal(data, &importedTree); err != nil {
		return fmt.Errorf("invalid json format: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.importTagNodes(tx, importedTree, 0)
	})
}

func (s *TagLogicService) importTagNodes(tx *gorm.DB, nodes []model.ExportTagNode, parentID uint64) error {
	for _, node := range nodes {
		// 检查当前名称在父节点下是否已存在
		var existingTag model.SysTag
		err := tx.Where("name = ? AND parent_id = ?", node.Name, parentID).First(&existingTag).Error

		var currentID uint64

		if err == gorm.ErrRecordNotFound {
			// 不存在则创建
			newTag := model.SysTag{
				Name:        node.Name,
				ParentID:    parentID,
				Color:       node.Color,
				Description: node.Description,
			}

			if parentID == 0 {
				newTag.Path = fmt.Sprintf("/%s/", newTag.Name)
				newTag.Level = 1
			} else {
				var parent model.SysTag
				tx.First(&parent, parentID)
				newTag.Path = fmt.Sprintf("%s%s/", parent.Path, newTag.Name)
				newTag.Level = parent.Level + 1
			}

			if err := tx.Create(&newTag).Error; err != nil {
				return fmt.Errorf("failed to create tag %s: %w", node.Name, err)
			}
			currentID = newTag.ID
		} else if err != nil {
			return err
		} else {
			// 已存在则复用，同时可以选择更新颜色和描述等属性
			existingTag.Color = node.Color
			existingTag.Description = node.Description
			if err := tx.Save(&existingTag).Error; err != nil {
				return err
			}
			currentID = existingTag.ID
		}

		// 递归导入子节点
		if len(node.Children) > 0 {
			if err := s.importTagNodes(tx, node.Children, currentID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ----------------- 规则管理 (Rule Management) -----------------

// SaveRule 创建或更新匹配规则
func (s *TagLogicService) SaveRule(rule *model.SysMatchRule) error {
	if rule.TagID == 0 {
		return fmt.Errorf("tag_id cannot be empty")
	}

	// 校验 rule_json 是否能被正确解析为 matcher.MatchRule
	var mRule matcher.MatchRule
	if err := json.Unmarshal([]byte(rule.RuleJSON), &mRule); err != nil {
		return fmt.Errorf("invalid rule_json format: %w", err)
	}

	if rule.ID > 0 {
		return s.db.Save(rule).Error
	}
	return s.db.Create(rule).Error
}

// GetRulesByTagID 获取某个标签下的所有规则
func (s *TagLogicService) GetRulesByTagID(tagID uint64) ([]model.SysMatchRule, error) {
	var rules []model.SysMatchRule
	err := s.db.Where("tag_id = ?", tagID).Find(&rules).Error
	return rules, err
}

// DeleteRule 删除规则
func (s *TagLogicService) DeleteRule(id uint64) error {
	return s.db.Delete(&model.SysMatchRule{}, id).Error
}

// ----------------- 试运行 (Dry Run) -----------------

// DryRunResult 试运行结果结构
type DryRunResult struct {
	RecordID string `json:"record_id"`
	Matched  bool   `json:"matched"`
	Data     string `json:"data"` // 原始数据预览
}

// DryRunRule 对给定的规则 JSON 在少量数据上进行试运行
// limit <= 0 表示查询全部数据，limit > 0 表示查询前 N 条数据
func (s *TagLogicService) DryRunRule(ruleJSON string, limit int) ([]DryRunResult, error) {
	var mRule matcher.MatchRule
	if err := json.Unmarshal([]byte(ruleJSON), &mRule); err != nil {
		return nil, fmt.Errorf("invalid rule_json format: %w", err)
	}

	var rawRecords []model.RawDataRecord
	query := s.db.Model(&model.RawDataRecord{})

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&rawRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sample data: %w", err)
	}

	var results []DryRunResult
	for _, record := range rawRecords {
		var dataMap map[string]interface{}
		if err := json.Unmarshal([]byte(record.Data), &dataMap); err != nil {
			continue // 跳过无法解析的数据
		}

		matched, err := matcher.Match(dataMap, mRule)
		if err != nil {
			matched = false // 匹配出错视作不匹配
		}

		results = append(results, DryRunResult{
			RecordID: fmt.Sprintf("%d", record.ID), // 基础模型ID为uint64，这里转为string方便前端展示
			Matched:  matched,
			Data:     record.Data,
		})
	}

	return results, nil
}
