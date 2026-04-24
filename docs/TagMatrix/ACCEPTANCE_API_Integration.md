# 开发进度与验收追踪 (API Integration Progress)

本文档用于实时追踪 TagMatrix 项目在“前后端接口联调与交互打磨”阶段的整体开发进度。基于 `TASK_API_Integration.md` 中的原子化拆分，在此记录每个阶段的当前状态。

## 📍 整体进度概览 (Overall Progress)

- **当前处于 6A 工作流阶段**: 🟡 Stage 4 & 5 - Automate (自动化联调执行中)
- **总体任务完成度**: 20% 🚀

---

## 🛠️ 原子任务执行状态 (Atomic Tasks Status)

### [任务1] 后端基础接口补充与编译 
*状态: ✅ 已完成 (Completed)*
- [x] 补充 `GetDashboardStats` 接口（在 `app.go` 中支持概览控制台数据统计）
- [x] 补充 `GetTaggedDataList` 接口（在 `app.go` 中支持打标结果的检索）
- [x] 在 `internal/model/dto.go` 中新增对应的数据传输结构体
- [x] 运行 `wails generate module` 重新生成前端 TypeScript Bindings (`wailsjs/go/main/App`)

---

### [任务2] 联调全局设置与数据源页面 (Settings & DataSource)
*状态: ⏳ 待执行 (Pending)*
- [ ] `SettingsDialog.vue`：绑定 `GetAppConfig()` 读取配置，绑定 `SaveAppConfig()` 保存配置。
- [ ] `DataSource.vue`：绑定 `ImportData` 选择本地文件导入，成功后触发 `ElMessage` 并刷新数据。
- [ ] `DataSource.vue`：表格分页器绑定 `GetRawDataList`，真实渲染 SQLite 中的数据。

---

### [任务3] 联调标签与规则配置页面 (Tag & Rule Config)
*状态: ⏳ 待执行 (Pending)*
- [ ] 左侧树形菜单绑定 `GetAllTags()` 并进行层级渲染。
- [ ] “新增标签”右键菜单绑定 `CreateTag()`。
- [ ] 规则构建器保存按钮绑定 `SaveRule()`。
- [ ] 规则试运行面板绑定 `DryRunRule(ruleJSON, 10)`，并正确渲染返回的命中结果表格。

---

### [任务4] 联调打标任务看板与结果检索页面 (Task & TaggedData)
*状态: ⏳ 待执行 (Pending)*
- [ ] `TaskKanban.vue`：历史任务表格绑定 `GetTaskBatches()`。
- [ ] `TaskKanban.vue`：发起任务表单绑定 `RunTaggingTask()`，支持一键回退绑定 `RollbackTask()`。
- [ ] `TaggedData.vue`：筛选表单（关键字/标签/批次）和分页组件绑定补充的 `GetTaggedDataList()` 接口。
- [ ] `TaggedData.vue`：导出数据按钮绑定 `ExportData()`。

---

### [任务5] 联调概览控制台 (Dashboard)
*状态: ⏳ 待执行 (Pending)*
- [ ] `Dashboard.vue`：初始化调用 `GetDashboardStats()`。
- [ ] 动态渲染“总数据量”、“已打标数据量”、“标签总数”、“规则总数”四个统计卡片。

---

*(本追踪文档将在每个原子任务完成后实时更新状态)*
