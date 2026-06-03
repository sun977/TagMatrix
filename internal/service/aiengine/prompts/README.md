# AI 引擎 Prompt 模板管理说明

此目录 (`internal/service/aiengine/prompts/`) 用于存放 TagMatrix 系统 AI Copilot 的核心提示词模板。这些模板会在编译时通过 Go 的 `//go:embed` 机制嵌入到二进制文件中，保证了核心业务逻辑的安全性与分发的一致性。

## 为什么使用模板文件？

1. **安全性**：避免核心业务逻辑（如 AST 解析规则、数据库 Schema 解释）以明文配置的形式暴露给最终用户。
2. **可维护性**：使用 `.tmpl` 文件格式，能够获得主流 IDE 的语法高亮支持，无需在 Go 代码中进行痛苦的字符串拼接和换行符转义。
3. **动态组装**：系统采用了“三明治”架构（Base -> User Custom -> Bottom Constraints），支持根据不同会话动态注入上下文信息。

## 目录结构与模块说明

当前目录下包含以下两个核心模板文件：

### 1. `base_prompt.tmpl` (基础核心模板)
这是系统提示词的“上半部分”，定义了 AI 的核心认知和行为准则。包含以下模块：
*   **`<role>` (角色)**：定义 AI 的身份和专业领域（如数据处理、风控专家）。
*   **`<context>` (上下文)**：包含占位符（如 `{{.DBSchema}}`），用于在请求发起前动态注入当前的数据库结构、页面环境和标签树。
*   **`<manual>` (操作手册)**：硬性规定系统的规则，包括 `SQL_Rules`（如何查询）和 `AST_Rules`（严格的 JSON 生成规范）。
*   **`<examples>` (示例库)**：提供正确（Positive）与错误（Negative）的 Few-Shot 示例，对齐模型的思维链路。

### 2. `bottom_prompt.tmpl` (防注入底线模板)
这是系统提示词的“下半部分”，也是 AI 的最终约束。
*   **`<constraints>` (约束红线)**：放置在 Prompt 的最末尾（模型注意力最高处）。强制规定输出的格式（如 Markdown 包裹）、严禁解释性废话，并包含针对用户自定义指令 (`<custom_prompt>`) 的防注入警告。

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
