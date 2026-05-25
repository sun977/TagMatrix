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

## 阶段二：Chat-to-Tag & Rule (标签与规则联合生成)
*定位：让 AI 能够理解并拆解复杂的业务诉求，不仅能生成符合 Matcher 引擎要求的 AST JSON，还能连带创建标签及设置规则的特殊属性（如行级计数），并在前端渲染为多步执行链。*

### 1. 深度上下文注入 (Schema & Tag Tree 感知)
**机制设计**：AI 在执行完整打标闭环前需要掌握两部分信息：
1. **数据结构**：抓取当前业务所在的 `schema_keys`，使 AI 知道条件字段（如 `target`）是否合法。
2. **已有标签树**：轻量级注入当前系统已有的标签目录层级（例如已存在 `/系统`），以便 AI 判断是新建根标签还是新建子标签。

### 2. 引入原生 Function Calling (Agent 后台自驱模式)
**机制跃升**：对于系统内聚的写库操作（建标签、建规则），不同于 SQL 查询（需要用户去控制台看结果），AI 应当完全自主代劳。我们将在此全面接入 OpenAI 原生的 `tools` 机制（Function Calling）。
向 AI 引擎层注册两个核心内部工具：
*   `create_system_tag(tag_name, parent_path, description)`
*   `create_tag_rule(target_tag_path, condition_json, is_count_mode)`

### 3. 后端自动化执行闭环 (Auto-Execution Loop)
**执行流**：
1. **AI 决策**：AI 识别到用户意图后，不再输出带有前端按钮的 XML 标签，而是返回底层的 `tool_calls`。
2. **后台静默执行**：后端的 `ChatWithAIStream`（或专门的 Agent Loop）会直接拦截到工具调用，**无需前端用户确认**，直接在 Go 服务层调用 `taglogic_service` 完成标签与规则的真实入库操作。
3. **状态回传与防幻觉**：后端如果在解析 AST JSON 或创建标签时发生规则冲突错误，会将“执行失败信息”打包为 `tool_message` 再次喂回给 AI，让其依靠自身的推理能力自我修正和重试。
4. **结果输出**：后台一系列操作全部成功闭环后，AI 再向前端流式输出最终的“总结报告”。

### 4. 前端极简交互呈现 (Result Only)
**交互呈现**：
在整个 Agent 执行过程中，前端页面无需再渲染诸如【确认执行】的按钮（唯独高危或探索性的 SQL 查询仍保留按钮）。
用户发号施令后，只需看到 AI 进入思考/操作状态。待后台动作完毕后，AI 直接回复：
> *"✅ 操作完成！我已为您在 `/系统` 目录下新建了子标签 `日志`，并为其挂载了目标字段包含 '日志' 的正则规则（已开启行级计数）。您可以前往标签配置页查看。"*

---

## 阶段三：Chat-to-Task 全自动工作流 (终极 Agent 闭环)
*定位：在标签与规则自动生成的基础上，连带触发海量数据的批处理打标任务，实现最高级的一句话自动化调度。*

### 1. 扩展执行任务工具 (Execute Task Tool)
**注册新工具**：
向大模型额外注册 `execute_tagging_task(dataset_id, rule_ids, tag_mode)` 工具。

### 2. 状态全局联动同步
**交互设计**：
*   **后台自驱任务**：一旦用户下达指令“打上标签并立即运行打标任务”，AI 同样在后台自驱调起 `TaskEngine` 开始跑批操作。
*   **前端进度条穿透**：任务在后端被触发后，利用 Wails Events 直接向前端下发全局任务事件，前端页面自动弹出系统级进度条，让用户直观看到处理进度，彻底免去手动去【打标任务看板】点执行的割裂感。

---

## 检查清单 (CheckList)
- [x] 确保前端所有跨页路由的状态接盘（如 `pendingSQL`）在消费后立即清空，避免下次进入页面引发幽灵填入。
- [x] 确保 SQL 和 JSON 内嵌套的单双引号都经过了安全的转义处理。
- [ ] 确保后台 Function Calling 闭环具备良好的自我修复能力（当生成的 AST JSON 非法时能向大模型抛回 Exception 并重试，而非终止对话）。
- [ ] 确保 Agent 静默修改库后，能通过 Wails 事件通知前端局部刷新（如自动刷新左侧菜单树或标签列表状态）。