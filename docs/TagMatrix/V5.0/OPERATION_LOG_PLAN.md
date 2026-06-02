# TagMatrix V5.0 规划：操作日志与 AI 调试追踪系统

## 1. 背景与目标
随着 TagMatrix 系统复杂度的提升，当前的日志体系仅依赖底层的 `app.log` 文件记录。这种方式存在以下痛点：
- **无业务语义**：包含大量 GORM SQL 和底层调试信息，噪音极大，难以追踪用户的实际操作。
- **排障困难**：在处理规则误删、打标任务异常时，无法快速还原“案发现场”。
- **AI 调试黑盒**：AI 引擎的输入输出（Prompt与生成结果）缺乏结构化的追踪手段，难以对大模型响应进行分析和调优。

**目标**：在 V5.0 中引入一套完善的日志分级架构，将底层系统日志与高层业务日志解耦，并提供前端可视化界面，支持业务审计与 AI 引擎的深度调试。

## 2. 日志分类架构

为了满足不同维度的追踪需求，系统将日志划分为两大类：

### 2.1 系统运行日志 (System Log)
- **存储介质**：本地文本文件 `app.log`（基于 Zap 等日志库）
- **主要内容**：GORM 打印的 SQL 语句、服务启动/关闭信息、底层 Panic 与系统级异常。
- **使用场景**：仅供开发者排查底层系统 Bug，不对终端用户暴露。
- **保留策略**：维持现状，按文件大小或日期自动滚动分割（Log Rotation）。

### 2.2 业务操作日志 (Business Log)
- **存储介质**：SQLite 业务数据库表（如 `sys_operation_logs`）
- **主要内容**：用户的增删改查动作（如操作数据源、规则、任务）、AI 引擎的请求与响应追踪。
- **使用场景**：供系统管理员和高级用户在前端界面中查询、溯源、排障，以及调优大模型 Prompt。

## 3. 业务日志模型设计

在 `internal/model/models.go` 中新增 `SysOperationLog` 模型。

| 字段名 | 类型 | 说明与示例 |
| :--- | :--- | :--- |
| `ID` | `uint` | 主键，自增 |
| `LogType` | `string` | 日志大类：`UserAction` (常规用户操作), `AITrace` (AI 追踪) |
| `Module` | `string` | 业务模块枚举：如 `DataSource`, `RuleEngine`, `AICopilot`, `TaskEngine` |
| `Action` | `string` | 具体动作枚举：如 `Create`, `Update`, `Delete`, `GenerateRule`, `Chat` |
| `TargetID` | `uint` | 被操作业务对象的 ID（如数据集ID），便于关联查询 |
| `Status` | `string` | 执行结果状态：`Success` (成功), `Failed` (失败), `Timeout` (超时) |
| `Message` | `string` | 人类可读的短摘要：如“删除了数据集 [电商测试数据]” |
| `ReqPayload` | `text` | **请求负载(JSON)**：AI的完整 Prompt 上下文，或常规请求的核心入参 |
| `RespPayload` | `text` | **响应负载(JSON)**：AI的原始返回内容及 Token 消耗，或错误的 Stack 信息 |
| `Duration` | `int` | 执行耗时 (毫秒)：主要用于分析 AI 响应速度和打标任务性能瓶颈 |
| `CreatedAt` | `datetime`| 操作发生时间 |

## 4. 核心功能与机制规划

### 4.1 全局 AI 调试开关 (Debug Mode)
- **机制**：在“系统设置”提供 `EnableAIDebugMode` 选项。
- **效果**：关闭时仅记录 `UserAction` 的重要增删改；开启后，`aiengine` 将全量记录每次大模型调用的详细请求/响应上下文（即 `LogType=AITrace`）。

### 4.2 异步日志落库
- **避免阻塞**：不应在核心业务流程中同步执行日志的 `INSERT` 操作。
- **实现方案**：在 Service 层维护一个 Go Channel，通过独立的 Goroutine 消费 Channel 实现异步批量写库。

### 4.3 大字段保护与性能优化
- **分页列表优化**：在前端查询日志列表时，后端默认不返回 `ReqPayload` 和 `RespPayload` 字段，避免内存激增和网络带宽浪费。
- **按需加载**：只有当用户在前端点击“查看详情”时，才通过 ID 查询单条日志的完整 Payload。

### 4.4 日志生命周期管理
- **自动清理**：AI Trace 负载占用较大空间。系统后台支持配置保留策略（如：仅保留最近 7 天或 2000 条），定期执行自动清理。

## 5. 前端交互视图设计

在 TagMatrix 菜单栏（如系统设置内）新增「操作日志」页面。

### 5.1 列表视图
- **过滤条件**：提供日期范围选择器、`LogType` 下拉框、`Module` 多选框、`Status` 筛选。
- **数据表格**：按时间倒序展示模块、动作、状态、摘要和耗时。

### 5.2 详情抽屉 (Drawer/Dialog)
- 点击表格行的“详情”按钮，右侧滑出抽屉。
- **常规操作详情**：高亮展示变更前后的字段 JSON。
- **AI Trace 详情面板**：提供左右分栏视图（Diff 视图风格）：
  - **左侧**：渲染完整的 `messages` 数组（System Prompt, User Prompt）。
  - **右侧**：渲染 AI 原始 Markdown 返回及 Token 消耗统计，方便直观调优。

## 6. 实施路径 (Roadmap)

- **Phase 1 (MVP)**：建立基础 DB 模型，实现前端列表。完成数据源、规则等**高危操作**的埋点。
- **Phase 2 (AI Trace 增强)**：引入 Debug 模式开关，完成 AI 请求/响应的 JSON 序列化与抽屉双栏对比视图。
- **Phase 3 (性能与维护)**：实现 Channel 异步写入改造与定时清理机制，保障系统的长期稳定运行。