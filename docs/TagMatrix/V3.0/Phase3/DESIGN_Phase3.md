# 设计文档 - Phase 3: 打标任务引擎与看板改造

## 架构概览

Phase 3 实际上主要聚焦于 `TaskKanban.vue` 页面的“历史任务展示和过滤”增强，由于核心打标逻辑的隔离 (Rule Matching Isolation) 已经在 Phase 2 同步实现，所以此处设计的重点在于前端界面的完善和展示逻辑。

### 核心组件

#### 1. 后端服务与实体模型
- **`TagTaskBatch` 模型**:
  - `DatasetID`: uint64，已经存在，代表该任务绑定的目标数据集。
  - `GetTaskBatches` 接口: 仍然返回 `[]model.TagTaskBatch`，无需为了展示名称而大规模重构 DTO，由前端通过 `ListDatasets` 进行轻量级的映射关联。

#### 2. 前端展示层 (Frontend)
- **`TaskKanban.vue` 视图增强**:
  - **数据集列表数据源**:
    - 利用现有的 `availableDatasets`，这是通过 `ListDatasets` 取得的。确保在页面加载时完成拉取。
  - **任务历史顶部过滤器 (Header Filters)**:
    - 新增 `<el-select v-model="filterDataset" ...>`，用于选择要过滤的目标数据集（默认选项为“全部数据集”）。
  - **任务历史列表 (Table)**:
    - 新增 `<el-table-column label="目标数据集" />`，位于“任务名称”和“状态”之间。
    - 单元格内容：通过一个计算属性或方法 `getDatasetName(row.datasetId)` 返回对应的数据集名称。如果匹配不到，显示“未知数据集”或 ID。
  - **计算属性 `filteredTaskHistory` 更新**:
    - 在现有的 `filterStatus` 和 `filterTime` 基础上，增加对 `filterDataset` 的过滤条件。

## 接口设计

无需新增或修改后端 API，完全复用:
- `GetTaskBatches()` -> 获取历史任务列表
- `ListDatasets()` -> 获取映射用的全量数据集字典

## 数据流向
1. 页面加载 `onMounted` 时，并行触发 `loadData` (获取数据集) 和 `fetchBatches` (获取历史任务)。
2. 数据加载完毕后，`filteredTaskHistory` computed 函数基于 `taskHistory` 和当前的 `filterStatus`, `filterTime`, `filterDataset` 进行多维度交叉过滤。
3. 渲染 `<el-table>`，使用已缓存的 `availableDatasets` 实时转换 `datasetId` 为中文名称。
