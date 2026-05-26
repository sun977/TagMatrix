// ============================================================================
//  ___________              _____          __         .__
//  \__    ___/____    ____ /     \ _____ _/  |________|__|__  ___
//    |    |  \__  \  / ___/  \ /  \\__  \\   __\_  __ \  \  \/  /
//    |    |   / __ \/ /_/  >  Y    \/ __ \|  |  |  | \/  |>    <
//    |____|  (____  /\___  /____|__  (____  /__|  |__|  |__/__/\_ \
//                 \//_____/        \/     \/                     \/
// ============================================================================
// ⚡️ TagMatrix :: AI Copilot Dispatch Engine
//
// 👤 SYSTEM_ARCHITECT : sun977 (SunHaobo)
// 🌐 GITHUB_REF       : https://github.com/sun977
// 📧 CONTACT_MAIL     : jiuwei977@foxmail.com
// 📅 INIT_YEAR        : 2026
//
// 📝 [DESC] 负责与大语言模型 (LLM) 进行流式交互、管理多代理会话，并将系统上下文环境无缝注入 AI 决策链路。
//
// 💡 "A somewhat obsessive developer in cybersecurity & AI scenarios."
// ============================================================================

package aiengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/service/network"
	"TagMatrix/internal/service/taglogic"

	"github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

var (
	aiSem       *semaphore.Weighted
	aiSemMutex  sync.Mutex
	aiSemWeight int64
)

// 当系统偶尔碰到多条记录同时需要“深度平局 AI 裁决”时，
// 那 50 个满负荷运行的 Worker 在发起外部 AI 询问时，会自动挂起排队获取那 5 个网络请求名额，
// 既保证了本地吞吐最大化，又完美的防止了大模型 API 的 429 限流报错。完全拦截
func getAISemaphore() *semaphore.Weighted {
	cfg := config.GetConfig()
	weight := int64(cfg.Adv.Concurrency)
	if weight <= 0 {
		weight = 5
	}

	aiSemMutex.Lock()
	defer aiSemMutex.Unlock()

	if aiSem == nil || aiSemWeight != weight {
		aiSem = semaphore.NewWeighted(weight)
		aiSemWeight = weight
	}
	return aiSem
}

// AIEngineService 处理与 AI 相关的业务逻辑
type AIEngineService struct {
	db           *gorm.DB
	proxyService *network.ProxyService
	tagLogic     *taglogic.TagLogicService
}

// NewAIEngineService 创建 AIEngineService 实例
func NewAIEngineService() *AIEngineService {
	return &AIEngineService{
		db:           model.DB,
		proxyService: network.NewProxyService(), // 引入网络代理服务
		tagLogic:     taglogic.NewTagLogicService(),
	}
}

// cleanBaseURL 清理 baseURL 确保以 /chat/completions 结尾
func cleanBaseURL(url string) string {
	url = strings.TrimSpace(url)
	if strings.HasSuffix(url, "/chat/completions") {
		return strings.TrimSuffix(url, "/chat/completions")
	}
	return url
}

// getClient 动态获取最新的 OpenAI Client 实例
func (s *AIEngineService) getClient() (*openai.Client, string) {
	cfg := config.GetConfig().AI

	openAIConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		openAIConfig.BaseURL = cleanBaseURL(cfg.BaseURL)
	}
	// 使用代理
	openAIConfig.HTTPClient = s.proxyService.GetHTTPClient()

	modelName := cfg.Model
	if modelName == "" {
		modelName = openai.GPT4oMini
	}

	return openai.NewClientWithConfig(openAIConfig), modelName
}

// getSchema 获取当前 SQLite 数据库的核心表结构 (DDL)
func (s *AIEngineService) getSchema() (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// 核心业务表
	tables := []string{
		"raw_data_records", // 用户原始数据表
		"sys_tags",
		"sys_match_rules",
		"tag_task_batches",
		"tag_task_logs",
		"sys_entity_tags",
		"sys_sql_templates",
	}

	var schemaBuilder strings.Builder
	schemaBuilder.WriteString("当前 SQLite 数据库包含以下核心表结构：\n\n")

	for _, tableName := range tables {
		var createSQL string
		// SQLite 特有的获取建表语句的方法
		err := s.db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL).Error
		if err != nil {
			return "", fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
		}
		schemaBuilder.WriteString(fmt.Sprintf("-- Table: %s\n", tableName))
		schemaBuilder.WriteString(createSQL)
		schemaBuilder.WriteString(";\n\n")
	}

	return schemaBuilder.String(), nil
}

// ChatWithAI 发送消息给 AI 并获取回复。
// 自动注入 Schema 上下文，使 AI 能够回答关于数据查询或 SQL 生成的问题。
func (s *AIEngineService) ChatWithAI(ctx context.Context, message string) (string, error) {
	client, modelName := s.getClient()

	schema, err := s.getSchema()
	if err != nil {
		// 如果获取 schema 失败，依然可以进行普通对话，但不携带 schema
		schema = "无法获取数据库结构。"
	}

	cfg := config.GetConfig().AI
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `你是TagMatrix系统的全局智能助手，精通数据处理、标签规则配置和SQLite编写。

TagMatrix操作指南：
1.数据管理与SQL控制台:
底层使用SQLite数据库。原始导入数据在raw_data_records表的data字段(JSON格式文本)，查询时务必使用json_extract函数(或->/->>操作符)。根据用户需求生成准确的查询SQL。
2.标签规则引擎语法(AST JSON规范):
- 根节点必须且只能是逻辑节点：{"and": [...]}、{"or": [...]} 或 {"evaluate_all": [...]}。绝对不能直接以条件节点(如 {"field": ...})作为根节点！如果只有一个条件，也必须用 {"and": [条件节点]} 包裹。
- 条件节点(叶子节点)放在逻辑节点的数组中，必须包含 field(待匹配字段)、operator(操作符)、value(目标值)，可选 "ignore_case": true，可选附加动作 "action": "row_inc" 或 "global_inc"（用于频次统计）。
- 支持的操作符: equals, not_equals, contains, not_contains, starts_with, ends_with, greater_than, less_than, greater_than_or_equal, less_than_or_equal, in (此时value必须为数组), not_in, is_null, is_not_null, regex, like, exists, cidr, list_contains。
- 示例 (正确)：{"and": [{"field": "message", "operator": "contains", "value": "质量", "action": "row_inc"}]}
- 示例 (错误)：{"field": "message", "operator": "contains", "value": "质量"} (错误原因：最外层缺少 and/or 逻辑节点包裹)
3.页面上下文感知:
若问题带有指代词(如"这个页面")，请结合系统注入的当前页面环境信息解答；若提问显然与当前页面无关，请直接忽略上下文提示。

回答原则：
1.直入主题：先给代码/规则结果，再解析，不长篇大论。
2.格式规范：SQL/正则/JSON/代码等必用Markdown代码块包裹。涉及界面操作用有序列表。`
	}

	// 将 Schema 附加到系统提示词后
	fullSystemPrompt := systemPrompt + "\n\n以下是当前系统的数据库结构信息：\n" + schema

	// 构造AI请求(OpenAI协议)
	req := openai.ChatCompletionRequest{
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: fullSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: message,
			},
		},
	}

	// 发送请求,拿到AI响应
	var resp openai.ChatCompletionResponse

	// 控制 AI 并发请求
	sem := getAISemaphore()
	if acquireErr := sem.Acquire(ctx, 1); acquireErr != nil {
		return "", fmt.Errorf("failed to acquire AI semaphore: %w", acquireErr)
	}
	defer sem.Release(1)

	retries := config.GetConfig().Adv.Retries
	if retries < 0 {
		retries = 0
	}
	for i := 0; i <= retries; i++ {
		resp, err = client.CreateChatCompletion(ctx, req)
		if err == nil {
			break
		}
		if i < retries {
			// 可以选择加上延时等待
			// time.Sleep(time.Second)
			continue
		}
	}
	if err != nil {
		return "", fmt.Errorf("AI response error after %d retries: %w", retries, err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from AI")
}

// ChatWithAIStream 发送消息给 AI 并获取流式回复，通过 Wails events 发送至前端。
func (s *AIEngineService) ChatWithAIStream(ctx context.Context, reqId string, message string) error {
	client, modelName := s.getClient()

	schema, err := s.getSchema()
	if err != nil {
		schema = "无法获取数据库结构。"
	}

	cfg := config.GetConfig().AI
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = `你是TagMatrix系统的全局智能助手，精通数据处理、标签规则配置和SQLite编写。

TagMatrix操作指南：
1.数据管理与SQL控制台:
底层使用SQLite数据库。原始导入数据在raw_data_records表的data字段(JSON格式文本)，查询时务必使用json_extract函数(或->/->>操作符)。根据用户需求生成准确的查询SQL。
2.标签规则引擎语法(AST JSON规范):
- 根节点必须且只能是逻辑节点：{"and": [...]}、{"or": [...]} 或 {"evaluate_all": [...]}。绝对不能直接以条件节点(如 {"field": ...})作为根节点！如果只有一个条件，也必须用 {"and": [条件节点]} 包裹。
- 条件节点(叶子节点)放在逻辑节点的数组中，必须包含 field(待匹配字段)、operator(操作符)、value(目标值)，可选 "ignore_case": true，可选附加动作 "action": "row_inc" 或 "global_inc"（用于频次统计）。
- 支持的操作符: equals, not_equals, contains, not_contains, starts_with, ends_with, greater_than, less_than, greater_than_or_equal, less_than_or_equal, in (此时value必须为数组), not_in, is_null, is_not_null, regex, like, exists, cidr, list_contains。
- 示例 (正确)：{"and": [{"field": "message", "operator": "contains", "value": "质量", "action": "row_inc"}]}
- 示例 (错误)：{"field": "message", "operator": "contains", "value": "质量"} (错误原因：最外层缺少 and/or 逻辑节点包裹)
3.页面上下文感知:
若问题带有指代词(如"这个页面")，请结合系统注入的当前页面环境信息解答；若提问显然与当前页面无关，请直接忽略上下文提示。

回答原则：
1.直入主题：先给代码/规则结果，再解析，不长篇大论。
2.格式规范：SQL/正则/JSON/代码等必用Markdown代码块包裹。涉及界面操作用有序列表。`
	}

	actionInstruction := "\n\n[系统交互指令]\n1. 数据查询操作：如果请求编写SQL查询，你必须先使用 Markdown 代码块 ```sql ... ``` 展示 SQL 语句给用户看，然后再在回答末尾附上动作标签：\n<action type=\"execute_sql\" query=\"YOUR_SQL_HERE\" label=\"一键去 SQL 控制台执行\" />\n前端将渲染为按钮。*(注意：Action属性用双引号。SQL内字符串字面量用单引号避免冲突。罕见双引号用HTML实体&quot;转义。换行保留)*\n" +
		"2. 敏感/高危拦截：对于不可逆的删除动作（例如目前支持的“删除标签”操作），请绝对不要尝试直接调用内部工具，而是统一输出交互标签供前端渲染二次确认按钮，例如：\n<action type=\"delete_tag\" query=\"/目标/标签/路径/\" label=\"确认删除该标签\" />"

	// 解析传入的 message (可能是一个包含 is_agent 的对象，或者是单纯的消息数组)
	type ChatMsgJSON struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type PayloadJSON struct {
		Messages []ChatMsgJSON `json:"messages"`
		IsAgent  bool          `json:"is_agent"`
	}

	var incomingMsgs []ChatMsgJSON
	var isAgentMode bool
	var tagTreeContext string

	if strings.HasPrefix(strings.TrimSpace(message), "{") {
		var payload PayloadJSON
		if err := json.Unmarshal([]byte(message), &payload); err == nil {
			incomingMsgs = payload.Messages
			isAgentMode = payload.IsAgent
		} else {
			incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
		}
	} else if strings.HasPrefix(strings.TrimSpace(message), "[") {
		if err := json.Unmarshal([]byte(message), &incomingMsgs); err != nil {
			// 解析失败，作为一个普通字符串处理
			incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
		}
	} else {
		// 纯文本，作为单条 user 消息
		incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
	}

	var modeInstruction string
	if isAgentMode {
		modeInstruction = "\n\n【运行模式提醒】当前处于 Agent (后台自驱) 模式。系统已开放相关 tools 供你调度。针对用户的指令需求，你可以并且应该直接调用对应的 tools 工具来完成系统配置或变更操作（包括但不限于当前的标签与规则管理，以及未来扩充的其它业务逻辑）。切勿仅仅回复文字让用户手动去操作。"
		// 获取当前系统的标签树作为上下文
		treeNodes, _ := s.tagLogic.GetTagTree()
		treeBytes, _ := json.MarshalIndent(treeNodes, "", "  ")

		// 获取数据集列表上下文
		var dsInfos []string
		var datasets []model.SysDataset
		if s.db != nil {
			s.db.Find(&datasets)
			for _, d := range datasets {
				dsInfos = append(dsInfos, fmt.Sprintf("- 数据集名: %s (字段/SchemaKeys: %s)", d.Name, d.SchemaKeys))
			}
		}
		dsContext := "当前系统中存在的数据集列表及可用字段：\n" + strings.Join(dsInfos, "\n")

		tagTreeContext = "\n\n【系统当前运行时上下文】\n1. " + dsContext + "\n2. 当前已有标签目录树结构(仅供参考)：\n" + string(treeBytes)
	} else {
		modeInstruction = "\n\n【运行模式提醒】当前处于 Ask (纯问答辅助) 模式！在该模式下你**被剥夺了任何底层工具的调用权限**。如果用户要求你直接执行系统级数据变更操作（如数据的创建、修改、移动、删除等业务能力），请委婉说明当前问答模式的限制，并仅提供操作路径和原理解释。切勿假装执行成功，绝对不要模仿输出伪造的执行过程提示！"
	}

	fullSystemPrompt := systemPrompt + actionInstruction + modeInstruction + "\n\n以下是当前系统的数据库结构信息：\n" + schema + tagTreeContext

	// 构造发送给大模型的 Messages 数组
	var messages []openai.ChatCompletionMessage
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: fullSystemPrompt,
	})

	for _, m := range incomingMsgs {
		role := openai.ChatMessageRoleUser
		if m.Role == "ai" || m.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   true,
	}

	if isAgentMode {
		req.Tools = []openai.Tool{
			{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "create_system_tag",
					Description: "在系统中创建新标签",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"tag_name": map[string]interface{}{
								"type":        "string",
								"description": "要创建的标签名称，例如 '日志'",
							},
							"parent_path": map[string]interface{}{
								"type":        "string",
								"description": "父级标签的完整路径，以 / 开头。如果在根目录下创建，则传 '/'",
							},
							"description": map[string]interface{}{
								"type":        "string",
								"description": "标签描述",
							},
						},
						"required": []string{"tag_name", "parent_path"},
					},
				},
			},
			{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "update_system_tag",
					Description: "修改标签的名称、颜色或描述",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"target_tag_path": map[string]interface{}{
								"type":        "string",
								"description": "要修改的目标标签当前完整路径，例如 '/系统/日志'",
							},
							"new_name": map[string]interface{}{
								"type":        "string",
								"description": "可选。新的标签名称",
							},
							"new_color": map[string]interface{}{
								"type":        "string",
								"description": "可选。新的标签颜色(HEX，如 #ff0000)",
							},
							"new_description": map[string]interface{}{
								"type":        "string",
								"description": "可选。新的标签描述",
							},
						},
						"required": []string{"target_tag_path"},
					},
				},
			},
			{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "move_system_tag",
					Description: "移动标签到新的父级路径下",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"target_tag_path": map[string]interface{}{
								"type":        "string",
								"description": "要移动的目标标签当前完整路径，例如 '/安全/QA/'",
							},
							"new_parent_path": map[string]interface{}{
								"type":        "string",
								"description": "新的父级标签完整路径（仅父节点路径，切勿包含当前标签的名称！）。例如移动到 '/测试/' 下，就传 '/测试/'。若移动到根目录则传 '/'",
							},
						},
						"required": []string{"target_tag_path", "new_parent_path"},
					},
				},
			},
			{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "create_tag_rule",
					Description: "为目标标签挂载匹配/提取规则",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"target_tag_path": map[string]interface{}{
								"type":        "string",
								"description": "目标标签完整路径，例如 '/系统/日志'",
							},
							"dataset_name": map[string]interface{}{
								"type":        "string",
								"description": "要挂载规则的数据集名称",
							},
							"condition_json": map[string]interface{}{
								"type":        "string",
								"description": "符合 TagMatrix 规则引擎规范的 JSON 字符串。注意：1. 最外层必须是逻辑节点(and/or/evaluate_all)；2. 若用户要求配置动作(如行级计数或全局计数)，请务必将 'action' 属性写在 JSON 的条件节点内部，不要漏掉！当前支持的动作值有：'row_inc' (行级计数) 和 'global_inc' (全局计数)。",
							},
						},
						"required": []string{"target_tag_path", "dataset_name", "condition_json"},
					},
				},
			},
			{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "update_tag_rule",
					Description: "修改标签在指定数据集下的匹配规则",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"target_tag_path": map[string]interface{}{
								"type":        "string",
								"description": "目标标签完整路径，例如 '/系统/日志'",
							},
							"dataset_name": map[string]interface{}{
								"type":        "string",
								"description": "规则所在的数据集名称",
							},
							"new_condition_json": map[string]interface{}{
								"type":        "string",
								"description": "符合 TagMatrix 规则引擎规范的新的 JSON 字符串（可选，如果不修改则不传。注意：最外层必须是 logic 节点！如需动作请写在JSON节点内，支持 'row_inc' 和 'global_inc'）",
							},
						},
						"required": []string{"target_tag_path", "dataset_name"},
					},
				},
			},
			{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        "delete_tag_rule",
					Description: "删除标签在指定数据集下的匹配规则",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"target_tag_path": map[string]interface{}{
								"type":        "string",
								"description": "目标标签完整路径，例如 '/系统/日志'",
							},
							"dataset_name": map[string]interface{}{
								"type":        "string",
								"description": "规则所在的数据集名称",
							},
						},
						"required": []string{"target_tag_path", "dataset_name"},
					},
				},
			},
		}
	}

	for {
		// 控制 AI 并发请求
		sem := getAISemaphore()
		if acquireErr := sem.Acquire(ctx, 1); acquireErr != nil {
			return fmt.Errorf("failed to acquire AI semaphore: %w", acquireErr)
		}

		var stream *openai.ChatCompletionStream
		retries := config.GetConfig().Adv.Retries
		if retries < 0 {
			retries = 0
		}
		for i := 0; i <= retries; i++ {
			stream, err = client.CreateChatCompletionStream(ctx, req)
			if err == nil {
				break
			}
			if i < retries {
				continue
			}
		}
		if err != nil {
			sem.Release(1)
			return fmt.Errorf("AI response error after %d retries: %w", retries, err)
		}

		var collectedToolCalls []openai.ToolCall
		var assistantMessage string

		for {
			response, streamErr := stream.Recv()
			if streamErr != nil {
				if errors.Is(streamErr, io.EOF) {
					break
				}
				stream.Close()
				sem.Release(1)
				if reqId != "" {
					runtime.EventsEmit(ctx, "ai_chat_error_"+reqId, streamErr.Error())
				} else {
					runtime.EventsEmit(ctx, "ai_chat_error", streamErr.Error())
				}
				return streamErr
			}

			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta

				// Handle Tool Calls in stream
				for _, tc := range delta.ToolCalls {
					idx := 0
					if tc.Index != nil {
						idx = *tc.Index
					}
					for len(collectedToolCalls) <= idx {
						collectedToolCalls = append(collectedToolCalls, openai.ToolCall{
							Type: openai.ToolTypeFunction,
						})
					}
					if tc.ID != "" {
						collectedToolCalls[idx].ID = tc.ID
					}
					if tc.Function.Name != "" {
						collectedToolCalls[idx].Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						collectedToolCalls[idx].Function.Arguments += tc.Function.Arguments
					}
				}

				content := delta.Content
				if content != "" {
					assistantMessage += content
					if reqId != "" {
						runtime.EventsEmit(ctx, "ai_chat_chunk_"+reqId, content)
					} else {
						runtime.EventsEmit(ctx, "ai_chat_chunk", content)
					}
				}
			}
		}
		stream.Close()
		sem.Release(1)

		if len(collectedToolCalls) > 0 {
			// Append assistant's tool call message
			messages = append(messages, openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   assistantMessage,
				ToolCalls: collectedToolCalls,
			})

			// Execute tools and append results
			for _, tc := range collectedToolCalls {
				// Emit an event to UI that AI is executing a tool
				toolMsg := fmt.Sprintf("\n> 🤖 [Agent] 正在执行操作: `%s`...\n", tc.Function.Name)
				if reqId != "" {
					runtime.EventsEmit(ctx, "ai_chat_chunk_"+reqId, toolMsg)
				} else {
					runtime.EventsEmit(ctx, "ai_chat_chunk", toolMsg)
				}

				result := s.executeAITool(ctx, tc)
				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
			// Update request messages and loop again
			req.Messages = messages
			continue
		} else {
			// No more tool calls, we are done
			if reqId != "" {
				runtime.EventsEmit(ctx, "ai_chat_end_"+reqId)
			} else {
				runtime.EventsEmit(ctx, "ai_chat_end")
			}
			break
		}
	}
	return nil
}

// TestConnection 测试用户提供的 AI 连通性
func (s *AIEngineService) TestConnection(ctx context.Context, apiKey, baseUrl, modelName string) error {
	openAIConfig := openai.DefaultConfig(apiKey)
	if baseUrl != "" {
		openAIConfig.BaseURL = cleanBaseURL(baseUrl)
	}
	// 使用代理
	openAIConfig.HTTPClient = s.proxyService.GetHTTPClient()

	if modelName == "" {
		modelName = openai.GPT4oMini
	}

	client := openai.NewClientWithConfig(openAIConfig)
	req := openai.ChatCompletionRequest{
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Ping",
			},
		},
		MaxTokens: 5,
	}

	// 控制 AI 并发请求
	sem := getAISemaphore()
	if acquireErr := sem.Acquire(ctx, 1); acquireErr != nil {
		return fmt.Errorf("failed to acquire AI semaphore: %w", acquireErr)
	}
	defer sem.Release(1)

	var err error
	retries := config.GetConfig().Adv.Retries
	if retries < 0 {
		retries = 0
	}
	for i := 0; i <= retries; i++ {
		_, err = client.CreateChatCompletion(ctx, req)
		if err == nil {
			break
		}
	}
	return err
}
