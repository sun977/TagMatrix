# 设计文档 - 打标结果页面 (TaggedData_View)

## 架构概览

### 页面组件设计
- **`TaggedData.vue`**: 独立路由视图，采用 Flex 布局。
  - **Header 区**: 包含页面大标题及“导出数据”操作按钮。
  - **Filter Bar (筛选区)**: 卡片样式，横向排列输入框、选择器及“查询/重置”按钮，用于过滤打标结果。
  - **Table 区**: 卡片样式，使用 `el-table` 展示数据。对于 "标签" 字段，使用 `el-tag` 并配合自定义颜色进行高亮。底部固定 `el-pagination` 分页器。

## 接口设计 (前端 Mock)
目前暂无后端真实接口，使用前端生成的响应式 mock 数据填充表格：
- `tableData`: 包含 `id`, `content`, `tags`, `batchName`, `status`, `updateTime` 等字段。
- `filterForm`: 包含 `keyword`, `tag`, `batch`。

## UI 与样式规范
- 使用 `var(--tm-bg-main)` 和 `var(--tm-border-radius-lg)` 维护白底圆角卡片。
- 主按钮（导出、查询）采用薄荷绿 `var(--tm-accent-primary)`。
- 表格无外边框，头部使用极浅灰底色 `var(--tm-bg-sidebar)` 提升辨识度。
