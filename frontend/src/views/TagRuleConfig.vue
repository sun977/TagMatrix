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
                  <div class="spacer"></div>
                  <el-tooltip v-if="data.has_rule" content="该标签已在部分数据集中配置规则" placement="right">
                    <el-icon class="has-rule-icon" style="color: #67c23a; margin-left: 8px;"><Check /></el-icon>
                  </el-tooltip>
                </span>
              </template>
            </el-tree>
          </div>
        </div>
      </div>

      <!-- 右侧规则配置区域 -->
      <div class="right-pane" style="display: flex; flex-direction: column; gap: 16px; background-color: transparent; padding: 0;">
        <template v-if="selectedTag">
          <!-- 标签信息详情卡片 -->
          <el-card class="tag-info-card" shadow="never" style="border-radius: 8px; border: 1px solid var(--tm-border-light);">
            <template #header>
              <div class="card-header" style="display: flex; justify-content: space-between; align-items: center;">
                <div style="display: flex; align-items: center;">
                  <span class="tag-color-dot large" :style="{ backgroundColor: selectedTag.color }"></span>
                  <h2 style="margin: 0 0 0 12px; font-size: 18px; font-weight: 600;">{{ selectedTag.name }}</h2>
                </div>
                <el-button type="primary" class="action-btn-green" @click="handleUpdateTag" :loading="updatingTag" size="small">更新标签</el-button>
              </div>
            </template>
            <div class="config-section" style="margin-bottom: 0;">
              <div style="display: flex; gap: 24px; align-items: center; flex-wrap: wrap;">
                <div class="form-item inline" style="display: flex; align-items: center; gap: 8px; margin-bottom: 0;">
                  <label style="margin-bottom: 0; white-space: nowrap;">标签名称</label>
                  <el-input v-model="selectedTag.name" style="width: 200px;" />
                </div>
                <div class="form-item inline" style="display: flex; align-items: center; gap: 8px; margin-bottom: 0;">
                  <label style="margin-bottom: 0; white-space: nowrap;">标签颜色</label>
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
                <div class="form-item inline" style="display: flex; align-items: center; gap: 8px; margin-bottom: 0; flex: 1; min-width: 200px;">
                  <label style="margin-bottom: 0; white-space: nowrap;">标签描述</label>
                  <el-input v-model="selectedTag.description" placeholder="输入标签描述..." />
                </div>
              </div>
            </div>
          </el-card>

          <!-- 标签对应的规则列表卡片 -->
          <el-card class="rule-info-card" shadow="never" style="flex: 1; display: flex; flex-direction: column; border-radius: 8px; border: 1px solid var(--tm-border-light);">
            <template #header>
              <div class="card-header" style="margin-bottom: 0; display: flex; justify-content: space-between; align-items: center;">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <h3 style="margin: 0; font-size: 16px; font-weight: 600;">打标规则列表</h3>
                  <el-tooltip content="每个标签在一个数据集中只能配置一套规则" placement="top">
                    <el-icon class="help-icon" style="cursor: pointer; color: #909399;"><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
                <div style="display: flex; gap: 12px; align-items: center;">
                  <el-button type="primary" class="action-btn-green" @click="showAddRuleDialog" size="small">
                    <el-icon><Plus /></el-icon> 新增规则
                  </el-button>
                </div>
              </div>
            </template>

            <div class="scroll-content" style="flex: 1; overflow-y: auto;">
              <div v-if="rulesList && rulesList.length > 0" style="display: flex; flex-direction: column; gap: 16px;">
                <el-card v-for="rule in rulesList" :key="rule.id" shadow="hover" style="border: 1px solid var(--tm-border-color);">
                  <template #header>
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                      <span style="font-weight: 500;">针对【{{ getDatasetName(rule.dataset_id) }}】的规则：{{ rule.name || '未命名规则' }}</span>
                      <div>
                        <el-button size="small" type="primary" link @click="editRule(rule)">编辑</el-button>
                        <el-button size="small" type="danger" link @click="deleteRule(rule)">删除</el-button>
                      </div>
                    </div>
                  </template>
                  <div style="color: #606266; font-size: 13px;">
                    规则配置预览: 
                    <pre style="margin: 8px 0 0 0; padding: 8px; background: #f5f7fa; border-radius: 4px; max-height: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ rule.rule_json }}</pre>
                  </div>
                </el-card>
              </div>
              <el-empty v-else description="该标签暂无配置任何打标规则" />
            </div>
          </el-card>
        </template>

        <!-- 未选择标签时的空状态 -->
        <div class="empty-state" v-else style="background: #fff; border-radius: 8px; border: 1px solid var(--tm-border-light); height: 100%; display: flex; align-items: center; justify-content: center;">
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

    <!-- 规则配置弹窗 -->
    <el-dialog v-model="ruleDialogVisible" width="850px" top="5vh">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px;">
          <span class="el-dialog__title">{{ currentRuleId ? '编辑规则' : '新增规则' }}</span>
          <el-tooltip content="点击查看匹配算子说明" placement="top">
            <el-icon class="help-icon" @click="operatorHelpVisible = true" style="cursor: pointer; color: #909399;"><QuestionFilled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <div style="display: flex; flex-direction: column; gap: 20px;">
        <div style="display: flex; gap: 24px;">
          <div class="form-item inline" style="display: flex; align-items: center; gap: 12px; margin-bottom: 0;">
            <label style="margin-bottom: 0; white-space: nowrap; font-weight: 500;">目标数据集</label>
            <el-select v-model="ruleDatasetId" placeholder="请选择数据集" style="width: 200px;" :disabled="!!currentRuleId">
              <el-option v-for="ds in availableDatasets" :key="ds.id" :label="ds.name" :value="ds.id" />
            </el-select>
          </div>
          <div class="form-item inline" style="display: flex; align-items: center; gap: 12px; margin-bottom: 0;">
            <label style="margin-bottom: 0; white-space: nowrap; font-weight: 500;">规则名称</label>
            <el-input v-model="ruleName" :placeholder="selectedTag?.name + '-Rule'" style="width: 200px;" />
          </div>
          <div style="flex: 1; text-align: right;">
            <el-button size="small" type="info" plain @click="previewRuleJson">预览 JSON</el-button>
          </div>
        </div>

        <div class="rules-list" style="max-height: 40vh; overflow-y: auto; padding-right: 8px;">
          <RuleGroup v-model="ruleState" :is-root="true" :schema-keys="currentSchemaKeys" />
        </div>

        <!-- 规则测试 -->
        <div class="config-section" style="border-top: 1px dashed var(--tm-border-light); padding-top: 16px;">
          <div class="section-header-flex" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
            <h4 class="section-title" style="margin: 0;">规则测试 (试运行)</h4>
            <div style="display: flex; gap: 12px; align-items: center;">
              <el-select v-model="testLimit" placeholder="测试数据范围" size="small" style="width: 150px">
                <el-option label="前 1000 条数据" :value="1000" />
                <el-option label="全库数据" :value="0" />
              </el-select>
              <el-button type="primary" class="action-btn-green" @click="handleDryRun" :loading="runningDry" size="small" :disabled="!ruleDatasetId">
                <el-icon><VideoPlay /></el-icon> 测试此规则
              </el-button>
            </div>
          </div>

          <div v-if="hasRunDry" class="test-results">
            <div class="result-alert" style="margin-bottom: 12px; padding: 8px 12px; background-color: #f0f9eb; color: #67c23a; border-radius: 4px; font-size: 13px;">
              测试完成！抽样检测了 {{ testSummary.total }} 条数据，其中有 {{ testSummary.matched }} 条数据匹配当前规则，匹配率 {{ testSummary.ratio }}%。
            </div>

            <el-table :data="mockDryRunData" style="width: 100%" class="custom-table" max-height="250">
              <el-table-column label="匹配结果" width="100" align="center" fixed="left" prop="_matched" sortable>
                <template #default="scope">
                  <div class="match-pill" :class="scope.row._matched ? 'matched' : 'unmatched'">
                    <el-icon><Select v-if="scope.row._matched" /><CloseBold v-else /></el-icon>
                    {{ scope.row._matched ? '匹配' : '不匹配' }}
                  </div>
                </template>
              </el-table-column>
              <el-table-column v-for="col in dynamicColumns" :key="col" :prop="col" :label="col" min-width="150" show-overflow-tooltip />
            </el-table>
          </div>
        </div>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="ruleDialogVisible = false">取消</el-button>
          <el-button type="primary" class="action-btn-green" @click="handleSaveRule" :loading="savingRule">保存配置</el-button>
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

    <!-- 操作符帮助弹窗 -->
    <el-dialog v-model="operatorHelpVisible" title="匹配算子 (操作符) 说明" width="800px">
      <el-table :data="operatorHelpData" style="width: 100%" border size="small" height="500">
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column prop="operator" label="操作符" width="150">
          <template #default="scope">
            <code>{{ scope.row.operator }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="desc" label="说明" width="120" />
        <el-table-column prop="example" label="示例">
          <template #default="scope">
            <span class="example-text">{{ scope.row.example }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { Plus, VideoPlay, MoreFilled, DocumentCopy, Delete, Select, CloseBold, Download, Upload, QuestionFilled, Filter, Check } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CreateTag, DeleteTag, UpdateTag, ExportTags, ImportTags, GetTagTree, SaveRule, DryRunRule, GetRulesByTag, CheckTagHasRules, ListDatasets, DeleteRule } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import RuleGroup from '../components/RuleGroup.vue'

const filterText = ref('')
const treeRef = ref<any>()

// --- 左侧标签树逻辑 ---
const loadingTags = ref(false)
const tagTreeData = ref<model.TagTreeNode[]>([])

// --- 数据集列表 ---
const availableDatasets = ref<any[]>([])

const fetchDatasets = async () => {
  try {
    const list = await ListDatasets()
    availableDatasets.value = list || []
  } catch (e: any) {
    console.error('Failed to fetch datasets:', e)
  }
}

const getDatasetName = (datasetId: number) => {
  const ds = availableDatasets.value.find(d => d.id === datasetId)
  return ds ? ds.name : '未知数据集'
}


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
    const hasRules = await CheckTagHasRules(data.id)
    
    let confirmMessage = `确定要删除标签 "${data.name}" 及其所有子节点吗？此操作不可恢复。`
    if (hasRules) {
      confirmMessage += '<br><br><strong style="color:#f56c6c;">⚠️ 警告：该标签或其子标签已经配置了匹配规则，继续删除将永久销毁这些规则且不可恢复！</strong>'
    }

    await ElMessageBox.confirm(
      confirmMessage,
      '警告',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true,
      }
    )
    
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
const currentRuleId = ref<number | null>(null)
const ruleName = ref<string>('')
const ruleDatasetId = ref<number | null>(null)
const currentSchemaKeys = computed(() => {
  if (!ruleDatasetId.value) return []
  const ds = availableDatasets.value.find(d => d.id === ruleDatasetId.value)
  if (ds && ds.schema_keys) {
    try {
      return JSON.parse(ds.schema_keys)
    } catch(e) {
      return []
    }
  }
  return []
})
const updatingTag = ref(false)
const ruleState = ref<any>({
  logic: 'and',
  conditions: []
})
const previewDialogVisible = ref(false)
const previewJsonStr = ref('')
const operatorHelpVisible = ref(false)
const ruleDialogVisible = ref(false)
const rulesList = ref<any[]>([])

const showAddRuleDialog = () => {
  currentRuleId.value = null
  ruleName.value = ''
  ruleDatasetId.value = null
  ruleState.value = { logic: 'and', conditions: [] }
  ruleDialogVisible.value = true
}

const parseNeoScanRule = (neoRule: any): any => {
  if (!neoRule) return { logic: 'and', conditions: [] }
  if (neoRule.and) {
    return { logic: 'and', conditions: neoRule.and.map(parseNeoScanRule) }
  } else if (neoRule.or) {
    return { logic: 'or', conditions: neoRule.or.map(parseNeoScanRule) }
  } else if (neoRule.field !== undefined) {
    // Leaf node
    const value = (neoRule.operator && ['in', 'not_in', 'list_contains'].includes(neoRule.operator) && Array.isArray(neoRule.value))
      ? neoRule.value.join(', ')
      : neoRule.value
    return {
      field: neoRule.field,
      operator: neoRule.operator,
      value: value,
      ignore_case: neoRule.ignore_case
    }
  }
  return { logic: 'and', conditions: [] }
}

const editRule = (rule: any) => {
  currentRuleId.value = rule.id
  ruleName.value = rule.name || ''
  ruleDatasetId.value = rule.dataset_id
  try {
    const parsed = JSON.parse(rule.rule_json)
    // 如果是旧的带 logic 的格式（直接使用），否则解析 NeoScan 格式
    if (parsed.logic) {
      ruleState.value = parsed
    } else {
      ruleState.value = parseNeoScanRule(parsed)
    }
  } catch (e) {
    ruleState.value = { logic: 'and', conditions: [] }
  }
  ruleDialogVisible.value = true
}

const deleteRule = async (rule: any) => {
  try {
    await ElMessageBox.confirm(`确定要删除规则 "${rule.name}" 吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await DeleteRule(rule.id)
    ElMessage.success('规则删除成功')
    if (selectedTag.value) {
      const rules = await GetRulesByTag(selectedTag.value.id)
      rulesList.value = rules || []
    }
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('删除失败: ' + String(e))
  }
}


const operatorHelpData = [
  { category: '基础比较', operator: 'equals', desc: '等于', example: '字段的值完全等于目标值 (支持字符串/数字)' },
  { category: '基础比较', operator: 'not_equals', desc: '不等于', example: '字段的值不等于目标值' },
  { category: '基础比较', operator: 'exists', desc: '存在', example: '该字段在数据中存在 (无论值是否为空)，不需要输入目标值' },
  { category: '基础比较', operator: 'is_null', desc: '为空', example: '该字段不存在或值为 null' },
  { category: '基础比较', operator: 'is_not_null', desc: '不为空', example: '该字段存在且值不为 null' },
  
  { category: '文本匹配', operator: 'contains', desc: '包含', example: '"hello world" 包含 "world"' },
  { category: '文本匹配', operator: 'not_contains', desc: '不包含', example: '"hello" 不包含 "x"' },
  { category: '文本匹配', operator: 'starts_with', desc: '以...开头', example: '"server-01" 以 "server" 开头' },
  { category: '文本匹配', operator: 'ends_with', desc: '以...结尾', example: '"image.png" 以 ".png" 结尾' },
  { category: '文本匹配', operator: 'regex', desc: '正则匹配', example: '正则表达式匹配，如 ^192\\.168\\..*' },
  { category: '文本匹配', operator: 'like', desc: '模糊匹配', example: '类似SQL，支持 % (任意字符) 和 _ (单个字符)' },

  { category: '数值/大小', operator: 'greater_than', desc: '大于', example: 'count > 10 (也支持字符串字典序比较)' },
  { category: '数值/大小', operator: 'less_than', desc: '小于', example: 'count < 10' },
  { category: '数值/大小', operator: 'greater_than_or_equal', desc: '大于等于', example: 'count >= 10' },
  { category: '数值/大小', operator: 'less_than_or_equal', desc: '小于等于', example: 'count <= 10' },

  { category: '集合/特殊', operator: 'in', desc: '在列表中', example: '目标值用逗号分隔，如: admin, root' },
  { category: '集合/特殊', operator: 'not_in', desc: '不在列表中', example: '目标值用逗号分隔，如: guest, user' },
  { category: '集合/特殊', operator: 'list_contains', desc: '列表包含', example: '原始数据如果是数组 [1, 2]，列表包含 1' },
  { category: '集合/特殊', operator: 'cidr', desc: 'IP网段', example: '判断 IP "192.168.1.5" 是否属于网段 "192.168.1.0/24"' },
]

const hasRunDry = ref(false)
const runningDry = ref(false)
const mockDryRunData = ref<any[]>([])
const testLimit = ref<number>(1000)
const dynamicColumns = ref<string[]>([])
const testSummary = ref({ total: 0, matched: 0, ratio: '0.0' })

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
        // 对 in / not_in 等特殊操作符进行逗号切割转换为数组
        value: ['in', 'not_in', 'list_contains'].includes(cond.operator) && typeof cond.value === 'string'
          ? cond.value.split(',').map((s: string) => s.trim())
          : cond.value,
        ignore_case: cond.ignore_case ? true : undefined
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
  currentRuleId.value = null // 重置当前规则ID
  ruleName.value = '' // 重置规则名称
  rulesList.value = []
  
  try {
    const rules = await GetRulesByTag(data.id)
    rulesList.value = rules || []
  } catch (e) {
    rulesList.value = []
  }
}

const handleUpdateTag = async () => {
  if (!selectedTag.value) return
  updatingTag.value = true
  try {
    const tag = new model.SysTag()
    tag.id = selectedTag.value.id
    tag.name = selectedTag.value.name
    tag.color = selectedTag.value.color
    tag.description = selectedTag.value.description
    
    await UpdateTag(tag)
    ElMessage.success('标签更新成功')
    fetchTags() // 刷新树以显示最新信息
  } catch (e: any) {
    ElMessage.error('更新失败: ' + String(e))
  } finally {
    updatingTag.value = false
  }
}

const savingRule = ref(false)
const handleSaveRule = async () => {
  if (!ruleDatasetId.value) {
    ElMessage.warning('请选择目标数据集')
    return
  }

  savingRule.value = true
  try {
    const ruleObj = new model.SysMatchRule()
    if (currentRuleId.value) {
      ruleObj.id = currentRuleId.value
    }
    ruleObj.tag_id = selectedTag.value.id
    ruleObj.dataset_id = ruleDatasetId.value
    ruleObj.name = ruleName.value || (selectedTag.value.name + "-Rule")
    
    // 将 Vue 规则状态转换为后端支持的 NeoScan 格式
    const neoRule = buildNeoScanRule(ruleState.value)
    ruleObj.rule_json = JSON.stringify(neoRule) 
    ruleObj.is_enabled = true
    ruleObj.priority = 0

    await SaveRule(ruleObj)
    ElMessage.success('配置保存成功')
    ruleDialogVisible.value = false
    
    try {
      const updatedRules = await GetRulesByTag(selectedTag.value.id)
      rulesList.value = updatedRules || []
    } catch (ignore) {}
  } catch (e: any) {
    ElMessage.error('保存失败: ' + String(e))
  } finally {
    savingRule.value = false
  }
}

const handleDryRun = async () => {
  if (!ruleState.value.conditions || ruleState.value.conditions.length === 0) {
    ElMessage.warning('当前没有配置任何匹配规则，请先添加规则后再进行测试')
    return
  }

  runningDry.value = true
  try {
    // DryRun 依然使用 NeoScan 格式发送给后端
    const neoRule = buildNeoScanRule(ruleState.value)
    const ruleJSON = JSON.stringify(neoRule)
    const results = await DryRunRule(ruleJSON, testLimit.value) // Call Go API
    
    if (!results || results.length === 0) {
      ElMessage.warning('当前数据库中没有可供试运行的目标数据，请先导入数据！')
      mockDryRunData.value = []
      dynamicColumns.value = []
      testSummary.value = { total: 0, matched: 0, ratio: '0.0' }
      hasRunDry.value = true
      return
    }

    let matchedCount = 0
    const columnsSet = new Set<string>()

    mockDryRunData.value = results.map((r: any) => {
      if (r.matched) matchedCount++
      
      let d: any = {}
      try {
        d = JSON.parse(r.data || '{}')
        // 收集所有出现过的动态列
        Object.keys(d).forEach(k => {
          if (k !== '_source_sheet') { // 过滤掉内部字段
            columnsSet.add(k)
          }
        })
      } catch (e) {}
      
      return {
        _matched: r.matched,
        ...d
      }
    })

    dynamicColumns.value = Array.from(columnsSet)
    testSummary.value = {
      total: results.length,
      matched: matchedCount,
      ratio: results.length > 0 ? ((matchedCount / results.length) * 100).toFixed(1) : '0.0'
    }

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
  fetchDatasets()
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
  flex: 1;
  padding-right: 8px;

  .tag-color-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .node-label {
    font-weight: 500;
  }

  .node-actions {
    display: flex;
    align-items: center;
  }

  .spacer {
    flex: 1;
  }

  .has-rule-icon {
    color: var(--tm-accent-primary);
    font-size: 16px;
    font-weight: bold;
    opacity: 0.8;
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
    
    .help-icon {
      cursor: pointer;
      color: #909399;
      font-size: 16px;
      transition: color 0.2s;
      
      &:hover {
        color: var(--tm-accent-primary);
      }
    }
    
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
.example-text {
  font-family: monospace;
  background-color: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--tm-text-regular);
}
</style>