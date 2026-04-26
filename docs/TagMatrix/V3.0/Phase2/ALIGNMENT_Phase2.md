# 需求对齐文档 - Phase 2: 标签与规则配置中心重构 (核心挑战)

## 原始需求
根据 `V3.0_REFACTORING_PLAN.md` 及 `DESIGN_TagRule_Decoupling.md`，执行 Phase 2 重构。
- **核心思想**：Global Tagging, Local Rules (全局标签，局部规则)。
- **后端**：更新 `sys_match_rules` 表，引入 `dataset_id`；更新 `SaveRule` 和 `GetRuleByTag` 接口以支持按数据集过滤和保存。同时调整打标引擎的规则拉取逻辑。
- **前端**：【标签与规则配置】页面左侧保持全局标签树不变，右侧工作台重构为按数据集分组的规则卡片视图。新建规则时强制要求先选择目标数据集，并动态加载该数据集的 `schema_keys` 供条件字段下拉框使用。

## 项目上下文

### 技术栈
- 编程语言：Golang (Wails) / TypeScript (Vue 3 + Element Plus)
- 数据库：SQLite (GORM)

### 现有架构理解
- 在 Phase 1 中，我们已经实现了 `SysDataset` 实体以及数据的物理隔离（通过 `dataset_id` 关联 `raw_data_records`）。
- 目前 `sys_match_rules` 仍然是全局共享的（仅通过 `tag_id` 关联 `sys_tags`），导致同一标签在不同数据集无法有不同的规则定义。

## 需求理解

### 功能边界
**包含功能：**
- [ ] 数据库层：`sys_match_rules` 表新增 `dataset_id` 外键约束。
- [ ] 服务层：重构 `rule_service.go` 中的 `SaveRule`、`GetRuleByTag` 等接口，加入 `dataset_id` 维度。
- [ ] 服务层：重构打标引擎逻辑 (`taglogic` / `taskengine_service.go`)，在执行打标任务前，仅拉取属于当前任务 `dataset_id` 的专属规则。
- [ ] 前端：重构 `TagRuleConfig.vue` 页面。
  - 左侧全局标签树：移除“是否有规则”的直接提示图标。
  - 右侧工作区：展示“按数据集分组的规则卡片视图”。
  - 新建/编辑规则弹窗：强制要求选择目标数据集，并基于选定的数据集动态加载 `schema_keys`，以供“条件字段”下拉列表使用。

**明确不包含（Out of Scope）：**
- [ ] 导入导出资产解耦 (Phase 4 范围)。
- [ ] 标签大盘/画像的全局聚合分析逻辑 (独立迭代模块)。

## 疑问澄清

### P0级决策点
1. **一个标签在一个数据集下是否允许存在多套规则组？**
   - **背景**：设计文档中提到，如果不加唯一索引，可以通过 `priority` 区分。
   - **建议方案**：为保持业务逻辑清晰且界面交互简洁，建议目前设计为**一个标签在一个数据集下只允许配置一套规则组（包含多个 AND/OR 条件）**，即建立 `UNIQUE(tag_id, dataset_id)` 联合唯一索引。如果您有特殊需要可以随时推翻此建议。

## 验收标准

### 功能验收
- [ ] 后端模型迁移成功，`sys_match_rules` 拥有 `dataset_id`。
- [ ] 【标签与规则配置】页面可以成功地为一个全局标签，分别为“数据集 A”和“数据集 B”建立不同的匹配规则。
- [ ] 建立规则时，下拉框里的字段列表严格跟随所选数据集的 `schema_keys`。
- [ ] 启动打标任务时，系统仅使用与该任务目标数据集绑定的规则，不出现错配和空跑。
