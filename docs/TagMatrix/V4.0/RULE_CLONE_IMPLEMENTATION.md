# V4.0 规则克隆与继承功能 - 具体实施方案 (Implementation Plan)

## 一、 总体架构与依赖

本次功能更新属于“轻量级迭代”，不涉及数据库 Schema 变更，也不影响底层 Matcher 引擎。主要工作量集中在：
1. **后端 API 层**：新增两个克隆接口（单规则克隆、数据集批量继承）。
2. **前端 UI 层**：在 `TagRuleConfig.vue` 和 `DatasetList.vue` 中增加相应的按钮与交互逻辑。

---

## 二、 后端 API 实施细节

### 1. 核心工具函数：Schema 校验 (`CheckRuleSchema`)
在 `internal/service/taglogic` 下实现一个校验函数，用于递归提取 JSON 规则中的 `field` 并与目标数据集比对。

```go
// 校验规则是否兼容目标数据集表头
// 返回值：
// - status: "ok" (全匹配), "warning" (部分匹配), "error" (完全不匹配)
// - missingFields: 缺失的字段列表
func CheckRuleSchema(ruleJSON string, targetDatasetHeaders []string) (status string, missingFields []string) {
    // 1. 递归解析 JSON，提取所有 "field" 属性
    // 2. 将提取的 fields 与 targetDatasetHeaders 进行比对
    // 3. 返回校验结果
}
```

### 2. 接口 1：单规则克隆 API (`POST /api/v1/rules/clone`)
* **入参**：
  * `source_rule_id`: 源规则 ID
  * `target_dataset_id`: 目标数据集 ID
  * `tag_id`: 当前所属的标签 ID
* **逻辑**：
  1. 查询 `source_rule_id` 的完整 JSON 配置。
  2. 获取 `target_dataset_id` 的表头信息。
  3. 调用 `CheckRuleSchema` 进行校验。
  4. 如果 `status == "error"`，直接返回错误，中止操作。
  5. 否则，将 JSON 插入 `tag_rules` 表，绑定新的 `target_dataset_id` 和 `tag_id`。
* **返回值**：成功/失败状态，以及可能的 `warning` 提示和 `missingFields`。

### 3. 接口 2：数据集批量继承 API (`POST /api/v1/datasets/{id}/inherit-rules`)
* **入参**：
  * `source_dataset_id`: 源（模板）数据集 ID
  * `{id}`: 目标数据集 ID
* **逻辑**：
  1. 查出 `source_dataset_id` 下挂载的**所有规则列表**。
  2. 开启事务。遍历所有规则，逐一调用 `CheckRuleSchema`。
  3. 如果任何一条规则返回 `error`，或者整体缺失字段过多（比如超过 50%），提示用户风险并可选择回滚。
  4. 正常情况下，将所有规则深度拷贝，绑定到目标数据集。
* **返回值**：成功继承的规则数量，以及存在 `warning` 的规则列表。

---

## 三、 前端 UI 实施细节

### 1. 规则维度的单点克隆 (`views/TagRuleConfig.vue`)
* **位置**：右侧规则配置区的顶部，【添加条件】按钮右侧。
* **组件**：`<el-dropdown>` 触发的【从历史克隆】按钮。
* **交互逻辑**：
  1. 点击后，请求后端接口，拉取当前选中 `Tag` 在其他历史 `Dataset` 下配置过的记录。
  2. 下拉展示这些历史数据集的名称（如：“10月份流水记录”）。
  3. 用户点击某一项后，调用单规则克隆 API。
  4. 收到成功响应后，**前端强制刷新当前画布**，将拷贝过来的 JSON 树渲染出来。
  5. 如果 API 返回 `warning`，使用 `ElMessage.warning` 弹出提示，并在界面上用红色或黄色高亮对应的字段输入框（标记其为“无效字段”）。

### 2. 数据集维度的一键继承 (`views/DatasetList.vue`)
* **位置**：数据集列表的“操作”列。
* **组件**：`<el-button>` 触发的【继承历史规则】按钮。
* **交互逻辑**：
  1. 点击按钮，弹出一个 `<el-dialog>`，内部是一个选择器，让用户选择当前系统内已存在的其他数据集作为“模板”。
  2. 点击确认后，调用批量继承 API。
  3. API 可能会比较耗时，前端需要显示 Loading 状态。
  4. 成功后，弹窗提示“成功克隆了 15 条标签规则”，并刷新当前列表的状态。

---

## 四、 测试用例 (Test Cases)

1. **TC-01: 完美克隆**
   * 条件：源规则使用的所有字段，目标表头都有。
   * 预期：克隆成功，前端渲染出完整规则，无需修改即可直接测试。
2. **TC-02: 字段缺失预警**
   * 条件：源规则包含 3 个条件，其中 `ip_address` 字段在目标表头不存在。
   * 预期：克隆成功，但弹出警告：“缺少字段 ip_address”。用户保存时，如果没修正该字段，系统应给予二次拦截提醒。
3. **TC-03: 批量继承的隔离性**
   * 条件：数据集 A 继承 数据集 B 的规则。
   * 预期：继承完成后，修改数据集 A 的规则，**绝不能影响**数据集 B 原本的规则（验证深拷贝逻辑）。
