package taglogic

import (
	"encoding/json"
	"fmt"

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

// GetTagTree 获取所有标签 (前端可以自行组装为树，或者后端组装)
func (s *TagLogicService) GetAllTags() ([]model.SysTag, error) {
	var tags []model.SysTag
	err := s.db.Find(&tags).Error
	return tags, err
}

// DeleteTag 删除标签
func (s *TagLogicService) DeleteTag(id uint64) error {
	// 实际业务中可能需要级联删除子标签和相关的匹配规则
	return s.db.Delete(&model.SysTag{}, id).Error
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
func (s *TagLogicService) DryRunRule(ruleJSON string, limit int) ([]DryRunResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 10 // 默认限制 10 条，防止查询过多
	}

	var mRule matcher.MatchRule
	if err := json.Unmarshal([]byte(ruleJSON), &mRule); err != nil {
		return nil, fmt.Errorf("invalid rule_json format: %w", err)
	}

	var rawRecords []model.RawDataRecord
	if err := s.db.Limit(limit).Find(&rawRecords).Error; err != nil {
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
