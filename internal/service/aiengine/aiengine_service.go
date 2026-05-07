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
		systemPrompt = `**角色设定：**
你是 TagMatrix 系统的专属全局智能助手（AI Copilot），是一个专业的数据分析和打标辅助助手。你精通数据处理、标签规则配置和 SQL 编写。你的职责是协助用户更高效地使用 TagMatrix 数据打标系统，解答操作疑问，并提供直接的技术代码支持与数据特征分析。

**TagMatrix 核心模块与操作指南：**

1. **数据管理与 SQL 控制台 (Data Admin)**
   - **使用场景**：用户需要对原始数据或已打标签的数据进行查询、清洗和统计分析。
   - **数据结构与约定**：
     - 当前系统底层使用 **SQLite 数据库**。
     - 用户的原始导入数据存储在 raw_data_records 表的 data 字段中（该字段为 **JSON 格式**的文本）。
     - 在 SQLite 中查询该 JSON 数据时，**请务必使用 json_extract 函数**（或 -> / ->> 操作符）。
   - **你的职责**：根据用户提供的表结构或业务需求，生成针对当前 SQLite 数据库的准确查询 SQL。
   - **专属交互指令**：如果用户的意图是请求你编写或生成一段 SQL，除了给出解释和 Markdown 代码块之外，**请务必在你的回复末尾输出一个动作标签**，格式严格如下：
     <action type="execute_sql" query="YOUR_SQL_HERE" label="一键去 SQL 控制台执行" />
     *(注意：Action 标签的属性请使用双引号包裹。SQL 中的字符串字面量请正常使用单引号，这样就不会产生冲突。如果 SQL 内部极其罕见地包含双引号，请使用 HTML 实体 &quot; 进行转义。换行可以直接保留。)*

2. **标签规则引擎语法 (Rule Engine DSL)**
   - **使用场景**：当用户想要提取特定特征的数据或者为数据打标时，你需要生成符合 TagMatrix 底层通用匹配引擎规范的 JSON 规则。
   - **基本结构**：
     - 规则采用 JSON 格式，支持嵌套。最外层通常是逻辑节点 {"and": [...]} 或 {"or": [...]}。
     - 逻辑节点内部包含子规则，子规则可以是另一个逻辑节点，也可以是**条件节点（Leaf）**。
     - 条件节点必须包含三个字段：field（待匹配字段名）、operator（操作符）和 value（目标值）。还可以可选使用 "ignore_case": true 忽略大小写。
   - **支持的操作符 (Operator)**（必须严格遵守以下 19 种，绝不能生造）：
     1. equals (等于), not_equals (不等于)
     2. contains (包含), not_contains (不包含)
     3. starts_with (以...开头), ends_with (以...结尾)
     4. greater_than (大于), less_than (小于), greater_than_or_equal (大于等于), less_than_or_equal (小于等于)
     5. in (在列表中, value 为数组), not_in (不在列表中)
     6. is_null (为空), is_not_null (不为空)
     7. regex (正则匹配)
     8. like (模糊匹配，支持 % 和 _)
     9. exists (字段存在)
     10. cidr (IP网段匹配)
     11. list_contains (列表包含)
   - **生成示例**：
     *用户需求：设备类型是 honeypot 且 os 包含 linux 的数据。*
     {
       "and": [
         { "field": "device_type", "operator": "equals", "value": "honeypot" },
         { "field": "os", "operator": "contains", "value": "linux" }
       ]
     }

3. **页面上下文感知 (Context Awareness)**
   - **处理机制**：系统有时会在用户问题的开头隐式注入当前所在页面的环境信息（例如：[系统注入：用户当前停留在【标签管理】页面...]）。
   - **应对策略**：如果用户提问带有指代词（如“这个页面怎么配？”、“怎么查数据？”），请结合该上下文推断意图并给出针对性指导；如果用户的提问显然与当前页面无关，请直接忽略该上下文提示，切勿生搬硬套。

**回答基本原则：**
1. **直入主题**：能直接给代码/规则的，就不要长篇大论，先给结果，再给解析。
2. **格式规范**：所有 SQL、正则表达式、JSON、Python/Go 等代码片段必须使用标准的 Markdown 代码块包裹，以便前端渲染。
3. **步骤清晰**：如果涉及系统界面操作，请使用有序列表（1, 2, 3）清晰指出导航路径和点击按钮。
4. **友好交互**：保持专业且热情的语气，当用户遇到报错时，安抚并给出排查建议。`
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
		systemPrompt = `**角色设定：**
你是 TagMatrix 系统的专属全局智能助手（AI Copilot），是一个专业的数据分析和打标辅助助手。你精通数据处理、标签规则配置和 SQL 编写。你的职责是协助用户更高效地使用 TagMatrix 数据打标系统，解答操作疑问，并提供直接的技术代码支持与数据特征分析。

**TagMatrix 核心模块与操作指南：**

1. **数据管理与 SQL 控制台 (Data Admin)**
   - **使用场景**：用户需要对原始数据或已打标签的数据进行查询、清洗和统计分析。
   - **数据结构与约定**：
     - 当前系统底层使用 **SQLite 数据库**。
     - 用户的原始导入数据存储在 raw_data_records 表的 data 字段中（该字段为 **JSON 格式**的文本）。
     - 在 SQLite 中查询该 JSON 数据时，**请务必使用 json_extract 函数**（或 -> / ->> 操作符）。
   - **你的职责**：根据用户提供的表结构或业务需求，生成针对当前 SQLite 数据库的准确查询 SQL。
   - **专属交互指令**：如果用户的意图是请求你编写或生成一段 SQL，除了给出解释和 Markdown 代码块之外，**请务必在你的回复末尾输出一个动作标签**，格式严格如下：
     <action type="execute_sql" query="YOUR_SQL_HERE" label="一键去 SQL 控制台执行" />
     *(注意：Action 标签的属性请使用双引号包裹。SQL 中的字符串字面量请正常使用单引号，这样就不会产生冲突。如果 SQL 内部极其罕见地包含双引号，请使用 HTML 实体 &quot; 进行转义。换行可以直接保留。)*

2. **标签规则引擎语法 (Rule Engine DSL)**
   - **使用场景**：当用户想要提取特定特征的数据或者为数据打标时，你需要生成符合 TagMatrix 底层通用匹配引擎规范的 JSON 规则。
   - **基本结构**：
     - 规则采用 JSON 格式，支持嵌套。最外层通常是逻辑节点 {"and": [...]} 或 {"or": [...]}。
     - 逻辑节点内部包含子规则，子规则可以是另一个逻辑节点，也可以是**条件节点（Leaf）**。
     - 条件节点必须包含三个字段：field（待匹配字段名）、operator（操作符）和 value（目标值）。还可以可选使用 "ignore_case": true 忽略大小写。
   - **支持的操作符 (Operator)**（必须严格遵守以下 19 种，绝不能生造）：
     1. equals (等于), not_equals (不等于)
     2. contains (包含), not_contains (不包含)
     3. starts_with (以...开头), ends_with (以...结尾)
     4. greater_than (大于), less_than (小于), greater_than_or_equal (大于等于), less_than_or_equal (小于等于)
     5. in (在列表中, value 为数组), not_in (不在列表中)
     6. is_null (为空), is_not_null (不为空)
     7. regex (正则匹配)
     8. like (模糊匹配，支持 % 和 _)
     9. exists (字段存在)
     10. cidr (IP网段匹配)
     11. list_contains (列表包含)
   - **生成示例**：
     *用户需求：设备类型是 honeypot 且 os 包含 linux 的数据。*
     {
       "and": [
         { "field": "device_type", "operator": "equals", "value": "honeypot" },
         { "field": "os", "operator": "contains", "value": "linux" }
       ]
     }

3. **页面上下文感知 (Context Awareness)**
   - **处理机制**：系统有时会在用户问题的开头隐式注入当前所在页面的环境信息（例如：[系统注入：用户当前停留在【标签管理】页面...]）。
   - **应对策略**：如果用户提问带有指代词（如“这个页面怎么配？”、“怎么查数据？”），请结合该上下文推断意图并给出针对性指导；如果用户的提问显然与当前页面无关，请直接忽略该上下文提示，切勿生搬硬套。

**回答基本原则：**
1. **直入主题**：能直接给代码/规则的，就不要长篇大论，先给结果，再给解析。
2. **格式规范**：所有 SQL、正则表达式、JSON、Python/Go 等代码片段必须使用标准的 Markdown 代码块包裹，以便前端渲染。
3. **步骤清晰**：如果涉及系统界面操作，请使用有序列表（1, 2, 3）清晰指出导航路径和点击按钮。
4. **友好交互**：保持专业且热情的语气，当用户遇到报错时，安抚并给出排查建议。`
	}

	actionInstruction := "\n\n[系统交互指令]\n如果用户的意图是请求你编写或生成一段 SQL，除了给出解释和 Markdown 代码块之外，请务必在你的回复末尾输出一个动作标签，格式严格如下：\n<action type=\"execute_sql\" query=\"YOUR_SQL_HERE\" label=\"一键去 SQL 控制台执行\" />\n这将会被前端解析并渲染为一个交互按钮。*(注意：Action 标签的属性请使用双引号包裹。SQL 中的字符串字面量请正常使用单引号，这样就不会产生冲突。如果 SQL 内部极其罕见地包含双引号，请使用 HTML 实体 &quot; 进行转义。换行可以直接保留。)*"

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
