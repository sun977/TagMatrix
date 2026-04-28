# 系统数据库管理 (DatabaseAdmin) 待处理事项

## 当前存在的问题或遗漏配置
1. **外部数据库导入实现**
   - 现象：在 `BackupRestore.vue` 面板中，【导入数据库 (.db)】按钮的事件 `handleImportDB` 目前只弹出了提示“暂未实现文件导入选择框绑定”。
   - 建议处理方式：需要在 Go 后端 `app.go` 或服务中，封装一个调用 Wails `runtime.OpenFileDialog` 选择外部文件的 API，然后将选中的路径直接传给 `RestoreDatabase` 接口进行文件覆盖和恢复。

2. **大数据量分页性能**
   - 现象：当前在 `TableExplorer.vue` 的 `GetVirtualDatasetData` 中，从 `raw_data_records` 表取出一个分页的数据，然后在 Go 循环里面通过 `json.Unmarshal` 获取字段动态组装表头。如果 JSON 结构很大或者该页有特别深层的嵌套，可能前端会卡顿。
   - 建议处理方式：目前为了实现灵活性采用了内存打平方案。建议以后加上字段的忽略列表，或者采用 sqlite json 扩展直接生成打平的结构。

3. **开发者模式初始状态**
   - 如果用户首次安装软件，并未开启开发者模式，需要明确告知用户如何进入“设置”中将其开启。目前已经写在文档，但建议在软件其他地方如“关于”或者普通文档里添加说明。