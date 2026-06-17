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
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/template"

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/service/network"
	"TagMatrix/internal/service/taglogic"

	"github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

//go:embed prompts/base_prompt.tmpl
var BaseSystemPrompt string

//go:embed prompts/bottom_prompt.tmpl
var BottomSystemPrompt string

//go:embed prompts/mdct_arbiter_prompt.tmpl
var MDCTArbiterPromptTmpl string

var (
	aiSem       *semaphore.Weighted
	aiSemMutex  sync.Mutex
	aiSemWeight int64

	cancelFuncs      = make(map[string]context.CancelFunc)
	cancelFuncsMutex sync.Mutex
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
	db                 *gorm.DB
	proxyService       *network.ProxyService
	tagLogic           *taglogic.TagLogicService
	RunTaggingTaskFunc func(ctx context.Context, datasetID uint64, ruleIDs []uint64, tagMode string) (uint64, error)
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

// buildFullSystemPrompt 构建企业级三明治提示词
func buildFullSystemPrompt(schema, tagTreeContext, customPrompt string, isAgentMode bool) string {
	tmpl, err := template.New("base").Parse(BaseSystemPrompt)
	if err != nil {
		// Fallback
		return BaseSystemPrompt + "\n\n" + customPrompt + "\n\n" + BottomSystemPrompt
	}

	data := map[string]interface{}{
		"DBSchema":       schema,
		"TagTreeContext": tagTreeContext,
		"IsAgentMode":    isAgentMode,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		buf.WriteString(BaseSystemPrompt)
	}

	baseStr := buf.String()

	fullPrompt := baseStr + "\n\n<custom_prompt>\n" + customPrompt + "\n</custom_prompt>\n\n" + BottomSystemPrompt
	return fullPrompt
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
	
	// 构建完整的系统提示词
	fullSystemPrompt := buildFullSystemPrompt(schema, "无", cfg.CustomPrompt, false)

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

// getAgentTools 返回 AI Agent 可以调用的工具列表
func (s *AIEngineService) getAgentTools() []openai.Tool {
	return []openai.Tool{
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
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "execute_tagging_task",
				Description: "执行打标任务，对指定数据集按给定规则进行自动打标",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"dataset_id": map[string]interface{}{
							"type":        "integer",
							"description": "要执行打标任务的数据集ID",
						},
						"rule_ids": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "integer",
							},
							"description": "要执行的规则ID列表",
						},
						"tag_mode": map[string]interface{}{
							"type":        "string",
							"description": "打标模式，可选值为 'single' (单标签), 'multiple' (多标签), 'mixed' (混合模式)",
							"enum":        []string{"single", "multiple", "mixed"},
						},
					},
					"required": []string{"dataset_id", "rule_ids", "tag_mode"},
				},
			},
		},
	}
}

// parseIncomingMessage 解析前端发来的消息 Payload
type ChatMsgJSON struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PayloadJSON struct {
	Messages []ChatMsgJSON `json:"messages"`
	IsAgent  bool          `json:"is_agent"`
}

func parseIncomingMessage(message string) ([]ChatMsgJSON, bool) {
	var incomingMsgs []ChatMsgJSON
	var isAgentMode bool

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
			incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
		}
	} else {
		incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
	}

	return incomingMsgs, isAgentMode
}

// ChatWithAIStream 发送消息给 AI 并获取流式回复，通过 Wails events 发送至前端。
func (s *AIEngineService) ChatWithAIStream(ctx context.Context, reqId string, message string) error {
	ctx, cancel := context.WithCancel(ctx)
	
	cancelFuncsMutex.Lock()
	cancelFuncs[reqId] = cancel
	cancelFuncsMutex.Unlock()
	
	defer func() {
		cancelFuncsMutex.Lock()
		delete(cancelFuncs, reqId)
		cancelFuncsMutex.Unlock()
		cancel()
	}()

	client, modelName := s.getClient()

	schema, err := s.getSchema()
	if err != nil {
		schema = "无法获取数据库结构。"
	}

	cfg := config.GetConfig().AI

	// 解析传入的 message
	incomingMsgs, isAgentMode := parseIncomingMessage(message)

	var tagTreeContext string
	if isAgentMode {
		// 获取当前系统的标签树作为上下文
		treeNodes, _ := s.tagLogic.GetTagTree()
		treeBytes, _ := json.MarshalIndent(treeNodes, "", "  ")

		// 获取数据集列表上下文
		var dsInfos []string
		var datasets []model.SysDataset
		if s.db != nil {
			s.db.Find(&datasets)
			for _, d := range datasets {
				dsInfos = append(dsInfos, fmt.Sprintf("- 数据集ID: %d, 数据集名: %s (字段/SchemaKeys: %s)", d.ID, d.Name, d.SchemaKeys))
			}
		}
		dsContext := "当前系统中存在的数据集列表及可用字段：\n" + strings.Join(dsInfos, "\n")

		tagTreeContext = "1. " + dsContext + "\n2. 当前已有标签目录树结构(仅供参考)：\n" + string(treeBytes)
	} else {
		tagTreeContext = "无（当前处于问答模式，不需要参考标签树）"
	}

	fullSystemPrompt := buildFullSystemPrompt(schema, tagTreeContext, cfg.CustomPrompt, isAgentMode)

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
		req.Tools = s.getAgentTools()
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

				if errors.Is(ctx.Err(), context.Canceled) || strings.Contains(streamErr.Error(), "context canceled") {
					if reqId != "" {
						runtime.EventsEmit(ctx, "ai_chat_end_"+reqId)
					} else {
						runtime.EventsEmit(ctx, "ai_chat_end")
					}
					return nil
				}

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

		// 增加 Agent 模式判断
		if len(collectedToolCalls) > 0 && isAgentMode {
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
		} else if len(collectedToolCalls) > 0 && !isAgentMode {
			// 如果在非 Agent 模式下模型仍然输出了 ToolCall，直接终止循环避免继续执行和返回假结果
			if reqId != "" {
				runtime.EventsEmit(ctx, "ai_chat_end_"+reqId)
			} else {
				runtime.EventsEmit(ctx, "ai_chat_end")
			}
			break
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

// CancelAIChatStream 取消指定的流式对话
func (s *AIEngineService) CancelAIChatStream(reqId string) {
	cancelFuncsMutex.Lock()
	defer cancelFuncsMutex.Unlock()
	if cancel, ok := cancelFuncs[reqId]; ok {
		cancel()
		delete(cancelFuncs, reqId)
	}
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
