# AI 引擎 Prompt 模板管理说明

此目录 (`internal/service/aiengine/prompts/`) 用于存放 TagMatrix 系统 AI Copilot 的核心提示词模板。这些模板会在编译时通过 Go 的 `//go:embed` 机制嵌入到二进制文件中，保证了核心业务逻辑的安全性与分发的一致性。

## 为什么使用模板文件？

1. **安全性**：避免核心业务逻辑（如 AST 解析规则、数据库 Schema 解释）以明文配置的形式暴露给最终用户。
2. **可维护性**：使用 `.tmpl` 文件格式，能够获得主流 IDE 的语法高亮支持，无需在 Go 代码中进行痛苦的字符串拼接和换行符转义。
3. **动态组装**：系统采用了“三明治”架构（Base -> User Custom -> Bottom Constraints），支持根据不同会话动态注入上下文信息。

## 目录结构与模块说明

当前目录下包含以下两个核心模板文件：

### 1. `base_prompt.tmpl` (基础核心模板)
这是系统提示词的“上半部分”，定义了 AI 的核心认知、工作模式和业务知识体系。包含以下模块：
*   **`<role>` (角色)**：定义 AI 的身份和专业领域（如数据处理、风控专家）。
*   **`<context>` (上下文)**：包含占位符（如 `{{.DBSchema}}`），用于在请求发起前动态注入当前的数据库结构、页面环境和标签树。
*   **`<manual>` (操作手册)**：硬性规定系统的规则，包括 `SQL_Rules`（如何查询）和 `AST_Rules`（严格的 JSON 生成规范）。
*   **`<Interaction_Instructions>` (交互模式指令)**：使用 Go 的条件语法（`{{if .IsAgentMode}}`），根据系统当前的运行状态，决定是赋予模型 Agent 后台自驱能力，还是将其限制在普通的 Ask 问答模式。
*   **`<examples>` (示例库)**：提供正确（Positive）与错误（Negative）的 Few-Shot 示例，对齐模型的思维链路。

### 2. `bottom_prompt.tmpl` (防御约束底线模板)
这是系统提示词的“下半部分”，放置在用户自定义提示词（CustomPrompt）之后。它利用大模型对 prompt 末尾记忆最深的“近因效应”，充当系统的安全拦截器与输出规范器。包含以下模块：
*   **`<Frontend_Action_Rules>` (前端动作指令)**：强制规定如果触发了 SQL 查询或高危删除，必须按特定格式输出 `<action>` XML 标签，以便前端拦截并渲染按钮。
*   **`<constraints>` (约束红线)**：系统的最终防御防线。强制规定输出的整体格式（如 Markdown 包裹）、严禁解释性废话，并要求模型在输出 AST 树前进行自我检查，避免结构崩溃。

### 3. `mdct_arbiter_prompt.tmpl` (多维共识打标 AI 裁决模板)
这是系统在遇到“规则平局”（如分数相近）触发延迟计算时，专用于呼叫大语言模型进行业务语义仲裁的短提示词。包含：
*   **`<role>` 和 `<task>`**：定义了它是一个数据分析与分类专家，任务是在冲突中二选一。
*   **`<context>`**：用于注入冲突数据本身，以及发生冲突的两个标签名称和描述。
*   **`<constraints>`**：强制要求模型只能输出 JSON，不允许废话，以便后端的 `mdct_scorer.go` 能够稳定地反序列化。

## 系统组装逻辑 (三明治架构)

在 `aiengine_service.go` 中，这两个模板会与用户的“本地业务语境”结合，最终拼接发送给大模型的 Prompt 结构如下：

```xml
[base_prompt.tmpl 的内容]

<custom_prompt>
{{这里是用户在前端设置中输入的自定义指令}}
</custom_prompt>

[bottom_prompt.tmpl 的内容]
```

## 如何修改和调试

1. **修改规则或示例**：如果您想让 AI 支持新的操作符（如新增了 `match_all`），或者想添加一个新的 Few-Shot 示例，请直接编辑 `base_prompt.tmpl` 文件。
2. **变量注入**：如果您在 `.tmpl` 中新增了占位符（如 `{{.NewVar}}`），请务必前往 `aiengine_service.go` 的 `template.Execute` 逻辑中，向 `map[string]any` 中补充对应的数据，否则编译或运行时会报错。
3. **生效方式**：因为使用了 `go:embed`，修改模板后**必须重新编译后端** (`wails build` 或 `go build`) 才会生效。修改配置不会立刻影响到这些静态挂载的模板。
