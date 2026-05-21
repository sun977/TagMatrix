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

	"TagMatrix/internal/config"
	"TagMatrix/internal/model"
	"TagMatrix/internal/service/network"

	"github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// AIEngineService 处理与 AI 相关的业务逻辑
type AIEngineService struct {
	db           *gorm.DB
	proxyService *network.ProxyService
}

// NewAIEngineService 创建 AIEngineService 实例
func NewAIEngineService() *AIEngineService {
	return &AIEngineService{
		db:           model.DB,
		proxyService: network.NewProxyService(), // 引入网络代理服务
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
2.标签规则引擎语法:
用于特征提取或打标，生成JSON规范规则。支持嵌套，逻辑节点{"and":[...]}、{"or":[...]}或非短路节点{"evaluate_all":[...]}。条件节点须含field(待匹配字段)、operator(操作符)、value(目标值)，可选"ignore_case":true。
支持的操作符(必须严格遵守):equals,not_equals,contains,not_contains,starts_with,ends_with,greater_than,less_than,greater_than_or_equal,less_than_or_equal,in(value为数组),not_in,is_null,is_not_null,regex,like,exists,cidr,list_contains。
新增频次与副作用算子:count_contains(统计子串出现次数),count_regex(统计正则命中次数),row_inc(当前行计数+N),global_inc(全局计数+N)。
示例:用户需求设备为honeypot且os含linux，规则为:{"and":[{"field":"device_type","operator":"equals","value":"honeypot"},{"field":"os","operator":"contains","value":"linux"}]}
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
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("AI response error: %w", err)
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
2.标签规则引擎语法:
用于特征提取或打标，生成JSON规范规则。支持嵌套，逻辑节点{"and":[...]}、{"or":[...]}或非短路节点{"evaluate_all":[...]}。条件节点须含field(待匹配字段)、operator(操作符)、value(目标值)，可选"ignore_case":true。
支持的操作符(必须严格遵守):equals,not_equals,contains,not_contains,starts_with,ends_with,greater_than,less_than,greater_than_or_equal,less_than_or_equal,in(value为数组),not_in,is_null,is_not_null,regex,like,exists,cidr,list_contains。
新增频次与副作用算子:count_contains(统计子串出现次数),count_regex(统计正则命中次数),row_inc(当前行计数+N),global_inc(全局计数+N)。
示例:用户需求设备为honeypot且os含linux，规则为:{"and":[{"field":"device_type","operator":"equals","value":"honeypot"},{"field":"os","operator":"contains","value":"linux"}]}
3.页面上下文感知:
若问题带有指代词(如"这个页面")，请结合系统注入的当前页面环境信息解答；若提问显然与当前页面无关，请直接忽略上下文提示。

回答原则：
1.直入主题：先给代码/规则结果，再解析，不长篇大论。
2.格式规范：SQL/正则/JSON/代码等必用Markdown代码块包裹。涉及界面操作用有序列表。`
	}

	actionInstruction := "\n\n[系统交互指令]\n如果请求编写SQL，除Markdown代码块外，请务必在末尾输出动作标签：\n<action type=\"execute_sql\" query=\"YOUR_SQL_HERE\" label=\"一键去 SQL 控制台执行\" />\n前端将渲染为按钮。*(注意：Action属性用双引号。SQL内字符串字面量用单引号避免冲突。罕见双引号用HTML实体&quot;转义。换行保留)*"

	fullSystemPrompt := systemPrompt + actionInstruction + "\n\n以下是当前系统的数据库结构信息：\n" + schema

	// 解析传入的 message (可能是 JSON 数组)
	type ChatMsgJSON struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var incomingMsgs []ChatMsgJSON

	if strings.HasPrefix(strings.TrimSpace(message), "[") {
		if err := json.Unmarshal([]byte(message), &incomingMsgs); err != nil {
			// 解析失败，作为一个普通字符串处理
			incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
		}
	} else {
		// 纯文本，作为单条 user 消息
		incomingMsgs = []ChatMsgJSON{{Role: "user", Content: message}}
	}

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

	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("AI response error: %w", err)
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if reqId != "" {
					runtime.EventsEmit(ctx, "ai_chat_end_"+reqId)
				} else {
					runtime.EventsEmit(ctx, "ai_chat_end")
				}
				break
			}
			if reqId != "" {
				runtime.EventsEmit(ctx, "ai_chat_error_"+reqId, err.Error())
			} else {
				runtime.EventsEmit(ctx, "ai_chat_error", err.Error())
			}
			return err
		}
		if len(response.Choices) > 0 {
			content := response.Choices[0].Delta.Content
			if content != "" {
				if reqId != "" {
					runtime.EventsEmit(ctx, "ai_chat_chunk_"+reqId, content)
				} else {
					runtime.EventsEmit(ctx, "ai_chat_chunk", content)
				}
			}
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

	_, err := client.CreateChatCompletion(ctx, req)
	return err
}
