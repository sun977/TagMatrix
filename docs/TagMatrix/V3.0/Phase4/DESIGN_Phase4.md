# 设计文档 - 局部资产导出导入 (dataset_with_rules)

## 架构概览

### 整体流程图
```mermaid
graph TD
    A[前端: 导出业务资产] --> B[Wails API: ExportDatasetWithRules]
    B --> C[根据 dataset_id 获取 SysDataset 和该数据集的 SysMatchRule]
    C --> D[关联 SysTag 获取其 Path]
    D --> E[组装 ExportJSON 并写入文件]

    F[前端: 导入业务资产] --> G[Wails API: ImportDatasetWithRules]
    G --> H[解析 dataset_with_rules.json]
    H --> I[事务: 查询或创建 SysDataset]
    I --> J[事务: 遍历规则, 根据 tag_path 查询 SysTag]
    J --> K[如果找到 Tag, 创建或更新 SysMatchRule (保证一个标签在一个数据集下只有一套规则)]
```

### 核心组件

#### 导出/导入服务 (`dataset_service.go` 或新开 `dataset_export.go`)
- 职责：处理 `dataset_with_rules.json` 的序列化与反序列化，并维护与 `sys_tags` 表的关联逻辑。
- 接口：
  - `ExportDatasetWithRules(datasetID uint64, exportPath string) error`
  - `ImportDatasetWithRules(importPath string) (model.ImportResult, error)` (返回导入了多少规则，跳过了多少规则等统计信息)
- 依赖：`model.DB` 

## 接口设计

### DTO 设计 (放在 `internal/model/dto.go` 中)

```go
type ExportRule struct {
    TagPath   string `json:"tag_path"` // 解耦 TagID
    Name      string `json:"name"`
    Priority  int    `json:"priority"`
    RuleJSON  string `json:"rule_json"`
    IsEnabled bool   `json:"is_enabled"`
}

type ExportDatasetWithRules struct {
    Version     string       `json:"version"` // e.g. "1.0"
    Name        string       `json:"name"`
    Description string       `json:"description"`
    SchemaKeys  string       `json:"schema_keys"`
    Rules       []ExportRule `json:"rules"`
}

type ImportResult struct {
    DatasetName string `json:"dataset_name"`
    RuleImported int   `json:"rule_imported"`
    RuleSkipped  int   `json:"rule_skipped"`
}
```

## 数据模型交互

### 实体设计
- 导出时：
  ```sql
  SELECT r.*, t.path as tag_path 
  FROM sys_match_rules r 
  JOIN sys_tags t ON r.tag_id = t.id 
  WHERE r.dataset_id = ?
  ```
- 导入时：
  ```sql
  SELECT id FROM sys_tags WHERE path = ? LIMIT 1
  -- 然后
  INSERT INTO sys_match_rules (dataset_id, tag_id, ...) VALUES (...) 
  ON CONFLICT(dataset_id, tag_id) DO UPDATE SET rule_json=...
  ```
