# TagMatrix 编码规范文档 (Coding Standards)

本文档旨在统一前后端开发过程中的代码风格、命名规则及结构规范，以保证代码的可读性和长期维护性。

---

## 1. 命名规范 (Naming Conventions)

### 1.1 Go 后端 (Backend)

遵循官方 [Effective Go](https://go.dev/doc/effective_go) 规范。

*   **包名 (Package Names)**:
    *   全小写单词，不要使用下划线或驼峰（如 `taglogic`, `taskengine`）。
    *   尽量简短且具有明确语义。
*   **文件命名 (File Names)**:
    *   全小写，多单词使用下划线分隔（如 `tag_service.go`, `match_rule.go`）。
    *   测试文件必须以 `_test.go` 结尾。
*   **变量与函数 (Variables & Functions)**:
    *   **驼峰命名法 (CamelCase)**。
    *   公开/导出 (Exported)：首字母大写，即 `PascalCase`（如 `GetRawDataList`, `SysTag`）。
    *   私有/未导出 (Unexported)：首字母小写，即 `camelCase`（如 `parseExcelFile`, `batchSize`）。
*   **接口名 (Interfaces)**:
    *   单方法接口通常以 `er` 结尾（如 `Reader`, `Matcher`）。
    *   多方法接口使用具有领域意义的 `PascalCase` 命名（如 `TaskEngineService`）。
*   **常量 (Constants)**:
    *   使用 `PascalCase`（如 `MaxBatchSize`）。不需要像 Java/C 里的全大写下划线命名。
*   **结构体与字段 (Structs & Fields)**:
    *   使用 `PascalCase`。必须为 JSON 序列化和 GORM 映射显式提供 `tag`。
    ```go
    type UserRecord struct {
        ID        uint   `json:"id" gorm:"primaryKey"`
        FirstName string `json:"first_name" gorm:"column:first_name"`
    }
    ```

### 1.2 Vue3 + TypeScript 前端 (Frontend)

遵循 [Vue 3 官方风格指南](https://cn.vuejs.org/style-guide/)。

*   **组件名 (Components)**:
    *   文件命名和导入时统一使用大驼峰 `PascalCase`（如 `TagTree.vue`, `RuleEditor.vue`）。
    *   基础/通用组件应该以特定前缀开头（如 `BaseButton.vue`, `AppIcon.vue`）。
*   **变量与函数 (Variables & Functions)**:
    *   小驼峰 `camelCase`（如 `fetchData`, `isModalVisible`）。
*   **常量 (Constants)**:
    *   全大写并使用下划线分隔 `UPPER_SNAKE_CASE`（如 `MAX_UPLOAD_SIZE`, `DEFAULT_PAGE_SIZE`）。
*   **类与接口 (Classes & Interfaces)**:
    *   大驼峰 `PascalCase`（如 `interface SysTag { ... }`）。
*   **CSS / 样式类名 (Classes)**:
    *   使用中划线/短横线 `kebab-case`（如 `.tag-list-container`, `.btn-primary`）。

### 1.3 数据库 (SQLite + GORM)

*   **表名 (Table Names)**:
    *   全小写，下划线分隔，使用**复数形式**（如 `sys_tags`, `raw_data_records`）。
*   **字段名 (Column Names)**:
    *   全小写，下划线分隔（如 `created_at`, `batch_id`, `rule_json`）。
    *   主键统一叫 `id`。外键使用 `[关联表单数]_id` 格式（如 `tag_id`）。
    *   布尔值前缀使用 `is_` 或 `has_`（如 `is_enabled`, `is_primary`）。

---

## 2. 目录结构规范 (Directory Structure)

### 2.1 后端结构 (Standard Go Project Layout)
*   `cmd/`: 项目主入口（包含 `main.go`）。
*   `internal/`: 私有应用代码，不允许被其他项目直接 import。
    *   `model/`: GORM 数据库实体模型定义。
    *   `service/`: 核心业务逻辑层（按业务域划分目录，如 `taglogic/`, `taskengine/`）。
    *   `pkg/`: 可在内部项目间复用的公共组件包（如 `matcher/` 规则引擎）。
*   `frontend/`: 存放 Wails 的前端源码（Vue3 + Vite）。

### 2.2 前端结构 (Vue3)
*   `src/components/`: 可复用组件（如 `RuleBuilder`, `TagSelector`）。
*   `src/views/` (或 `pages/`): 页面级组件（如 `DataImport.vue`, `TagManager.vue`）。
*   `src/store/`: Pinia 状态管理库。
*   `src/utils/`: 公共工具函数（如日期格式化、Wails API 封装）。
*   `src/assets/`: 静态资源（图片、全局 CSS）。

---

## 3. 代码质量与错误处理 (Code Quality & Error Handling)

### 3.1 错误处理 (Error Handling)
*   **Go 侧**:
    *   不能抑制错误！对于任何返回 error 的函数，必须检查并处理（`if err != nil { return err }`）。
    *   使用 `fmt.Errorf("do something failed: %w", err)` 包装并向上传递错误上下文。
*   **前端侧**:
    *   调用 Wails API 必须使用 `try...catch` 块捕获 Go 侧抛出的错误，并通过 Element Plus 的 `ElMessage` 统一向用户反馈。

### 3.2 注释规范 (Comments)
*   **公开的函数、方法、结构体**必须有注释，注释内容以该名称开头。
    ```go
    // GetRawDataList 分页获取原始数据列表
    func GetRawDataList(page, pageSize int) { ... }
    ```
*   业务逻辑复杂的代码块上方应使用多行注释解释**“为什么 (Why)”**这么做，而不仅仅是“做了什么 (What)”。

### 3.3 状态与常量维护
*   绝对禁止在代码中硬编码 (Hardcode) 魔法数字 (Magic Numbers) 或魔法字符串。
*   所有的业务状态（如打标任务状态：运行中、完成、失败）应该在 Go 中定义为常量 (Constants)，或者在 TypeScript 中定义为枚举 (Enum)。