# 系统数据库管理 (DatabaseAdmin) 验收与评估文档

## 执行结果验证

### 1. 需求与边界实现检查
- **[通过] 全局设置/Pinia状态控制**：在 `SettingsDialog.vue` 中增加了“开发者模式”的开关；在 `Layout.vue` 中通过 `appConfig.adv.developer_mode` 动态过滤 `requireDev` 为 true 的导航路由，实现了隐藏入口的需求。
- **[通过] SQL 查询终端 (SqlConsole)**：引入 `vue-codemirror` 和 `@codemirror/lang-sql`。支持编写 SQL（带有语法高亮）并发送到后端 Go `ExecuteRawSQL` 接口，自动区分 SELECT 语句（返回多态结果集/动态列渲染）和非 SELECT 语句（返回影响行数）。
- **[通过] 可视化表结构与数据 (TableExplorer)**：实现了物理表与业务表双模式。
  - 物理表模式：动态使用 `SELECT name FROM sqlite_master WHERE type='table'` 获取全部系统表名称，不受未来新表增加的影响。
  - 业务数据集模式：通过分析 `raw_data_records` 的 `data` 字段 JSON，打平列属性渲染表格，并在单元格级别实现了修改操作（回写至 JSON 且不破坏未修改的字段）。
- **[通过] 备份与还原中心 (BackupRestore)**：实现了快照式的文件备份。
  - 备份：读取配置目录下的 `.db` 文件进行保存。
  - 恢复机制：Go 端提供 `RestoreDatabase` 接口，在覆盖 `.db` 文件前**显式调用 `db.Close()`** 释放文件锁；完成后前端触发 `window.location.reload()`。
  - 删除快照：直接通过后端 API 擦除对应的备份文件。

### 2. 质量评估与规范符合度
- **代码质量**：
  - 前端：使用 Vue 3 组合式 API + Element Plus，拆分出独立路由组件 `DatabaseAdmin.vue` 及三个子页面组件，保持了模块内聚。
  - 后端：新建 `dataadmin` 包隔离职责。Go 的底层查询使用 `gorm.Raw` 和 `Rows.Scan` 将任意 SQL 转为 map slice，实现精细。
- **动态适配性**：符合“后续如果有新的系统表也能支持”的需求（直接查询 sqlite_master 而非硬编码表名数组）。
- **容错处理**：备份和恢复前增加了强制前端弹窗确认机制，将意外导致数据库丢失的风险降到最低。

## 待处理事项与优化建议 (TODO)
1. **[待办] 外部 DB 导入的交互完善**：`BackupRestore.vue` 面板中的“导入数据库 (.db)”按钮目前只预留了提示，实际开发可能需要调用 Wails 的 `OpenFileDialog` 获取本机外部的 `.db` 文件路径并传入 `RestoreDatabase`。
2. **[优化] TableExplorer 的大数据分页**：目前虚拟数据集是通过 GORM 进行 `Offset` 分页后，在内存中动态提取 JSON 列名，当单页条数很大或者字段巨多时，效率可能会受影响（建议后续考虑直接在 SQL 中使用 `json_tree` 来辅助打平）。

## 交付物列表
- 页面路由组件：`frontend/src/views/dataAdmin/DatabaseAdmin.vue`
- SQL终端子组件：`frontend/src/views/dataAdmin/SqlConsole.vue`
- 表浏览器子组件：`frontend/src/views/dataAdmin/TableExplorer.vue`
- 备份中心子组件：`frontend/src/views/dataAdmin/BackupRestore.vue`
- 后端业务包：`internal/service/dataadmin/admin_service.go`, `internal/service/dataadmin/backup_service.go`
- 全局绑定：更新了 `app.go` 中的 Service 注册及 Wails API 暴露。
- 依赖添加：`frontend/package.json` 添加了 `vue-codemirror` 和 `@codemirror/lang-sql`。

---
*流程总结：6A 模型已走完所有阶段，最终模块已达到设计文档中约定的可用状态。*