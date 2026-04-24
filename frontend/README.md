# TagMatrix 前端目录结构说明 (Frontend Directory Structure)

本项目前端基于 **Vue 3 + Vite + TypeScript + Element Plus** 构建，并与 Wails 后端深度集成。
以下是前端目录 (`frontend/`) 下各个文件与文件夹的作用说明，帮助您快速定位和开发。

---

## 📂 根目录 (`frontend/`)

*   **`index.html`**: Vite 项目的 HTML 入口文件。
*   **`package.json`**: 记录前端所有的 npm 依赖（如 vue, element-plus, pinia 等）和执行脚本。
*   **`tsconfig.json`** / **`tsconfig.node.json`**: TypeScript 的配置文件，定义了类型检查的严格程度和路径别名。
*   **`vite.config.ts`**: Vite 构建工具的配置文件，可以在这里配置代理、插件和别名。
*   **`wailsjs/`**: **【核心目录】**由 Wails 在编译或运行 `wails dev` 时**自动生成**的目录。
    *   里面包含了所有我们在 Go 后端 `App` 结构体中暴露出来的方法的 TypeScript 接口定义 (Bindings)。
    *   **注意**: 绝对不要手动修改此目录下的文件！每次 Go 代码更新，Wails 都会自动覆盖它。在 Vue 组件中，我们通过 `import { ImportData } from '../../wailsjs/go/main/App'` 来调用后端能力。

---

## 📂 源码目录 (`frontend/src/`)

所有的业务代码都在这里。

*   **`main.ts`**: 前端代码的真正入口。负责创建 Vue 实例，挂载路由 (Router)、状态管理 (Pinia) 和 UI 组件库 (Element Plus)。
*   **`App.vue`**: 根组件，通常只负责引入 `Layout.vue`，作为最外层的包裹器。

### 📁 `assets/` (静态资源)
存放不会被动态编译的基础静态文件。
*   **`images/`**: 图片资源。
*   **`styles/`**:
    *   `main.scss`: 全局样式表。这里定义了**主题色变量**（如 `--tm-accent-primary` 薄荷绿）并覆盖了 Element Plus 的默认蓝主题。同时定义了全局通用的滚动条、基础字体样式。

### 📁 `components/` (公共组件)
存放可以在多个页面中复用的组件，或者是为了拆分巨型页面而抽离出的木偶组件。
*   **`Layout.vue`**: **【核心骨架】**。定义了应用的“侧边栏 (Sidebar) + 顶部栏 (Header) + 右侧 AI 面板 (Right Panel) + 中央路由区”的整体布局结构。

### 📁 `router/` (路由配置)
*   **`index.ts`**: 定义了前端的页面路由表。使用 Hash 模式 (`createWebHashHistory`) 以保证在 Wails 打包为单体 exe 后，页面刷新和跳转不会出现 404。包含了如下核心路由：
    *   `/dashboard`: 概览
    *   `/data-source`: 数据源管理
    *   `/tag-rule`: 标签与规则配置
    *   `/task-kanban`: 任务看板

### 📁 `store/` (状态管理 - 计划中)
使用 Pinia 管理跨组件的全局状态。
*   (待创建) `tagStore.ts`: 可能用于缓存全局的标签树数据，避免在不同页面间频繁向 Go 后端发起请求。
*   (待创建) `taskStore.ts`: 可能用于记录当前是否有正在运行的异步打标任务，驱动右上角的跑马灯。

### 📁 `views/` (页面视图 - 计划中)
对应 `router/index.ts` 中的各个大页面。这里的内容将被渲染到 `Layout.vue` 的中央工作区中。
*   (待创建) `Dashboard.vue`: 首页统计。
*   (待创建) `DataSource.vue`: 用于拖拽上传 CSV/Excel 并预览数据表格的页面。
*   (待创建) `TagRuleConfig.vue`: 左右分栏页面，左边是标签树，右边是匹配规则构建器。
*   (待创建) `TaskKanban.vue`: 历史任务列表，支持查看日志和执行回退。
