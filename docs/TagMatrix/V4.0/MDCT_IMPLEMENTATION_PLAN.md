# V4.0 MDCT 算法实施规划

> **文档状态**：实施计划与任务拆解
> **相关需求**：基于 《多维共识打标算法 (MDCT) 架构蓝图》

我们将整个开发过程分为 **5 个核心阶段**，按照从底层数据、核心算法到上层业务集成的顺序推进，确保每一步都有明确的验收标准。

---

## 阶段一：数据模型与配置层升级（Model & Config Layer）
- [x] **任务 1.1**：修改**数据库实体/结构体**（如 `TaskResult`, `RecordTag` 等记录打标结果的表模型），新增以下字段：
  - `is_ai_intervened` (Boolean)：标识是否触发了 AI 介入。
  - `ai_arbitration_reason` (String/Text)：存储 AI 裁决的具体理由。
  - `confidence` (Float/String)：存储归一化后的置信度百分比。
- [x] **任务 1.2**：修改**全局设置/任务配置模型**，新增 W1~W4 权重配置字段，并设置推荐默认值（如 W1=1000, W2=10, W3=10, W4=100）。

## 阶段二：算法引擎底座与静态评分（Static Scoring Engine）
- [x] **任务 2.1**：在打标计算相关的 package 中（如 `internal/service/taglogic` 或 `taskengine`），新建 `mdct_scorer.go`，定义统一的 `RankScore` 核心计算函数及权重传参结构。
- [x] **任务 2.2**：实现 **W1 - PriorityScore** 的计算模块（提取 Rule 的优先级与 W1 加权）。
- [x] **任务 2.3**：实现 **W2 - ComplexityScore** 的计算模块（解析 AST 语法树 `matcher.go` 相关的规则条件结构，计算嵌套层级与算子确信度）。
- [x] **任务 2.4**：实现 **W3 - CompletenessScore** 的计算模块（扫描该行 record 数据的字段完整度进行打分）。

## 阶段三：AI 动态裁决与分数归一化（Dynamic AI & Normalization）
- [x] **任务 3.1**：在算法引擎中加入**“前置总分相近度检测逻辑”**（W1+W2+W3）。当发生碰撞且排名前两位的分数差值比例 `< 5%`，抛出熔断标识触发延迟计算。
- [x] **任务 3.2**：实现 **W4 - AIConsensusScore** 模块：
  - 联动 `aiengine_service.go` 的 LLM 接口。
  - 组装包含冲突标签和业务数据的 Prompt 模板（明确要求返回裁决结果及理由）。
  - 解析大模型的返回体，剥离出获胜规则和理由。
- [x] **任务 3.3**：实现**分数归一化算法 (Normalization)**，将发生碰撞的所有候选规则的绝对总分，转化为相加等于 100% 的置信度数组。

## 阶段四：引擎逻辑集成与接口调整（Engine Integration）
- [x] **任务 4.1**：重构 `taskengine_service.go` 中的 `processRecords` 排序机制。废弃旧版的 ID 对比，完整接入第二、三阶段封装好的 `MDCT 评分引擎`。
- [x] **任务 4.2**：更新 API 接口（如查询打标结果明细、获取任务详情等 API 的 Response 结构），将新加的 `IsAiIntervened`、`AiReason`、`Confidence` 暴露给前端。
- [x] **任务 4.3**：编写单元测试代码（涵盖悬殊分差直接出结果、相近分差触发 AI 的场景模拟）。

## 阶段五：前端体验重构（Frontend UI）
- [x] **任务 5.1**：**参数配置页改造**：添加 W1~W4 的自定义权重组件（带输入校验及“W1 应该保持绝对主导地位”的交互提示）。
- [x] **任务 5.2**：**数据结果展示页改造**：
  - 将命中标签的生硬展示改为带百分比的进度条/文本（如 `92.5%`）。
  - 针对 `is_ai_intervened == true` 的数据增加特殊徽标（Badge）。
  - 支持悬浮或点击查阅 AI 裁决理由（Tooltip / 弹窗）。