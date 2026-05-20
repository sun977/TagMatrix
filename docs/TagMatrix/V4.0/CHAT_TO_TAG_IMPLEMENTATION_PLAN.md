# V4.0 全链路 Chat-to-Tag 实施计划 (分阶段路线图)

> **目标**：实现 AI Copilot 从“被动生成代码”到“主动执行系统级操作”的全面升级。最终让用户可以通过一句话完成：创建标签 -> 绑定规则 -> 执行任务。

---

## 阶段一：Chat-to-SQL (初探动作编排) ✅ **[已完成]**
*定位：跑通“AI生成可执行动作 -> 前端拦截渲染按钮 -> 点击跨页执行”的基础框架。*

### 1. 后端改动：系统提示词 (System Prompt) 升级
**文件**：`internal/service/aiengine/aiengine_service.go`
**任务**：强制 AI 在输出 SQL 时额外输出动作标签。
**修改策略**：
在现有的 `ChatWithAIStream` 和 `ChatWithAI` 方法中注入指令：
```text
[系统交互指令]
如果请求编写SQL，除Markdown代码块外，请务必在回答末尾输出动作标签，格式如下：
<action type="execute_sql" query="这里填入你写的完整SQL语句" label="一键去 SQL 控制台执行" />
注意：
1. Action属性必须用双引号。
2. query 属性内的 SQL 字符串字面量请务必使用单引号（'）。极少数情况必须用到双引号时，使用 HTML 实体 &quot; 进行转义。
```

### 2. 前端改动：聊天消息流解析器升级
**文件**：`frontend/src/components/AICopilot/AICopilotChat.vue`
**任务**：将 `<action>` 标签剥离并渲染为可点击的 UI 按钮。
**修改策略**：
1. 编写正则表达式，在接收流式数据或完毕时拦截 `content` 中所有 `<action type="..." ... />` 标签。
2. 提取 `query` 和 `label` 属性存入消息对象，用 `v-for` 渲染出带有“一键执行”按钮的 Action 卡片，并将标签从纯文本 `content` 中剔除。

### 3. 状态管理与路由接力 (接盘执行)
**文件**：`frontend/src/store/useAIStore.ts` & `frontend/src/views/dataAdmin/SqlConsole.vue`
**任务**：按钮点击后跨页携带数据。
**修改策略**：
1. 点击按钮调用 `aiStore.setPendingSQL(query)` 并跳转 `router.push({ name: 'SqlConsole' })`。
2. `SqlConsole.vue` 的 `onMounted` 和 `onActivated` 中检查 `pendingSQL`，有值则自动填入 SQL 编辑器并销毁状态。

---

## 阶段二：Chat-to-Rule (规则引擎渗透)
*定位：让 AI 生成符合 Matcher 引擎要求的 AST JSON，并直接应用到表单。攻克幻觉问题。*

### 1. 上下文强注入 (Schema 感知)
**机制设计**：AI 写规则前必须知道有啥字段可以查。前端在用户当前处于“标签配置页”或明确提及某数据集时，抓取对应的 `schema_keys`（如 `["user_id", "age", "city"]`），并在发送消息时隐式追加到 Prompt 中。

### 2. 定义 Create Rule Action
**指令扩展**：
要求 AI 如果要生成打标规则，必须输出：
```xml
<action type="create_rule" condition_json="{...AST JSON...}" label="将此规则应用到编辑器" />
```

### 3. 数据层合法性校验 (防御幻觉)
**安全防护**：为了防止 AI 乱写 JSON 搞崩页面，前端在拦截到 `create_rule` action 后，必须先调用一次后端的 `DryRun` 或校验 API。
*   **若合法**：正常渲染应用按钮。
*   **若非法**：前端在后台自动向 AI 发送错误信息（用户不可见），要求重写，直到校验通过。

### 4. 路由与编辑器接力
**目标接盘侠**：`TagRuleConfig.vue`。
当点击“应用到编辑器”时，前端读取 `condition_json`，并将其自动还原到可视化的规则树形选择器 (Rule Builder) 中。

---

## 阶段三：Chat-to-Task 全自动工作流 (终极 Agent 闭环)
*定位：彻底摆脱页面间的手动点击，实现“一句话调度全系统接口”。*

### 1. 接入 OpenAI Function Calling 协议
**架构升级**：在后端 `aiengine_service.go` 中，不再使用粗暴的正则 XML 拦截，而是正式引入 OpenAI 原生的 `tools` 数组配置。
**开放系统 API**：
向大模型注册三个核心工具：
1. `create_tag(name, desc, dataset_id)`
2. `bind_rule(tag_id, condition_ast)`
3. `execute_task(dataset_id, rule_ids, tag_mode)`

### 2. 授权流 (Human-in-the-loop)
**交互设计**：
AI 判断出意图后，直接返回 `tool_calls`。前端接收后不渲染 Markdown，而是渲染一张 **“任务执行确认单 (Execution Plan)”**：
> **即将执行以下操作：**
> 1. 新建标签：[高净值老人]
> 2. 绑定规则：[age > 60 AND amount > 1000]
> 3. 运行任务：针对 [测试数据集]
> 
> [ 确认并一键执行 ] | [ 取消 ]

### 3. 编排执行器
前端点击“确认”后，按照 AI 规划的顺序，依次静默调用后台的 API 接口，并展示全局进度条。完成后返回一条系统回复：“已为您执行完毕，共命中 105 条数据。”

---

## 检查清单 (CheckList)
- [x] 确保前端所有跨页路由的状态接盘（如 `pendingSQL`）在消费后立即清空，避免下次进入页面引发幽灵填入。
- [x] 确保 SQL 和 JSON 内嵌套的单双引号都经过了安全的转义处理。
- [ ] 确保 AI 只有在被明确授权（点击按钮）的情况下才能调用写库操作（如建标签/跑任务）。