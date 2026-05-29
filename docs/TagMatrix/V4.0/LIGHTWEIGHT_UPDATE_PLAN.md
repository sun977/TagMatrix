# V4.0 轻量级在线升级提示 (Lightweight Update Prompt) 方案蓝图

> **文档状态**：功能演进规划
> **目标阶段**：V4.0 后续优化 / V4.1
> **核心议题**：以最低的开发成本和零系统风险，实现客户端版本的在线检测与升级引导。

---

## 1. 业务背景与方案选型

随着 TagMatrix 功能的不断迭代，我们需要一种机制通知旧版本用户进行升级。

传统的桌面软件热更新（OTA / Auto-Update）需要应用在后台下载二进制包，并利用操作系统底层 API 替换正在运行的自身。这种方式在 Windows 下极易触发“文件被占用”报错，在 macOS 下会遭遇严重的沙盒和签名隔离问题，开发成本高且容易导致软件崩溃。

**最终选型**：摒弃高风险的后台自替换逻辑，采用 **“后端检测版本 -> 前端弹窗提示 -> 引导系统浏览器至 GitHub 手动下载”** 的轻量级方案。

---

## 2. 方案实现全链路设计

整个闭环仅需三个极简步骤，完全复用现有架构和开源生态。

### 2.1 第一步：利用 GitHub API 进行版本自检 (Go 后端)

GitHub 官方免费提供了读取仓库 Release 信息的标准 API。我们不需要自己搭建版本校验服务器。

*   **执行时机**：在 Wails 启动的 `OnStartup` 钩子中，异步执行（不阻塞主界面的渲染）。
*   **请求接口**：发起 HTTP GET 请求到 `https://api.github.com/repos/您的用户名/TagMatrix/releases/latest`。
*   **逻辑处理**：
    1.  系统解析返回的 JSON 数据，提取出 `tag_name` 字段（例如 `"v4.1.0"`）和 `html_url` 字段（Release 下载页链接）。
    2.  将其与代码中硬编码的全局当前版本号（如 `CurrentVersion = "v4.0.0"`）进行比对。
    3.  如果发现 GitHub 上的版本号高于当前版本，说明有新版发布。

### 2.2 第二步：Wails 事件通信与前端弹窗 (Vue 前端)

*   **事件下发**：一旦 Go 后端检测到有新版本，立刻通过 Wails 的机制：
    `runtime.EventsEmit(ctx, "update_available", updateInfo)` 
    将包含版本号、更新日志和下载链接的信息推送给前端。
*   **UI 呈现**：前端（例如 `Layout.vue` 或全局入口处）监听到该事件，利用 Element Plus 的 Notification 或 MessageBox 弹出一个优雅的提示窗。
    *   **文案示例**：
        > 🚀 **发现新版本可用！**
        > 当前版本：V4.0.0
        > 最新版本：V4.1.0
        > 
        > 是否前往浏览器下载最新版本？
    *   提供两个按钮：`[前往下载]` 和 `[忽略本次]`。

### 2.3 第三步：系统原生浏览器唤起 (打通闭环)

*   当用户点击 `[前往下载]` 时，我们**不需要**在应用内内嵌浏览器。
*   直接利用 Wails 提供的原生操作系统能力，唤起用户的默认浏览器（Chrome / Edge 等）：
    ```go
    // Go 侧代码或前端直接绑定调用
    import "github.com/wailsapp/wails/v2/pkg/runtime"

    runtime.BrowserOpenURL(ctx, "https://github.com/您的用户名/TagMatrix/releases/latest")
    ```
*   浏览器被唤起并跳转到 GitHub 页面后，用户自行下载最新版的 `.exe` 或 `.app` 进行解压覆盖即可。

---

## 3. 方案优势总结

1.  **绝对零风险**：不涉及任何本地二进制文件的强行覆写操作，彻底杜绝了因升级失败导致软件变砖的灾难性后果。
2.  **零服务器/带宽成本**：纯粹“白嫖” GitHub 的 API 请求与对象存储能力，开发者无需维护静态资源服务器。
3.  **开发极其敏捷**：前后端打通预估只需几十行代码，无需引入任何第三方笨重的更新管理库（如 `go-update`），完美契合 TagMatrix 极致轻量、绿色开箱即用的产品定位。

## 4. 后续注意事项
1.  **限频控制**：为避免用户频繁遭遇弹窗打扰，可在前端 `localStorage` 或本地 SQLite 的设置表中记录“最后一次忽略更新的时间”。如果用户点击了“忽略本次”，一周内不要再重复弹窗。
2.  **网络连通性**：因 GitHub API 在国内存在偶发性访问不畅的情况。检测请求必须设置较短的超时时间（如 5 秒），一旦超时直接静默放弃，绝不能卡死主应用的启动流程。

## 5. 开发者发版操作指南 (Release Workflow)

为了使上述检测机制生效，开发者每次发布新版本时，需要遵循以下标准的 GitHub Release 流程：

### 5.1 准备工作
1. **修改代码版本号**：在 Go 代码（如 `app.go` 或 `config.go`）以及前端配置中，将 `CurrentVersion` 提升至新版本号（如 `"v4.1.0"`）。**注意：必须确保代码里的版本号与 GitHub Tag 完全一致，通常带上 `v` 前缀**。
2. **本地打包**：执行 `wails build`，生成对应平台的可执行文件（如 Windows 的 `.exe`，macOS 的 `.app` 压缩包）。

### 5.2 在 GitHub 上创建 Release
1. 进入您的 TagMatrix GitHub 仓库主页，点击右侧的 **Releases** 区域，然后点击 **"Draft a new release"**。
2. **Choose a tag (选择标签)**：输入并创建一个新的 Tag，例如 `v4.1.0`（需与代码里的版本号完全对应）。Target 选择 `main` 或主干分支。
3. **Release title (发布标题)**：输入本次发布的标题，例如 `TagMatrix V4.1.0 跨表视图重磅发布！`。
4. **Describe this release (版本描述)**：
   * 在这里填写本次更新的 ChangeLog（新增功能、Bug 修复等）。
   * **注意**：此处填写的内容，将被客户端通过 API 读取，并展示在用户的弹窗提示中。
5. **Attach binaries (上传附件)**：
   * 将第一步打包好的 `.exe` / `.zip` 等二进制文件拖拽上传到这里。
6. 点击 **"Publish release"**，完成发布。

### 5.3 闭环生效
发布成功后，GitHub 的 `https://api.github.com/repos/sun977/TagMatrix/releases/latest` 接口会自动更新。此时所有旧版 TagMatrix 客户端在启动时，便能瞬间感知到这个 `v4.1.0` 的存在，并向用户弹窗引导下载。
