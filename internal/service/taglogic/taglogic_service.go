package taglogic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

// UpdateTag 更新标签的基本信息（名称、颜色、描述）并级联更新路径
func (s *TagLogicService) UpdateTag(tag *model.SysTag) error {
	if tag.ID == 0 {
		return fmt.Errorf("tag id cannot be empty")
	}
	if tag.Name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var oldTag model.SysTag
		if err := tx.First(&oldTag, tag.ID).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"name":        tag.Name,
			"color":       tag.Color,
			"description": tag.Description,
		}

		if oldTag.Name != tag.Name {
			// 名称发生改变，需要重新计算当前标签的路径
			var newPath string
			if oldTag.ParentID == 0 {
				newPath = fmt.Sprintf("/%s/", tag.Name)
			} else {
				var parent model.SysTag
				if err := tx.First(&parent, oldTag.ParentID).Error; err != nil {
					return fmt.Errorf("parent tag not found: %w", err)
				}
				newPath = fmt.Sprintf("%s%s/", parent.Path, tag.Name)
			}
			updates["path"] = newPath

			// 查找所有子标签并更新它们的路径前缀
			var children []model.SysTag
			if err := tx.Where("path LIKE ? AND id != ?", oldTag.Path+"%", tag.ID).Find(&children).Error; err != nil {
				return err
			}

			for _, child := range children {
				if len(child.Path) >= len(oldTag.Path) {
					childNewPath := newPath + child.Path[len(oldTag.Path):]
					if err := tx.Model(&child).Update("path", childNewPath).Error; err != nil {
						return err
					}
				}
			}
		}

		// 仅更新允许修改的字段，防止意外修改 ParentID 等结构字段
		return tx.Model(tag).Updates(updates).Error
	})
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

// buildTagTree 递归构建标签树
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

// GetTagByPath 根据路径精确查找标签
func (s *TagLogicService) GetTagByPath(path string) (*model.SysTag, error) {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	var tag model.SysTag
	if err := s.db.Where("path = ?", path).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
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
			if err := tx.Unscoped().Where("tag_id IN ?", tagIDs).Delete(&model.SysEntityTag{}).Error; err != nil {
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

// MoveTag 移动标签到新的父节点下，支持拖拽层级结构变化
func (s *TagLogicService) MoveTag(tagID uint64, newParentID uint64) error {
	if tagID == 0 {
		return fmt.Errorf("tag id cannot be empty")
	}
	if tagID == newParentID {
		return fmt.Errorf("cannot move tag to itself")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var currentTag model.SysTag
		if err := tx.First(&currentTag, tagID).Error; err != nil {
			return fmt.Errorf("tag not found: %w", err)
		}

		oldPath := currentTag.Path
		oldLevel := currentTag.Level

		var newParentPath string
		var newLevel int

		if newParentID > 0 {
			var parent model.SysTag
			if err := tx.First(&parent, newParentID).Error; err != nil {
				return fmt.Errorf("parent tag not found: %w", err)
			}
			// 防环校验：不能将父节点移动到它的子孙节点下面
			if strings.HasPrefix(parent.Path, oldPath) {
				return fmt.Errorf("cannot move a tag into its own descendant")
			}
			newParentPath = parent.Path
			newLevel = parent.Level + 1
		} else {
			newParentPath = ""
			newLevel = 0
		}

		// 计算当前节点的新 Path
		newPath := fmt.Sprintf("%s/%s/", strings.TrimSuffix(newParentPath, "/"), currentTag.Name)
		if newParentPath == "" {
			newPath = fmt.Sprintf("/%s/", currentTag.Name)
		}
		newPath = strings.ReplaceAll(newPath, "//", "/") // 安全处理多余斜杠

		// 1. 更新当前节点
		if err := tx.Model(&currentTag).Updates(map[string]interface{}{
			"parent_id": newParentID,
			"path":      newPath,
			"level":     newLevel,
		}).Error; err != nil {
			return fmt.Errorf("failed to update tag: %w", err)
		}

		// 2. 查找并级联更新所有子孙节点
		var descendants []model.SysTag
		if err := tx.Where("path LIKE ? AND id != ?", oldPath+"%", tagID).Find(&descendants).Error; err != nil {
			return fmt.Errorf("failed to find descendants: %w", err)
		}

		for _, desc := range descendants {
			// 替换路径前缀
			descNewPath := strings.Replace(desc.Path, oldPath, newPath, 1)
			descNewLevel := desc.Level + (newLevel - oldLevel)

			if err := tx.Model(&desc).Updates(map[string]interface{}{
				"path":  descNewPath,
				"level": descNewLevel,
			}).Error; err != nil {
				return fmt.Errorf("failed to cascade update descendant %d: %w", desc.ID, err)
			}
		}

		return nil
	})
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

	// 强制校验根节点是否是逻辑节点
	if len(mRule.And) == 0 && len(mRule.Or) == 0 && len(mRule.EvaluateAll) == 0 {
		return fmt.Errorf("无效的规则JSON：根节点必须是逻辑节点(and/or/evaluate_all)，不能直接是条件节点")
	}

	if rule.ID > 0 {
		return s.db.Save(rule).Error
	}

	// 检查是否已经存在相同标签和数据集的规则（包含软删除）
	var existingRule model.SysMatchRule
	err := s.db.Unscoped().Where("tag_id = ? AND dataset_id = ?", rule.TagID, rule.DatasetID).First(&existingRule).Error
	if err == nil {
		// 如果找到了记录
		if existingRule.DeletedAt.Valid {
			// 如果是已经软删除的，物理删除它以便为新规则腾出空间，避免 UNIQUE 约束冲突
			s.db.Unscoped().Delete(&existingRule)
		} else {
			// 如果是还在生效的，返回报错
			return fmt.Errorf("当前标签在该数据集下已存在规则，请直接编辑现有规则")
		}
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

// GetAllRules 获取所有规则
func (s *TagLogicService) GetAllRules() ([]model.SysMatchRule, error) {
	var rules []model.SysMatchRule
	err := s.db.Find(&rules).Error
	return rules, err
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
func (s *TagLogicService) DryRunRule(ruleJSON string, limit int, datasetID uint64) ([]DryRunResult, error) {
	var mRule matcher.MatchRule
	if err := json.Unmarshal([]byte(ruleJSON), &mRule); err != nil {
		return nil, fmt.Errorf("invalid rule_json format: %w", err)
	}

	var rawRecords []model.RawDataRecord
	query := s.db.Model(&model.RawDataRecord{})

	if datasetID > 0 {
		query = query.Where("dataset_id = ?", datasetID)
	}

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

		// 对于单个规则测试匹配，也需要初始化基本上下文
		ctx := context.Background()
		rc := matcher.NewRowCounter()
		rowCtx := matcher.WithRowCounter(ctx, rc)
		// 这里暂无 tag 信息，传入 "test_tag" 兜底
		tagCtx := matcher.WithCurrentTag(rowCtx, "test_tag")

		matched, err := matcher.Match(tagCtx, dataMap, mRule)
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

// ----------------- 规则克隆与继承 (Rule Cloning & Inheritance) -----------------

// CloneRuleResult 克隆规则返回的结果
type CloneRuleResult struct {
	Status        string   `json:"status"`         // "ok", "warning", "error"
	MissingFields []string `json:"missing_fields"` // 目标数据集缺失的字段
	Message       string   `json:"message"`        // 提示信息
	NewRuleID     uint64   `json:"new_rule_id"`
}

// InheritRulesResult 批量继承规则返回的结果
type InheritRulesResult struct {
	TotalCloned int               `json:"total_cloned"`
	Warnings    []CloneRuleResult `json:"warnings"` // 带有 warning 的克隆结果
}

// extractFields 递归提取规则中的 field 列表
func extractFields(rule matcher.MatchRule) []string {
	var fields []string
	if rule.Field != "" {
		fields = append(fields, rule.Field)
	}
	for _, subRule := range rule.And {
		fields = append(fields, extractFields(subRule)...)
	}
	for _, subRule := range rule.Or {
		fields = append(fields, extractFields(subRule)...)
	}
	for _, subRule := range rule.EvaluateAll {
		fields = append(fields, extractFields(subRule)...)
	}
	return fields
}

// uniqueStrings 字符串切片去重
func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// CheckRuleSchema 校验规则是否兼容目标数据集表头
func (s *TagLogicService) CheckRuleSchema(ruleJSON string, targetDatasetHeaders []string) (string, []string) {
	var mRule matcher.MatchRule
	if err := json.Unmarshal([]byte(ruleJSON), &mRule); err != nil {
		return "error", nil
	}

	fields := extractFields(mRule)

	headerMap := make(map[string]bool)
	for _, h := range targetDatasetHeaders {
		headerMap[h] = true
	}

	var missingFields []string
	for _, f := range fields {
		if f != "" && !headerMap[f] {
			missingFields = append(missingFields, f)
		}
	}

	missingFields = uniqueStrings(missingFields)
	if len(missingFields) > 0 {
		return "warning", missingFields
	}
	return "ok", nil
}

// CloneRule 克隆一条单条规则到新数据集
func (s *TagLogicService) CloneRule(sourceRuleID uint64, targetDatasetID uint64, tagID uint64) (*CloneRuleResult, error) {
	// 1. 查源规则
	var sourceRule model.SysMatchRule
	if err := s.db.First(&sourceRule, sourceRuleID).Error; err != nil {
		return nil, fmt.Errorf("source rule not found: %w", err)
	}

	// 2. 查目标数据集
	var targetDataset model.SysDataset
	if err := s.db.First(&targetDataset, targetDatasetID).Error; err != nil {
		return nil, fmt.Errorf("target dataset not found: %w", err)
	}

	var headers []string
	if err := json.Unmarshal([]byte(targetDataset.SchemaKeys), &headers); err != nil {
		headers = []string{} // 解析失败则假设为空
	}

	// 3. 校验 schema
	status, missingFields := s.CheckRuleSchema(sourceRule.RuleJSON, headers)
	if status == "error" {
		return &CloneRuleResult{Status: "error", Message: "规则 JSON 格式不合法"}, nil
	}

	// 4. 执行克隆 (深拷贝 JSON)
	newRule := model.SysMatchRule{
		DatasetID: targetDatasetID,
		TagID:     tagID,
		Name:      sourceRule.Name,
		Priority:  sourceRule.Priority,
		RuleJSON:  sourceRule.RuleJSON,
		IsEnabled: sourceRule.IsEnabled,
	}

	// 先物理删除旧规则(如果存在同一个tag+dataset组合)，避免UNIQUE约束冲突
	if err := s.db.Unscoped().Where("tag_id = ? AND dataset_id = ?", tagID, targetDatasetID).Delete(&model.SysMatchRule{}).Error; err != nil {
		return nil, err
	}

	if err := s.db.Create(&newRule).Error; err != nil {
		return nil, fmt.Errorf("failed to clone rule: %w", err)
	}

	msg := "克隆成功"
	if status == "warning" {
		msg = fmt.Sprintf("克隆成功，但缺少字段: %s", strings.Join(missingFields, ", "))
	}

	return &CloneRuleResult{
		Status:        status,
		MissingFields: missingFields,
		Message:       msg,
		NewRuleID:     newRule.ID,
	}, nil
}

// InheritRules 批量继承数据集下的所有规则
func (s *TagLogicService) InheritRules(sourceDatasetID uint64, targetDatasetID uint64) (*InheritRulesResult, error) {
	var rules []model.SysMatchRule
	if err := s.db.Where("dataset_id = ?", sourceDatasetID).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch source rules: %w", err)
	}

	var targetDataset model.SysDataset
	if err := s.db.First(&targetDataset, targetDatasetID).Error; err != nil {
		return nil, fmt.Errorf("target dataset not found: %w", err)
	}

	var headers []string
	if err := json.Unmarshal([]byte(targetDataset.SchemaKeys), &headers); err != nil {
		headers = []string{}
	}

	result := &InheritRulesResult{
		TotalCloned: 0,
		Warnings:    []CloneRuleResult{},
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			status, missingFields := s.CheckRuleSchema(rule.RuleJSON, headers)
			if status == "error" {
				// 跳过错误的规则
				continue
			}

			newRule := model.SysMatchRule{
				DatasetID: targetDatasetID,
				TagID:     rule.TagID,
				Name:      rule.Name,
				Priority:  rule.Priority,
				RuleJSON:  rule.RuleJSON,
				IsEnabled: rule.IsEnabled,
			}

			// 先物理删除旧规则(如果存在同一个tag+dataset组合)，避免UNIQUE约束冲突
			if err := tx.Unscoped().Where("tag_id = ? AND dataset_id = ?", rule.TagID, targetDatasetID).Delete(&model.SysMatchRule{}).Error; err != nil {
				return err
			}

			if err := tx.Create(&newRule).Error; err != nil {
				return err
			}

			result.TotalCloned++

			if status == "warning" {
				result.Warnings = append(result.Warnings, CloneRuleResult{
					Status:        status,
					MissingFields: missingFields,
					Message:       fmt.Sprintf("标签ID [%d] 克隆成功，但缺少字段", rule.TagID),
					NewRuleID:     newRule.ID,
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return result, nil
}
