# 全局智能助手 (AI Copilot Sidebar) 实施方案

## 1. 概述
全局智能助手（AI Copilot Sidebar）是 TagMatrix 系统的重要组成部分，作为常驻侧边栏的“全局副驾驶”，其生命周期独立于应用主路由页面，为用户在标签管理、数据清洗和规则配置全流程中提供无缝的 AI 辅助支持。

本文档基于 `UI_DESIGN.md` 中 2.8 节的设计要求，详细规划前端组件、状态管理、后端（Wails/Go）交互及分阶段开发计划。

## 2. 架构设计

### 2.1 前端架构 (Vue 3 + Pinia)
*   **布局层 (`Layout.vue`)**: 在全局主框架右侧新增可折叠面板区域，通过 CSS 控制动画展开与收起。
*   **状态管理 (`Pinia - useAIStore`)**: 管理对话历史记录、展开/收起状态、当前选中模型、上下文感知开关状态。确保跨路由切换时聊天记录不丢失。
*   **组件拆分**:
    *   `AICopilotSidebar.vue`: 侧边栏主容器。
    *   `AICopilotHeader.vue`: 头部控制面板（模型切换、上下文开关、快捷指令）。
    *   `AICopilotChat.vue`: 对话渲染区（气泡、Markdown渲染、Action Blocks）。
    *   `AICopilotInput.vue`: 底部输入区（多行文本、快捷命令 `/`）。

### 2.2 后端架构 (Wails + Go)
*   **AI Service (`internal/service/aiengine`)**: 负责对接 OpenAI/Claude 等大模型接口，支持流式输出 (Streaming)。
*   **Config Service (`internal/config`)**: 负责大模型 API Key 及默认设置的读取与保存。
*   **Action Dispatcher**: 解析大模型返回的特定 Action Block JSON，转译并调用底层系统函数（如执行 SQL 提取、调用 API 等）。

### 2.3 UI 风格与主题适配 (UI Style & Theme Switching)
*   **统一设计语言**: 助手侧边栏的 UI 视觉（如薄荷绿主色调、圆角按钮、卡片阴影等）必须与 TagMatrix 现有页面的全局风格严格保持一致。
*   **统一主题切换支持**:
    *   面板需全面响应全局的主题切换系统（Light / Dark 模式切换）。
    *   背景色、文字颜色、边框和气泡高亮均应使用全局 CSS 变量（或对应的原子化类名）进行定义，坚决避免硬编码色值。
    *   代码高亮（Highlight.js）与 Markdown 渲染区也需对应准备两套样式，跟随应用主题自动切换，保证代码在深色/浅色模式下均清晰可读。

## 3. 详细模块设计

### 3.1 面板头部 (Header)
1.  **模型切换器 (Model Switcher)**:
    *   从后端读取已配置的可用模型列表（如 `GPT-4o`, `Claude-3.5-Sonnet`）。
    *   使用精简风格的 Dropdown 组件进行快速切换，切换结果同步更新至全局 Config 并在 `Pinia` 持久化。
2.  **当前上下文感知 (Context Awareness)**:
    *   通过 Vue Router 钩子或全局事件总线，获取当前用户停留的页面信息及选中数据（如选中标签 ID、数据库表结构）。
    *   当开启时，每次发送对话会将这些上下文数据隐式拼接至 Prompt 中发送。
3.  **系统指令快捷注入 (Prompt Inject)**:
    *   提供快捷弹窗或气泡，允许用户临时追加系统级提示词，针对本次 Task 调整 AI 的行为边界。

### 3.2 主对话流区 (Chat Stream)
1.  **Markdown与代码高亮**:
    *   引入 `markdown-it` (或 `marked`) 及 `highlight.js` 实现丰富排版。
    *   封装自定义代码块组件，针对 SQL、JSON 提供“一键复制(Copy)”按钮。
2.  **动作组件 (Action Blocks)**:
    *   **交互机制**: 当后端 AI 判断可直接执行操作时，返回特定格式的文本（例如 `<action type="execute_sql" query="SELECT * FROM tags"/>`）。
    *   **前端渲染**: 拦截该特定格式，渲染为“执行该操作”的交互按钮。
    *   **Wails 交互**: 用户点击按钮后，调用对应的 Go 方法 `window.go.main.App.ExecuteAction(...)` 实现“对话即交互”。
3.  **空状态引导**:
    *   若后端检测到未配置 API Key，拦截渲染逻辑，展示缺省页及“立即配置 AI 引擎”按钮，点击后通过事件触发打开全局设置（Settings）面板的对应 Tab。

### 3.3 底部输入区 (Input Box)
1.  **输入交互**:
    *   使用可自适应高度的 `textarea`。
    *   绑定 `Enter` 键发送，`Shift+Enter` 键换行。
2.  **快捷指令 (Slash Commands)**:
    *   监听输入框值的变化，当输入 `/` 且位于行首时，弹出快捷指令浮层（如 `/sql 帮我写提取SQL`、`/regex 生成正则`）。
    *   支持键盘上下键选择及回车确认上屏。

### 3.4 多智能体支持 (Agent Hub - 未来扩展预留)
*   预留 Agent Store 结构：在后端 Config 中设计 `agents` 数组结构，保存多个预设的自定义 Agent（包含 Name, Avatar, SystemPrompt, Tools）。
*   在前端 Header 预留扩展入口，未来可升级为 Agent 切换器。

## 4. 实施计划 (分期路线图)

### Phase 1: 基础框架与 UI 搭建 (预计 2 天)
*   **Task 1**: 在 `Layout.vue` 中完成右侧边栏布局与动画，编写基础的 `Pinia` 状态。
*   **Task 2**: 拆分并实现三个子组件 (`Header`, `Chat`, `Input`) 的纯静态 UI。
*   **Task 3**: 实现输入框自适应高度及 Enter/Shift+Enter 发送交互。
*   **Task 4**: 接入系统全局样式变量，完成对 Light/Dark 模式统一主题切换的适配验证。

### Phase 2: AI 核心对话链路打通 (预计 3 天)
*   **Task 1**: Go 后端实现 API Key 配置读取与大模型流式对话 API 对接 (`service/aiengine`)。
*   **Task 2**: 前端接入 Wails 后端接口，实现真实对话、流式文字渲染 (Streaming Typewriter effect)。
*   **Task 3**: 引入 Markdown 渲染器，支持代码块高亮及“一键复制”。
*   **Task 4**: 完善 API Key 缺失时的空状态引导交互。

### Phase 3: 上下文感知与 Action Blocks 高级特性 (预计 3-4 天)
*   **Task 1**: 实现 Context 收集机制，将左侧工作区（如 Dataset表格结构）作为提示词背景发送。
*   **Task 2**: 定义前后端通信的 Action 协议，前端实现 `<action>` 标签拦截与薄荷绿按钮渲染。
*   **Task 3**: 实现至少 1-2 个基础的 Action 调用（如“一键导入提取规则”、“一键执行SQL校验”）。

### Phase 4: 体验优化与多 Agent 基础 (✅ 已完成)
*   **Task 1**: 实现 `/` 快捷指令菜单提示。 (✅ 已完成)
*   **Task 2**: Header 中加入 System Prompt 的快捷注入操作。 (✅ 已完成)
*   **Task 3**: 走查所有 UI 细节、调整气泡间距与配色、异常处理与边界测试。 (✅ 已完成)

## 5. 潜在风险与注意事项
1.  **流式通信**: Wails 原生目前对流式(SSE)的支持主要通过事件 (Events) 实现。在 Go 侧循环发送 Event，在 Vue 侧监听该 Event 并拼接到当前气泡中，需注意并发对话时的 Event ID 隔离。
2.  **上下文长度管理**: 若用户开启“上下文感知”，需在发送前在本地计算 Token 长度。如果当前页面携带的表格元数据过长，需要进行截断处理，避免超出模型限制或浪费过多 Token。
3.  **安全性**: Action Blocks 涉及“通过对话执行系统级操作”。应在执行高危操作（如删除、修改数据库等）前，确保 Action 按钮触发二次确认弹窗。