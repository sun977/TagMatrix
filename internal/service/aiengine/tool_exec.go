package aiengine

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"TagMatrix/internal/model"

	"github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (s *AIEngineService) executeAITool(ctx context.Context, tc openai.ToolCall) string {
	switch tc.Function.Name {
	case "create_system_tag":
		var args struct {
			TagName    string `json:"tag_name"`
			ParentPath string `json:"parent_path"`
			Desc       string `json:"description"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		// 解析父路径，获取 ParentID
		parentID := uint64(0)
		if args.ParentPath != "/" && args.ParentPath != "" {
			parent, err := s.tagLogic.GetTagByPath(args.ParentPath)
			if err != nil {
				// 如果父标签不存在，返回错误让AI自己决定是否要先建父标签
				return fmt.Sprintf("{\"status\":\"error\",\"message\":\"parent path %s not found. Please create parent tag first or use / as parent_path.\"}", args.ParentPath)
			}
			parentID = parent.ID
		}

		tag := &model.SysTag{
			Name:        args.TagName,
			ParentID:    parentID,
			Description: args.Desc,
			Color:       "#52c48f", // 默认给个主题色
		}

		if err := s.tagLogic.CreateTag(tag); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return "{\"status\":\"error\",\"message\":\"Tag already exists.\"}"
			}
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		// 通知前端标签树有更新
		runtime.EventsEmit(ctx, "tag_tree_updated")

		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Tag created successfully\",\"tag_id\":%d}", tag.ID)

	case "update_system_tag":
		var args struct {
			TargetTagPath string `json:"target_tag_path"`
			NewName       string `json:"new_name"`
			NewColor      string `json:"new_color"`
			NewDesc       string `json:"new_description"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		tag, err := s.tagLogic.GetTagByPath(args.TargetTagPath)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"target tag path %s not found.\"}", args.TargetTagPath)
		}

		if args.NewName != "" {
			tag.Name = args.NewName
		}
		if args.NewColor != "" {
			tag.Color = args.NewColor
		}
		if args.NewDesc != "" {
			tag.Description = args.NewDesc
		}

		if err := s.tagLogic.UpdateTag(tag); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return "{\"status\":\"error\",\"message\":\"Tag name already exists in this level.\"}"
			}
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		runtime.EventsEmit(ctx, "tag_tree_updated")
		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Tag updated successfully\",\"tag_id\":%d}", tag.ID)

	case "move_system_tag":
		var args struct {
			TargetTagPath string `json:"target_tag_path"`
			NewParentPath string `json:"new_parent_path"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		tag, err := s.tagLogic.GetTagByPath(args.TargetTagPath)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"target tag path %s not found.\"}", args.TargetTagPath)
		}

		newParentID := uint64(0)
		if args.NewParentPath != "/" && args.NewParentPath != "" {
			parent, err := s.tagLogic.GetTagByPath(args.NewParentPath)
			if err != nil {
				// 容错处理：有时 AI 会自作主张把自己的名字带到 new_parent_path 里
				// 例如目标是 "/测试/", 但 AI 传了 "/测试/QA/"
				trimmedPath := strings.TrimSuffix(args.NewParentPath, "/")
				if strings.HasSuffix(trimmedPath, "/"+tag.Name) {
					fallbackPath := strings.TrimSuffix(trimmedPath, "/"+tag.Name)
					if fallbackPath == "" {
						fallbackPath = "/"
					}
					parent, err = s.tagLogic.GetTagByPath(fallbackPath)
				}
				if err != nil {
					return fmt.Sprintf("{\"status\":\"error\",\"message\":\"new parent path %s not found.\"}", args.NewParentPath)
				}
			}
			newParentID = parent.ID
		}

		if err := s.tagLogic.MoveTag(tag.ID, newParentID); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		runtime.EventsEmit(ctx, "tag_tree_updated")
		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Tag moved successfully\",\"tag_id\":%d}", tag.ID)

	case "create_tag_rule":
		var args struct {
			TargetTagPath string `json:"target_tag_path"`
			DatasetName   string `json:"dataset_name"`
			ConditionJSON string `json:"condition_json"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		tag, err := s.tagLogic.GetTagByPath(args.TargetTagPath)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"target tag path %s not found.\"}", args.TargetTagPath)
		}

		var dataset model.SysDataset
		if err := s.db.Where("name = ?", args.DatasetName).First(&dataset).Error; err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"dataset %s not found.\"}", args.DatasetName)
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		rule := &model.SysMatchRule{
			TagID:     tag.ID,
			DatasetID: dataset.ID,
			Name:      fmt.Sprintf("AI生成规则-%04d-Rule", r.Intn(10000)), // 规则名称增加随机数
			RuleJSON:  args.ConditionJSON,
			Priority:  0,
		}

		if err := s.tagLogic.SaveRule(rule); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		runtime.EventsEmit(ctx, "rule_list_updated")

		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Rule created successfully\",\"rule_id\":%d}", rule.ID)

	case "update_tag_rule":
		var args struct {
			TargetTagPath    string  `json:"target_tag_path"`
			DatasetName      string  `json:"dataset_name"`
			NewConditionJSON *string `json:"new_condition_json"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		tag, err := s.tagLogic.GetTagByPath(args.TargetTagPath)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"target tag path %s not found.\"}", args.TargetTagPath)
		}

		var dataset model.SysDataset
		if err := s.db.Where("name = ?", args.DatasetName).First(&dataset).Error; err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"dataset %s not found.\"}", args.DatasetName)
		}

		rule, err := s.tagLogic.GetRulesByTagAndDataset(tag.ID, dataset.ID)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"rule not found for tag %s in dataset %s.\"}", args.TargetTagPath, args.DatasetName)
		}

		if args.NewConditionJSON != nil && *args.NewConditionJSON != "" {
			rule.RuleJSON = *args.NewConditionJSON
		}
		// IsAction 暂时在 model.SysMatchRule 中没有对应字段（可能包含在 RuleJSON 内部），这里先预留接口

		if err := s.tagLogic.SaveRule(rule); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		runtime.EventsEmit(ctx, "rule_list_updated")
		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Rule updated successfully\",\"rule_id\":%d}", rule.ID)

	case "delete_tag_rule":
		var args struct {
			TargetTagPath string `json:"target_tag_path"`
			DatasetName   string `json:"dataset_name"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		tag, err := s.tagLogic.GetTagByPath(args.TargetTagPath)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"target tag path %s not found.\"}", args.TargetTagPath)
		}

		var dataset model.SysDataset
		if err := s.db.Where("name = ?", args.DatasetName).First(&dataset).Error; err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"dataset %s not found.\"}", args.DatasetName)
		}

		rule, err := s.tagLogic.GetRulesByTagAndDataset(tag.ID, dataset.ID)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"rule not found for tag %s in dataset %s.\"}", args.TargetTagPath, args.DatasetName)
		}

		if err := s.tagLogic.DeleteRule(rule.ID); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		runtime.EventsEmit(ctx, "rule_list_updated")
		return "{\"status\":\"success\",\"message\":\"Rule deleted successfully\"}"

	case "execute_tagging_task":
		var args struct {
			DatasetID uint64   `json:"dataset_id"`
			RuleIDs   []uint64 `json:"rule_ids"`
			TagMode   string   `json:"tag_mode"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		if s.RunTaggingTaskFunc == nil {
			return "{\"status\":\"error\",\"message\":\"RunTaggingTaskFunc is not initialized\"}"
		}

		batchID, err := s.RunTaggingTaskFunc(ctx, args.DatasetID, args.RuleIDs, args.TagMode)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		// 通知前端任务列表更新，并可以切换到任务视图
		runtime.EventsEmit(ctx, "task_list_updated")
		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Task execution started successfully\",\"batch_id\":%d}", batchID)

	}

	return "{\"status\":\"error\",\"message\":\"Unknown function name\"}"
}
