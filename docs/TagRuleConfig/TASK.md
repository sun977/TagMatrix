# 任务拆分文档 - TagRuleConfig 多级标签与规则重构

## 任务列表

### 任务1：后端数据结构与标签树 API (Backend Tree API)
- 目标：在 `internal/model/dto.go` 定义 `TagTreeNode`。在 `taglogic.go` 及 `app.go` 中实现 `GetTagTree`（递归组装树）、`DeleteTag`（级联删除）。
- 输出：可供 Wails 绑定的新 API 接口。

### 任务2：后端标签导入/导出 API (Backend Import/Export)
- 目标：实现 `ExportTags(filePath)` 将数据库标签表转换为层级 JSON 并保存。实现 `ImportTags(filePath)` 解析 JSON，使用“存在则忽略，不存在则新增并重建关系”策略入库。
- 输出：导入导出的 Wails 绑定方法。

### 任务3：前端左侧标签树 UI 改造 (Frontend Tree UI)
- 目标：在 `TagRuleConfig.vue` 替换原有静态菜单为 `el-tree`。
- 功能：节点悬浮显示【+】【-】按钮；节点提供右键菜单或顶部按钮用于【导入 JSON】和【导出 JSON】。点击树节点时联动右侧的表单区域。

### 任务4：前端右侧多级规则构建器 (Frontend Rule Builder)
- 目标：重写现有的单层表单。创建一个递归的规则配置结构（Vue 3 Composition API reactive state）。
- 功能：
  - 外层支持选择 `AND` 或 `OR`。
  - 内层可添加简单条件（字段、操作符、值）或添加“子逻辑组”。
  - 提供一个【预览 JSON】按钮，点击后根据配置状态序列化出与 `NeoScan/匹配规则.json` 结构一致的代码并弹窗高亮显示。
  - 点击【保存规则】时，将生成的 JSON 提交给后端的 `SaveRule` 接口。
- 难点：规则状态向 `NeoScan` 格式的转换算法。

### 任务5：全面编译与测试验证 (Build & Test)
- 目标：运行 `wails generate module`，启动应用进行端到端测试。