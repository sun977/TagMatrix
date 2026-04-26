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

// UpdateTag 更新标签的基本信息（名称、颜色、描述）
func (s *TagLogicService) UpdateTag(tag *model.SysTag) error {
	if tag.ID == 0 {
		return fmt.Errorf("tag id cannot be empty")
	}
	if tag.Name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}
	// 仅更新允许修改的字段，防止意外修改 Path 和 ParentID 等结构字段
	return s.db.Model(tag).Updates(map[string]interface{}{
		"name":        tag.Name,
		"color":       tag.Color,
		"description": tag.Description,
	}).Error
}

// GetTagTree 获取所有标签并组装为树形结构
func (s *TagLogicService) GetTagTree() ([]model.TagTreeNode, error) {
	var tags []model.SysTag
	err := s.db.Find(&tags).Error
	if err != nil {
		return nil, err
	}

	// 提前查询出所有规则，构建映射字典
	var rules []model.SysMatchRule
	s.db.Find(&rules)
	ruleMap := make(map[uint64]bool)
	for _, r := range rules {
		ruleMap[r.TagID] = true
	}

	return buildTagTree(tags, 0, ruleMap), nil
}

// 递归构建标签树
func buildTagTree(tags []model.SysTag, parentID uint64, ruleMap map[uint64]bool) []model.TagTreeNode {
	var tree []model.TagTreeNode
	for _, tag := range tags {
		if tag.ParentID == parentID {
			node := model.TagTreeNode{
				SysTag:  tag,
				HasRule: ruleMap[tag.ID],
			}
			children := buildTagTree(tags, tag.ID, ruleMap)
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

// DeleteTag 删除标签（包含子标签，并在事务中级联删除关联的匹配规则）
func (s *TagLogicService) DeleteTag(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var tag model.SysTag
		if err := tx.First(&tag, id).Error; err != nil {
			return err
		}

		// 1. 找到所有子标签的 ID
		var children []model.SysTag
		if err := tx.Where("path LIKE ?", tag.Path+"%").Find(&children).Error; err != nil {
			return err
		}

		var tagIDs []uint64
		for _, child := range children {
			tagIDs = append(tagIDs, child.ID)
		}

		// 2. 级联删除这些标签关联的所有匹配规则
		if len(tagIDs) > 0 {
			// 在软删除前将 is_enabled 置为 false
			if err := tx.Model(&model.SysMatchRule{}).Where("tag_id IN ?", tagIDs).Update("is_enabled", false).Error; err != nil {
				return err
			}
			if err := tx.Where("tag_id IN ?", tagIDs).Delete(&model.SysMatchRule{}).Error; err != nil {
				return err
			}
			// 删除相关的打标结果关联 (sys_entity_tags)
			if err := tx.Where("tag_id IN ?", tagIDs).Delete(&model.SysEntityTag{}).Error; err != nil {
				return err
			}
		}

		// 3. 删除标签及其子标签
		if err := tx.Where("path LIKE ?", tag.Path+"%").Delete(&model.SysTag{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// CheckTagHasRules 检查标签或其子标签是否配置了匹配规则
func (s *TagLogicService) CheckTagHasRules(id uint64) (bool, error) {
	var tag model.SysTag
	if err := s.db.First(&tag, id).Error; err != nil {
		return false, err
	}

	var children []model.SysTag
	if err := s.db.Where("path LIKE ?", tag.Path+"%").Find(&children).Error; err != nil {
		return false, err
	}

	var tagIDs []uint64
	for _, child := range children {
		tagIDs = append(tagIDs, child.ID)
	}

	if len(tagIDs) == 0 {
		return false, nil
	}

	var count int64
	if err := s.db.Model(&model.SysMatchRule{}).Where("tag_id IN ?", tagIDs).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// ----------------- 标签导入导出 (Import/Export) -----------------

// ExportTags 导出标签树为 JSON 文件
func (s *TagLogicService) ExportTags(exportPath string) error {
	tree, err := s.GetTagTree()
	if err != nil {
		return fmt.Errorf("failed to get tag tree: %w", err)
	}

	exportTree := s.convertToExportNodes(tree)

	data, err := json.MarshalIndent(exportTree, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tag tree: %w", err)
	}

	return os.WriteFile(exportPath, data, 0644)
}

func (s *TagLogicService) convertToExportNodes(nodes []model.TagTreeNode) []model.ExportTagNode {
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

		// 级联查询并挂载匹配规则
		var rule model.SysMatchRule
		if err := s.db.Where("tag_id = ?", node.ID).First(&rule).Error; err == nil {
			exportNode.RuleName = rule.Name
			exportNode.RuleJSON = rule.RuleJSON
		}

		if len(node.Children) > 0 {
			exportNode.Children = s.convertToExportNodes(node.Children)
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

		// 处理级联导入的匹配规则
		if node.RuleJSON != "" {
			var rule model.SysMatchRule
			err := tx.Where("tag_id = ?", currentID).First(&rule).Error
			switch err {
			case gorm.ErrRecordNotFound:
				// 不存在则创建
				newRule := model.SysMatchRule{
					TagID:     currentID,
					Name:      node.RuleName,
					RuleJSON:  node.RuleJSON,
					Priority:  0,
					IsEnabled: true, // 默认为启用状态
				}
				if err := tx.Create(&newRule).Error; err != nil {
					return err
				}
			case nil:
				// 存在则更新
				rule.Name = node.RuleName
				rule.RuleJSON = node.RuleJSON
				if err := tx.Save(&rule).Error; err != nil {
					return err
				}
			default:
				// 其他数据库错误
				return err
			}
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
	if rule.DatasetID == 0 {
		return fmt.Errorf("dataset_id cannot be empty")
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

// GetRulesByDataset 获取某个数据集下的所有规则 (打标任务引擎使用的批量拉取接口)
func (s *TagLogicService) GetRulesByDataset(datasetID uint64) ([]model.SysMatchRule, error) {
	var rules []model.SysMatchRule
	err := s.db.Where("dataset_id = ?", datasetID).Find(&rules).Error
	return rules, err
}

// GetRulesByTagAndDataset 按标签和数据集获取规则
func (s *TagLogicService) GetRulesByTagAndDataset(tagID uint64, datasetID uint64) (*model.SysMatchRule, error) {
	var rule model.SysMatchRule
	err := s.db.Where("tag_id = ? AND dataset_id = ?", tagID, datasetID).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// DeleteRule 删除规则
func (s *TagLogicService) DeleteRule(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 在软删除前将 is_enabled 置为 false
		if err := tx.Model(&model.SysMatchRule{}).Where("id = ?", id).Update("is_enabled", false).Error; err != nil {
			return err
		}
		// 执行软删除
		return tx.Delete(&model.SysMatchRule{}, id).Error
	})
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
