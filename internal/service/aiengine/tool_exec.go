package aiengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
				return fmt.Sprintf("{\"status\":\"error\",\"message\":\"new parent path %s not found.\"}", args.NewParentPath)
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
			ConditionJSON string `json:"condition_json"`
			IsCountMode   bool   `json:"is_count_mode"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		tag, err := s.tagLogic.GetTagByPath(args.TargetTagPath)
		if err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"target tag path %s not found.\"}", args.TargetTagPath)
		}

		rule := &model.SysMatchRule{
			TagID:     tag.ID,
			DatasetID: 0, // 默认0表示全部数据集（或需要AI指定）
			Name:      "AI 自动生成规则",
			RuleJSON:  args.ConditionJSON,
			Priority:  0, // 默认给0
		}

		if err := s.tagLogic.SaveRule(rule); err != nil {
			return fmt.Sprintf("{\"status\":\"error\",\"message\":\"%v\"}", err)
		}

		// 规则更新可以复用同一个事件或用专门的事件
		runtime.EventsEmit(ctx, "rule_list_updated")

		return fmt.Sprintf("{\"status\":\"success\",\"message\":\"Rule created successfully\",\"rule_id\":%d}", rule.ID)
	}

	return "{\"status\":\"error\",\"message\":\"Unknown function name\"}"
}
