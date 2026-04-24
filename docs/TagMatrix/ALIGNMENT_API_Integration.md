# 需求对齐文档 - 前后端功能联调与交互打磨 (API_Integration)

## 原始需求
前端页面与布局已经完成重构并获得认可，接下来的核心任务是“打磨功能，前后端交互”。需要将目前所有 Vue 视图组件中使用的 Mock 数据替换为 Wails 生成的真实 Go 接口 (`wailsjs/go/main/App.js`) 的调用，打通整个业务链路。

## 项目上下文

### 技术栈
- 前端：Vue 3 + Composition API (`<script setup>`) + Element Plus
- 后端：Go + Wails + SQLite (GORM)
- 接口绑定层：`wailsjs/go/main/App` 及其对应的 TypeScript 类型定义 `wailsjs/go/models.ts`

### 现有后端接口分析 (`app.go`)
- **配置管理**: `GetAppConfig()`, `SaveAppConfig(newConfig)`
- **数据源**: `ImportData(filePath)`, `ExportData(batchID, exportPath)`, `GetRawDataList(page, pageSize)`
- **标签规则**: `CreateTag(tag)`, `GetAllTags()`, `SaveRule(rule)`, `DryRunRule(ruleJSON, limit)`
- **任务看板**: `RunTaggingTask(ruleIDs, batchName, isPrimary)`, `RollbackTask(batchID)`, `GetTaskBatches()`
- **AI 助手**: `ChatWithAI(message)`
- **打标结果**: *目前 `app.go` 似乎缺乏针对 TaggedData 页面（过滤/分页/查询已打标数据）的具体接口。*

## 需求理解

### 功能边界
**包含联调的页面功能：**
1. **全局设置 (`SettingsDialog.vue`)**: 对接 `GetAppConfig` / `SaveAppConfig`。
2. **数据源管理 (`DataSource.vue`)**: 对接 `ImportData` / `GetRawDataList`，实现真实的表格分页渲染。
3. **标签与规则配置 (`TagRuleConfig.vue`)**: 对接 `GetAllTags` / `CreateTag` / `SaveRule` / `DryRunRule`。
4. **打标任务看板 (`TaskKanban.vue`)**: 对接 `GetTaskBatches` / `RunTaggingTask` / `RollbackTask`。
5. **打标结果与数据检索 (`TaggedData.vue`)**: 确认是否需要补充后端查询接口，或者复用现有接口。

**疑问澄清：**
1. **Q: 概览控制台 (`Dashboard.vue`) 的统计数据有对应接口吗？**
   A: 目前 `app.go` 中没有直接返回 Dashboard 统计数据的接口（如总数据量、已打标数据量等）。需要我们在 Go 端补充一个 `GetDashboardStats()` 接口。
2. **Q: 打标结果 (`TaggedData.vue`) 的复杂过滤查询有对应接口吗？**
   A: 同样缺乏。`GetRawDataList` 只返回原始数据，不包含过滤条件。需要补充一个 `GetTaggedDataList(filter, page, pageSize)`。

## 结论
我们将分两步走：首先补充后端 (`app.go`) 缺失的统计查询与检索接口，然后逐个前端页面进行 API 的绑定与联调。