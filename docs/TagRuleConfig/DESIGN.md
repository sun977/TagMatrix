# 设计文档 - 多级标签树与规则构建器

## 架构概览

### 核心组件
1. **TagTreeManager (左侧树形菜单)**
   - 职责：展示多级标签树，处理新建子标签、删除分支、导入/导出。
   - 交互：使用 `el-tree` 提供丰富的插槽自定义。悬浮显示增删按钮，右键展示导入导出操作。
2. **RuleBuilder (右侧规则构建面板)**
   - 职责：动态表单构建器，提供嵌套的 `AND/OR` 分支条件逻辑，最后将其序列化为特定 JSON 格式。
   - 结构：递归组件 `RuleGroup.vue`，内部包含“添加条件”或“添加子逻辑组”。
   - JSON 预览：将树形配置深度遍历，转化为 `NeoScan` 定义的规范结构并弹窗预览。

## 接口设计

### 后端 API (Wails)
- **标签树相关 (`app.go` & `taglogic`)**
  - `GetTagTree() ([]model.TagTreeNode, error)`: 组装好的树形结构，方便前端直接渲染。
  - `CreateTag(tag model.SysTag) error`: 支持传递 `ParentID` 进行创建。
  - `DeleteTag(tagID uint64) error`: 级联删除该节点及其所有子节点。
  - `ExportTags(exportPath string) error`: 序列化标签树为 JSON 并写入文件。
  - `ImportTags(filePath string) error`: 解析外部 JSON，平滑合并入现有标签体系。

- **规则相关 (`app.go` & `taglogic`)**
  - `SaveRule(rule model.SysMatchRule) error`: 保存序列化后的 JSON 字符串。
  - `GetRuleByTag(tagID uint64) (*model.SysMatchRule, error)`: 读取现存的规则供回显。

## 数据模型

### 实体设计
1. **SysTag (已存在)**: `ID`, `ParentID`, `Name`...
2. **TagTreeNode (新增DTO)**: 
```go
type TagTreeNode struct {
    model.SysTag
    Children []TagTreeNode `json:"children"`
}
```
3. **NeoScan 规则格式**: 
```json
{
  "and/or": [
    { "field": "x", "operator": "y", "value": "z" },
    { "or": [ ... ] }
  ]
}
```