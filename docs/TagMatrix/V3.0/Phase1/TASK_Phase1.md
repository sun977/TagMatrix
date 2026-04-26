# 任务拆分文档 - Phase 1: 基础设施与数据源管理重构

## 任务列表

### ✅ 任务1：底层数据库模型改造与数据清空
#### 输入契约
- 环境依赖：SQLite 数据库
#### 输出契约
- 新建 `sys_datasets` 表。
- `raw_data_records`, `tag_task_batches`, `sys_match_rules` 表新增 `dataset_id`。
- 清空当前所有的旧数据记录。
- GORM AutoMigrate 成功。
#### 实现约束
- 修改 `internal/model/models.go`。

### ✅ 任务2：DatasetService CRUD 接口实现
#### 输入契约
- 前置依赖：任务1
#### 输出契约
- `internal/service/dataset/dataset_service.go`
- `CreateDataset`, `ListDatasets`, `UpdateDataset`, `DeleteDataset` 逻辑。
- 绑定到 `app.go` (Wails)。

### 任务3：数据导入逻辑重构
#### 输入契约
- 前置依赖：任务2
#### 输出契约
- 修改 `internal/service/data/data_service.go` 或对应导入的地方。
- 支持接收 `DatasetID` 或 `NewDatasetName`。
- 解析 Excel 表头，将 `schema_keys` 更新到 `SysDataset` 中。
- 保存 `raw_data_records` 时附加 `dataset_id`。
- Wails 暴露 `ImportData` API。

### ✅ 任务4：数据源列表展示重构 (Wails)
#### 输入契约
- 前置依赖：任务2, 3
#### 输出契约
- 修改 `app.go` 中的 `GetRawDataList`，必须接收 `datasetID` 作为参数。

### ✅ 任务5：前端 UI 布局与逻辑重构 (Vue3)
#### 输入契约
- 前置依赖：任务4
#### 输出契约
- `DataSource.vue` 左右分栏。
- 左侧 `DatasetList` 组件。
- 导入弹窗包含新建数据集或选择已有数据集逻辑。
- 右侧根据选定的数据集，动态渲染 `schema_keys` 表头并加载数据。

## 依赖关系图
```mermaid
graph LR
    A[任务1: 数据库模型改造] --> B[任务2: DatasetService 接口]
    B --> C[任务3: 数据导入重构]
    C --> D[任务4: 数据查询重构]
    D --> E[任务5: 前端 UI 重构]
```