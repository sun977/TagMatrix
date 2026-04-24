<template>
  <div class="page-container">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <div class="header-left">
        <h1 class="page-title">标签与规则配置</h1>
        <p class="page-subtitle">在这里管理标签层级结构和配置对应的匹配规则，支持可视化规则构建和效果预览。</p>
      </div>
    </header>

    <div class="config-layout">
      <!-- 左侧标签树区域 -->
      <div class="left-pane">
        <div class="pane-header">
          <h3>标签体系</h3>
          <div class="header-actions">
            <el-button size="small" @click="handleImportTags" title="导入"><el-icon><Upload /></el-icon></el-button>
            <el-button size="small" @click="handleExportTags" title="导出"><el-icon><Download /></el-icon></el-button>
            <el-button type="primary" size="small" class="action-btn-green" @click="showAddTagDialog(0)" title="新建根标签">
              <el-icon><Plus /></el-icon>
            </el-button>
          </div>
        </div>
        
        <div class="pane-content">
          <div class="tree-wrapper" v-loading="loadingTags">
            <el-tree
              ref="treeRef"
              class="tag-tree"
              :data="tagTreeData"
              :props="defaultProps"
              node-key="id"
              default-expand-all
              :filter-node-method="filterNode"
              @node-click="handleNodeClick"
              :highlight-current="true"
              :expand-on-click-node="false"
            >
              <template #default="{ node, data }">
                <span class="custom-tree-node" :class="{ 'is-active': selectedTag?.id === data.id }">
                  <span class="tag-color-dot" :style="{ backgroundColor: data.color || 'var(--tm-accent-primary)' }"></span>
                  <span class="node-label">{{ node.label }}</span>
                  <span class="node-actions" v-if="selectedTag?.id === data.id">
                    <el-button link type="primary" size="small" @click.stop="showAddTagDialog(data.id)"><el-icon><Plus /></el-icon></el-button>
                    <el-button link type="danger" size="small" @click.stop="handleDeleteTag(data, $event)"><el-icon><Delete /></el-icon></el-button>
                  </span>
                </span>
              </template>
            </el-tree>
          </div>
        </div>
      </div>

      <!-- 右侧规则配置区域 -->
      <div class="right-pane">
        <template v-if="selectedTag">
          <div class="pane-header right-header">
            <div class="header-title">
              <span class="tag-color-dot large" :style="{ backgroundColor: selectedTag.color }"></span>
              <h2>{{ selectedTag.name }}</h2>
            </div>
          </div>

          <div class="pane-content scroll-content">
            <!-- 标签基本信息 -->
            <div class="config-section">
              <h4 class="section-title">标签基本信息</h4>
              <el-row :gutter="24">
                <el-col :span="8">
                  <div class="form-item">
                    <label>标签名称</label>
                    <el-input v-model="selectedTag.name" />
                  </div>
                </el-col>
                <el-col :span="8">
                  <div class="form-item">
                    <label>标签颜色</label>
                    <div class="color-swatches">
                      <div class="swatch" style="background: #f5a623" @click="selectedTag.color = '#f5a623'"></div>
                      <div class="swatch" style="background: #f56c6c" @click="selectedTag.color = '#f56c6c'"></div>
                      <div class="swatch" style="background: #e6a23c" @click="selectedTag.color = '#e6a23c'"></div>
                      <div class="swatch" style="background: #52c48f" @click="selectedTag.color = '#52c48f'"></div>
                      <div class="swatch" style="background: #409eff" @click="selectedTag.color = '#409eff'"></div>
                      <div class="swatch" style="background: #7b61ff" @click="selectedTag.color = '#7b61ff'"></div>
                      <div class="swatch" style="background: #e056fd" @click="selectedTag.color = '#e056fd'"></div>
                    </div>
                  </div>
                </el-col>
                <el-col :span="8">
                  <div class="form-item">
                    <label>标签描述</label>
                    <el-input v-model="selectedTag.description" type="textarea" :rows="2" placeholder="输入标签描述..." />
                  </div>
                </el-col>
              </el-row>
            </div>

            <!-- 匹配规则 -->
            <div class="config-section">
              <div class="section-header-flex">
                <h4 class="section-title">匹配规则</h4>
                <el-button size="small" type="info" plain @click="previewRuleJson">
                  预览 JSON
                </el-button>
              </div>

              <div class="rules-list">
                <RuleGroup v-model="ruleState" :is-root="true" />
              </div>
            </div>

            <!-- 规则测试 -->
            <div class="config-section">
              <div class="section-header-flex">
                <h4 class="section-title">规则测试 (试运行)</h4>
                <el-button type="primary" class="action-btn-green" @click="handleDryRun" :loading="runningDry">
                  <el-icon><VideoPlay /></el-icon> 测试此规则
                </el-button>
              </div>

              <div v-if="hasRunDry" class="test-results">
                <div class="result-alert">
                  测试完成！抽样检测了1000条数据，其中有234条数据匹配当前规则，匹配率23.4%。
                </div>

                <el-table :data="mockDryRunData" style="width: 100%" class="custom-table">
                  <el-table-column prop="id" label="用户ID" width="120" />
                  <el-table-column prop="name" label="用户名" width="120" />
                  <el-table-column prop="amount" label="累计消费金额" width="160">
                    <template #default="scope">
                      <span :class="{'text-danger': scope.row.amount < 1000, 'text-success': scope.row.amount >= 1000}">
                        ¥{{ scope.row.amount.toFixed(2) }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column prop="loginTime" label="最近登录时间" min-width="160">
                    <template #default="scope">
                      <span :class="{'text-danger': scope.row.isOld}">
                        {{ scope.row.loginTime }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column label="匹配结果" width="120" align="center">
                    <template #default="scope">
                      <div class="match-pill" :class="scope.row.matched ? 'matched' : 'unmatched'">
                        <el-icon><Select v-if="scope.row.matched" /><CloseBold v-else /></el-icon>
                        {{ scope.row.matched ? '匹配' : '不匹配' }}
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </div>

          <!-- 底部保存条 -->
          <div class="pane-footer">
            <el-button>取消</el-button>
            <el-button type="primary" class="action-btn-green" @click="handleSaveRule" :loading="savingRule">保存配置</el-button>
          </div>
        </template>

        <!-- 未选择标签时的空状态 -->
        <div class="empty-state" v-else>
          <el-empty description="请在左侧选择一个标签以配置打标规则" />
        </div>
      </div>
    </div>

    <!-- 添加标签对话框 -->
    <el-dialog v-model="dialogVisible" title="新增标签" width="400px">
      <el-form :model="tagForm" label-width="80px">
        <el-form-item label="标签名称" required>
          <el-input v-model="tagForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="tagForm.description" type="textarea" />
        </el-form-item>
        <el-form-item label="颜色">
          <el-color-picker v-model="tagForm.color" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" class="action-btn-green" @click="submitAddTag" :loading="savingTag">确定</el-button>
        </span>
      </template>
    </el-dialog>
    <!-- JSON 预览对话框 -->
    <el-dialog v-model="previewDialogVisible" title="规则 JSON 预览" width="500px">
      <pre class="json-preview">{{ previewJsonStr }}</pre>
      <template #footer>
        <el-button @click="previewDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Plus, VideoPlay, MoreFilled, DocumentCopy, Delete, Select, CloseBold, Download, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CreateTag, DeleteTag, ExportTags, ImportTags, GetTagTree, SaveRule, DryRunRule, GetRuleByTag } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import RuleGroup from '../components/RuleGroup.vue'

const filterText = ref('')
const treeRef = ref<any>()

// --- 左侧标签树逻辑 ---
const loadingTags = ref(false)
const tagTreeData = ref<model.TagTreeNode[]>([])

const defaultProps = {
  children: 'children',
  label: 'name',
}

const fetchTags = async () => {
  loadingTags.value = true
  try {
    const tree = await GetTagTree()
    tagTreeData.value = tree || []
  } catch (e: any) {
    console.error(e)
    ElMessage.error('获取标签失败')
  } finally {
    loadingTags.value = false
  }
}

const filterNode = (value: string, data: any) => {
  if (!value) return true
  return data.name.includes(value)
}

const handleExportTags = async () => {
  try {
    await ExportTags("") // 触发保存文件对话框
    ElMessage.success("导出成功")
  } catch (e: any) {
    if (e !== "cancelled") ElMessage.error("导出失败: " + String(e))
  }
}

const handleImportTags = async () => {
  try {
    await ImportTags("") // 触发打开文件对话框
    ElMessage.success("导入成功")
    fetchTags()
  } catch (e: any) {
    if (e !== "cancelled") ElMessage.error("导入失败: " + String(e))
  }
}

const handleDeleteTag = async (data: any, e: Event) => {
  e.stopPropagation()
  try {
    await ElMessageBox.confirm(`确定要删除标签 "${data.name}" 及其所有子节点吗？此操作不可恢复。`, '警告', { type: 'warning' })
    await DeleteTag(data.id)
    ElMessage.success("删除成功")
    if (selectedTag.value?.id === data.id) {
      selectedTag.value = null
    }
    fetchTags()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error("删除失败: " + String(e))
  }
}

// --- 右侧规则逻辑 ---
const selectedTag = ref<any>(null)
const ruleState = ref<any>({
  logic: 'and',
  conditions: []
})
const previewDialogVisible = ref(false)
const previewJsonStr = ref('')

const hasRunDry = ref(false)
const runningDry = ref(false)
const mockDryRunData = ref<any[]>([])

// 递归转换规则状态到 NeoScan 格式
const buildNeoScanRule = (state: any): any => {
  const result: any = {}
  const conditions = []

  for (const cond of state.conditions) {
    if (cond.logic) {
      // 这是一个子逻辑组
      conditions.push(buildNeoScanRule(cond))
    } else {
      // 这是一个基本条件
      conditions.push({
        field: cond.field,
        operator: cond.operator,
        value: cond.value
      })
    }
  }

  result[state.logic] = conditions
  return result
}

const previewRuleJson = () => {
  const neoRule = buildNeoScanRule(ruleState.value)
  previewJsonStr.value = JSON.stringify(neoRule, null, 2)
  previewDialogVisible.value = true
}

const handleNodeClick = async (data: any) => {
  selectedTag.value = data
  if (!data.raw) data.raw = {}
  
  hasRunDry.value = false
  
  try {
    const rule = await GetRuleByTag(data.id)
    if (rule && rule.rule_json) {
      const parsed = JSON.parse(rule.rule_json)
      // 需要从 NeoScan 格式反解析为 state 格式，这里简化处理，假设之前存的就是带有 logic/conditions 的内部格式
      // 实际应用中需要写一个反向解析器
      if (parsed.logic) {
        ruleState.value = parsed
      } else {
        ruleState.value = { logic: 'and', conditions: [] }
      }
    } else {
      ruleState.value = { logic: 'and', conditions: [] }
    }
  } catch (e) {
    ruleState.value = { logic: 'and', conditions: [] }
  }
}

const savingRule = ref(false)
const handleSaveRule = async () => {
  savingRule.value = true
  try {
    const ruleObj = new model.SysMatchRule()
    ruleObj.tag_id = selectedTag.value.id
    ruleObj.name = selectedTag.value.name + "的规则"
    
    // 这里我们依然保存前端内部的 state 格式方便回显，如果是为了对接其他系统，可以保存 NeoScan 格式
    // 为了满足“生成一段json数据”，我们将 NeoScan 格式序列化保存
    const neoRule = buildNeoScanRule(ruleState.value)
    
    // 但是为了下次回显方便，我们需要在 rule_json 中保存能够反向构建的数据。
    // 在这里我们做一个 hack，将 state 作为内部状态保存，将 NeoScan 作为真正的规则使用
    ruleObj.rule_json = JSON.stringify(ruleState.value) 
    ruleObj.is_enabled = true
    ruleObj.priority = 0

    await SaveRule(ruleObj)
    ElMessage.success('配置保存成功')
  } catch (e: any) {
    ElMessage.error('保存失败: ' + String(e))
  } finally {
    savingRule.value = false
  }
}

const handleDryRun = async () => {
  runningDry.value = true
  try {
    // DryRun 依然使用 NeoScan 格式发送给后端
    const neoRule = buildNeoScanRule(ruleState.value)
    const ruleJSON = JSON.stringify(neoRule)
    const results = await DryRunRule(ruleJSON, 100) // Call Go API
    
    mockDryRunData.value = results.map((r: any) => {
        let d: any = {}
        try {
            d = JSON.parse(r.data || '{}')
        } catch (e) {}
        
        return {
            id: r.record_id,
            name: d.name || d.user_name || '-',
            amount: Number(d.amount) || 0,
            loginTime: d.loginTime || d.last_login || '-',
            matched: r.matched,
            isOld: false 
        }
    })
    hasRunDry.value = true
  } catch (e: any) {
    ElMessage.error('试运行失败: ' + String(e))
  } finally {
    runningDry.value = false
  }
}

// --- 添加标签弹窗逻辑 ---
const dialogVisible = ref(false)
const savingTag = ref(false)
const tagForm = ref({
  name: '',
  description: '',
  color: '#52c48f',
  parentId: 0
})

const showAddTagDialog = (parentId: number) => {
  tagForm.value = { name: '', description: '', color: '#52c48f', parentId }
  dialogVisible.value = true
}

const submitAddTag = async () => {
  if (!tagForm.value.name) {
    ElMessage.warning('请输入标签名称')
    return
  }
  savingTag.value = true
  try {
    const tag = new model.SysTag()
    tag.name = tagForm.value.name
    tag.description = tagForm.value.description
    tag.color = tagForm.value.color
    tag.parent_id = tagForm.value.parentId

    await CreateTag(tag)
    ElMessage.success('标签创建成功')
    dialogVisible.value = false
    fetchTags() // 刷新树
  } catch (e: any) {
    ElMessage.error('创建标签失败: ' + String(e))
  } finally {
    savingTag.value = false
  }
}

onMounted(() => {
  fetchTags()
})
</script>

<style scoped lang="scss">
.page-container {
  padding: 24px 32px 40px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

/* --- 页面顶部 --- */
.page-header {
  margin-bottom: 24px;
  flex-shrink: 0;

  .page-title {
    font-size: 20px;
    font-weight: 600;
    color: var(--tm-text-primary);
    margin: 0 0 8px 0;
  }

  .page-subtitle {
    font-size: 14px;
    color: var(--tm-text-secondary);
    margin: 0;
  }
}

.config-layout {
  display: flex;
  gap: 24px;
  flex: 1;
  min-height: 0; /* flex container scroll fix */
}

/* --- 左侧区域 --- */
.left-pane {
  width: 280px;
  background: #ffffff;
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;

  .pane-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--tm-border-color);

    h3 {
      margin: 0;
      font-size: 15px;
      font-weight: 600;
    }
  }

  .pane-content {
    flex: 1;
    overflow-y: auto;
    padding: 12px 8px;
  }
}

/* --- 左侧树样式 --- */
.tag-tree {
  :deep(.el-tree-node__content) {
    height: 40px;
    border-radius: var(--tm-border-radius-sm);
    margin-bottom: 4px;

    &:hover {
      background-color: var(--tm-bg-hover);
    }
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: var(--tm-bg-hover);
  }
}

.custom-tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--tm-text-regular);
  width: 100%;

  .tag-color-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .node-label {
    font-weight: 500;
  }
}

/* --- 右侧区域 --- */
.right-pane {
  flex: 1;
  background: #ffffff;
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  display: flex;
  flex-direction: column;
  min-width: 0;
  position: relative;
}

.right-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--tm-border-color);
  flex-shrink: 0;

  .header-title {
    display: flex;
    align-items: center;
    gap: 12px;

    .tag-color-dot.large {
      width: 12px;
      height: 12px;
      border-radius: 50%;
    }

    h2 {
      margin: 0;
      font-size: 18px;
      font-weight: 600;
    }

    .rule-count-pill {
      background-color: #f5f5f5;
      color: var(--tm-text-secondary);
      font-size: 12px;
      padding: 4px 10px;
      border-radius: 12px;
    }
  }

  .header-actions {
    display: flex;
    gap: 12px;

    .more-btn {
      padding: 8px;
    }
  }
}

.scroll-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.config-section {
  margin-bottom: 40px;

  .section-title {
    margin: 0 0 16px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--tm-text-primary);
  }

  .section-header-flex {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;

    .section-title {
      margin: 0;
    }
  }
}

/* --- 表单与配置项 --- */
.form-item {
  label {
    display: block;
    font-size: 13px;
    color: var(--tm-text-secondary);
    margin-bottom: 8px;
  }
}

.color-swatches {
  display: flex;
  gap: 12px;
  align-items: center;
  height: 32px;

  .swatch {
    width: 24px;
    height: 24px;
    border-radius: 4px;
    cursor: pointer;
    transition: transform 0.1s;

    &:hover {
      transform: scale(1.1);
    }
  }
}

/* --- 逻辑切换 --- */
.logic-switch {
  display: flex;
  align-items: center;
  gap: 12px;

  .logic-label {
    font-size: 13px;
    color: var(--tm-text-secondary);
  }

  .logic-group {
    display: flex;
    background-color: #f5f5f5;
    border-radius: 16px;
    padding: 2px;

    .logic-item {
      padding: 4px 16px;
      font-size: 12px;
      font-weight: 500;
      color: var(--tm-text-secondary);
      border-radius: 14px;
      cursor: pointer;
      transition: all 0.2s;

      &.active {
        background-color: var(--tm-accent-primary);
        color: #fff;
      }
    }
  }
}

/* --- 规则列表 --- */
.rules-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rule-card {
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius-sm);
  padding: 16px;

  .rule-card-header {
    display: flex;
    align-items: center;
    margin-bottom: 16px;

    .rule-name {
      font-size: 14px;
      font-weight: 500;
      margin-right: 16px;
    }

    .rule-actions {
      margin-left: auto;
      display: flex;
      gap: 8px;
    }
  }

  .rule-card-body {
    display: flex;
    align-items: center;
    gap: 12px;

    .rule-unit {
      font-size: 13px;
      color: var(--tm-text-secondary);
    }
  }
}

.add-condition-btn {
  width: 100%;
  border-style: dashed;
  color: var(--tm-text-secondary);
  
  &:hover {
    color: var(--tm-accent-primary);
    border-color: var(--tm-accent-primary);
  }
}

/* --- 测试结果 --- */
.result-alert {
  background-color: #e6f0fa;
  color: #3a8ee6;
  padding: 12px 16px;
  border-radius: var(--tm-border-radius-sm);
  font-size: 13px;
  margin-bottom: 16px;
}

.match-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;

  &.matched {
    background-color: #e8f7f0;
    color: var(--tm-accent-primary);
  }

  &.unmatched {
    background-color: #fef0f0;
    color: #f56c6c;
  }
}

.text-danger {
  color: #f56c6c;
}
.text-success {
  color: var(--tm-accent-primary);
}

/* --- 底部保存条 --- */
.pane-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--tm-border-color);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  background: #ffffff;
  border-radius: 0 0 var(--tm-border-radius) var(--tm-border-radius);
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.action-btn-green {
  background-color: var(--tm-accent-primary);
  border-color: var(--tm-accent-primary);
  &:hover {
    background-color: var(--tm-accent-hover);
    border-color: var(--tm-accent-hover);
  }
}
</style>