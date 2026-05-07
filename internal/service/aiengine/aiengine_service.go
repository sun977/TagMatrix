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
		"raw_data_records",
		"sys_tags",
		"sys_match_rules",
		"tag_task_batches",
		"tag_task_logs",
		"sys_entity_tags",
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
		systemPrompt = "你是一个数据分析助手。"
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
func (s *AIEngineService) ChatWithAIStream(ctx context.Context, message string) error {
	client, modelName := s.getClient()

	schema, err := s.getSchema()
	if err != nil {
		schema = "无法获取数据库结构。"
	}

	cfg := config.GetConfig().AI
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一个数据分析助手。"
	}

	actionInstruction := "\n\n[系统交互指令]\n如果用户的意图是请求你编写或生成一段 SQL，除了给出解释和 Markdown 代码块之外，请务必在你的回复末尾输出一个动作标签，格式严格如下：\n<action type=\"execute_sql\" query=\"YOUR_SQL_HERE\" label=\"一键去 SQL 控制台执行\" />\n这将会被前端解析并渲染为一个交互按钮。请将 query 属性中的双引号转义为实体或直接使用单引号包裹字符串。"

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
				runtime.EventsEmit(ctx, "ai_chat_end")
				break
			}
			runtime.EventsEmit(ctx, "ai_chat_error", err.Error())
			return err
		}
		if len(response.Choices) > 0 {
			content := response.Choices[0].Delta.Content
			if content != "" {
				runtime.EventsEmit(ctx, "ai_chat_chunk", content)
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
