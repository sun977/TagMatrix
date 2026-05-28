<template>
  <div class="page-container">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <div class="header-left">
        <h1 class="page-title">任务管理</h1>
        <p class="page-subtitle">在这里发起和管理打标任务，查看执行进度和历史记录，支持任务回滚和日志查看。</p>
      </div>
    </header>

    <!-- 发起新任务区域 -->
    <div class="launch-section">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
        <h3 class="section-title" style="margin-bottom: 0;">发起新的打标任务</h3>
        <el-button type="primary" class="action-btn-green" @click="submitTask" :loading="isSubmitting" :disabled="!taskForm.datasetId">
          <el-icon><VideoPlay /></el-icon> 开始执行任务
        </el-button>
      </div>
      <div class="launch-form">
        <el-row :gutter="24">
          <el-col :span="4">
            <div class="form-item">
              <label>任务名称</label>
              <el-input v-model="taskForm.batchName" placeholder="请输入任务名称" />
            </div>
          </el-col>
          <el-col :span="4">
            <div class="form-item">
              <label>目标数据集 <span style="color: red">*</span></label>
              <el-select v-model="taskForm.datasetId" placeholder="请选择数据集" class="w-100" @change="handleDatasetChange">
                <el-option
                  v-for="ds in availableDatasets"
                  :key="ds.id"
                  :label="ds.name"
                  :value="ds.id"
                />
              </el-select>
            </div>
          </el-col>
<el-col :span="4">
<div class="form-item">
<label>选择来源文件</label>
<el-select v-model="taskForm.sourceFile" multiple collapse-tags collapse-tags-tooltip placeholder="请选择来源文件" class="w-100" @change="handleSourceFileChange">
<el-option :label="`全量库内数据 (${totalRecords}条)`" value="all" />
<el-option v-for="ds in availableSourceFiles" :key="ds.source_name" :label="`${ds.source_name} (${ds.count}条)`" :value="ds.source_name" />
</el-select>
</div>
</el-col>
          <el-col :span="4">
            <div class="form-item">
              <label>选择要执行的标签规则</label>
              <el-select v-model="taskForm.rules" multiple collapse-tags collapse-tags-tooltip placeholder="请选择规则" class="w-100" :disabled="!taskForm.datasetId" @change="handleRuleChange">
                <el-option label="全部生效规则" value="all" />
                <el-option v-for="rule in availableRules" :key="rule.id" :label="rule.name" :value="String(rule.id)" />
              </el-select>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="form-item">
              <label>执行策略</label>
              <el-select v-model="taskForm.execStrategy" class="w-100">
                <el-option label="追加模式 (保留原有标签)" value="append" />
                <el-option label="覆盖模式 (清除原有标签)" value="overwrite" />
              </el-select>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="form-item">
              <label style="display: flex; align-items: center; gap: 4px;">
                打标模式
                <el-tooltip effect="dark" placement="top" :hide-after="0" popper-class="tag-mode-tooltip">
                  <template #content>
                    <div style="line-height: 1.6; max-width: 350px;">
                      <div style="margin-bottom: 4px;"><b>多标签模式</b>：数据命中几条规则，就打上几个平级的标签。</div>
                      <div style="margin-bottom: 4px;"><b>单标签模式</b>：命中多条规则时，根据MDCT算法仅取优最优的一个标签入库。</div>
                      <div style="margin-bottom: 4px;"><b>混合模式</b>：命中的所有标签均入库，根据MDCT算法选取最优的一个设为主标签。</div>
                      <div style="margin-top: 8px; color: #a0cfff;">
                        <i>注：若开启MDCT的算法AI介入裁决功能，在遇到标签冲突的时候将由AI智能判断选出主标签。</i>
                      </div>
                    </div>
                  </template>
                  <el-icon style="cursor: pointer; color: var(--el-text-color-secondary);"><QuestionFilled /></el-icon>
                </el-tooltip>
              </label>
              <el-select v-model="taskForm.tagMode" class="w-100">
                <el-option label="多标签模式(默认)" value="multiple" />
                <el-option label="单标签模式" value="single" />
                <el-option label="混合模式" value="mixed" />
              </el-select>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="form-item">
              <label>任务描述 (选填)</label>
              <el-input v-model="taskForm.desc" placeholder="输入任务描述" />
            </div>
          </el-col>
        </el-row>
      </div>
    </div>

    <!-- 任务历史区域 -->
    <div class="history-section">
      <div class="section-header">
        <div class="section-title-wrapper" style="display: flex; align-items: center; gap: 16px;">
          <h3 class="section-title" style="margin-bottom: 0;">任务历史</h3>
          <el-button 
            v-if="selectedTaskIds.length > 0" 
            type="danger" 
            size="small" 
            @click="handleBatchDelete"
          >
            批量删除 ({{ selectedTaskIds.length }})
          </el-button>
        </div>
        <div class="header-filters">
          <el-select v-model="filterDataset" class="filter-select">
            <el-option label="全部数据集" value="all" />
            <el-option 
              v-for="ds in availableDatasets" 
              :key="ds.id" 
              :label="ds.name" 
              :value="ds.id" 
            />
          </el-select>
          <el-select v-model="filterStatus" class="filter-select">
            <el-option label="全部状态" value="all" />
            <el-option label="执行中" value="running" />
            <el-option label="已完成" value="completed" />
            <el-option label="失败" value="failed" />
          </el-select>
          <el-select v-model="filterTime" class="filter-select">
            <el-option label="全部时间" value="all" />
            <el-option label="近7天" value="7d" />
            <el-option label="近30天" value="30d" />
          </el-select>
          <el-button circle @click="fetchBatches" :loading="loadingBatches">
            <el-icon><RefreshRight /></el-icon>
          </el-button>
          
          <el-popover
            placement="bottom-end"
            title="展示列设置"
            :width="200"
            trigger="click"
          >
            <template #reference>
              <el-button circle>
                <el-icon><Setting /></el-icon>
              </el-button>
            </template>
            <div class="column-settings">
              <el-checkbox 
                v-for="col in allToggleableColumns" 
                :key="col" 
                :model-value="!hiddenColumns.includes(col)"
                @change="toggleColumn(col)"
              >
                {{ col }}
              </el-checkbox>
            </div>
          </el-popover>
        </div>
      </div>

      <el-table 
        :data="paginatedTaskHistory" 
        style="width: 100%" 
        height="100%"
        class="custom-table" 
        v-loading="loadingBatches"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" fixed="left" />
        <el-table-column prop="name" label="任务名称" min-width="180" fixed="left" />
        <el-table-column v-if="!hiddenColumns.includes('任务描述')" prop="desc" label="任务描述" min-width="150" show-overflow-tooltip />
        <el-table-column v-if="!hiddenColumns.includes('目标数据集')" label="目标数据集" min-width="120">
          <template #default="scope">
            {{ getDatasetName(scope.row.datasetId) }}
          </template>
        </el-table-column>
        <el-table-column v-if="!hiddenColumns.includes('文件来源')" prop="sourceFile" label="文件来源" min-width="120" show-overflow-tooltip />
        <el-table-column v-if="!hiddenColumns.includes('生效规则')" prop="rules" label="生效规则" min-width="180" show-overflow-tooltip />
        <el-table-column v-if="!hiddenColumns.includes('执行策略')" prop="execStrategy" label="执行策略" min-width="100" />
        <el-table-column v-if="!hiddenColumns.includes('打标模式')" prop="tagMode" label="打标模式" min-width="100" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="scope">
            <div class="status-pill" :class="scope.row.statusType">
              <span class="dot" v-if="scope.row.statusType !== 'running'"></span>
              <el-icon v-else class="is-loading" style="margin-right: 4px; font-size: 14px; position: relative; top: 1px;"><Loading /></el-icon>
              {{ scope.row.statusText }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="执行进度" min-width="200">
          <template #default="scope">
            <div class="progress-wrapper">
              <el-progress 
                :percentage="scope.row.progress" 
                :show-text="false" 
                :color="getProgressColor(scope.row.statusType)" 
                :stroke-width="8"
              />
              <span class="progress-text">{{ scope.row.progress }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column v-if="!hiddenColumns.includes('处理数据量')" prop="processed" label="处理数据量" width="160" />
        <el-table-column v-if="!hiddenColumns.includes('耗时')" prop="time" label="耗时" width="100" />
        <el-table-column v-if="!hiddenColumns.includes('创建人')" prop="creator" label="创建人" width="100" />
        <el-table-column v-if="!hiddenColumns.includes('创建时间')" prop="createTime" label="创建时间" width="160" />
        <el-table-column width="140" align="center" fixed="right">
          <template #header>
            <div style="display: flex; align-items: center; justify-content: center;">
              操作
              <el-tooltip effect="dark" placement="top-end">
            <template #content>
              <div style="line-height: 1.8;">
                <div><strong>查看</strong>：查看打标任务详细日志及命中规则。</div>
                <div><strong>导出</strong>：将该批次产生的打标日志导出为 CSV。</div>
                <div><strong style="color: #67C23A;">回退</strong>：撤销本次打标任务所产生的所有标签。</div>
                <div><strong style="color: #409EFF;">重试</strong>：复制原任务参数至表单并重新发起。</div>
                <div><strong style="color: #F56C6C;">删除</strong>：彻底删除该任务及其所有相关日志记录。</div>
              </div>
            </template>
                <el-icon style="font-size: 14px; margin-left: 4px; color: #909399; cursor: help;"><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
          </template>
          <template #default="scope">
            <template v-if="scope.row.statusType === 'running'">
              <div style="display: flex; gap: 4px; justify-content: center;">
                <el-button size="small" class="action-btn">查看详情</el-button>
                <el-button type="danger" link size="small">终止</el-button>
              </div>
            </template>
            <template v-else-if="scope.row.statusType === 'completed'">
              <div style="display: flex; gap: 4px; justify-content: center; margin-bottom: 6px;">
                <el-button size="small" class="action-btn" @click="viewLogs(scope.row.id)" style="margin: 0;">查看</el-button>
                <el-button size="small" class="action-btn" @click="exportLogs(scope.row.id)" style="margin: 0;">导出</el-button>
              </div>
              <div style="display: flex; gap: 4px; justify-content: center;">
          <el-button size="small" class="action-btn btn-text-green" @click="handleRollback(scope.row.id)" style="margin: 0;">回退</el-button>
          <el-button size="small" class="action-btn btn-text-red" @click="handleSingleDelete(scope.row.id)" style="margin: 0;">删除</el-button>
        </div>
      </template>
      <template v-else-if="scope.row.statusType === 'failed'">
        <div style="display: flex; gap: 4px; justify-content: center; margin-bottom: 6px;">
          <el-button type="danger" link size="small" style="margin: 0;">错误日志</el-button>
        </div>
        <div style="display: flex; gap: 4px; justify-content: center;">
          <el-button size="small" class="action-btn btn-text-blue" @click="cloneTask(scope.row)" style="margin: 0;">重试</el-button>
          <el-button size="small" class="action-btn btn-text-red" @click="handleSingleDelete(scope.row.id)" style="margin: 0;">删除</el-button>
        </div>
      </template>
      <template v-else-if="scope.row.statusType === 'rolled_back'">
        <div style="display: flex; gap: 4px; justify-content: center; margin-bottom: 6px;">
          <el-button size="small" class="action-btn" @click="viewLogs(scope.row.id)" style="margin: 0;">查看</el-button>
          <el-button size="small" class="action-btn" @click="exportLogs(scope.row.id)" style="margin: 0;">导出</el-button>
        </div>
        <div style="display: flex; gap: 4px; justify-content: center;">
          <el-button size="small" class="action-btn btn-text-blue" @click="cloneTask(scope.row)" style="margin: 0;">重试</el-button>
          <el-button size="small" class="action-btn btn-text-red" @click="handleSingleDelete(scope.row.id)" style="margin: 0;">删除</el-button>
        </div>
      </template>
            <template v-else-if="scope.row.statusType === 'pending'">
              <div style="display: flex; gap: 4px; justify-content: center; margin-bottom: 6px;">
                <el-button size="small" class="action-btn" style="margin: 0;">编辑</el-button>
                <el-button type="success" size="small" class="action-btn retry-btn" style="margin: 0;">执行</el-button>
              </div>
              <div style="display: flex; gap: 4px; justify-content: center;">
                <el-button type="danger" link size="small" @click="handleSingleDelete(scope.row.id)" style="margin: 0;">删除</el-button>
              </div>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          :current-page="currentPage"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="filteredTaskHistory.length"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>

    <!-- 查看日志弹窗 -->
    <el-dialog
      v-model="logDialogVisible"
      title="打标任务日志"
      width="70%"
      destroy-on-close
    >
      <el-table :data="taskLogs" style="width: 100%" max-height="500px" v-loading="loadingLogs">
        <el-table-column prop="recordId" label="数据ID" width="100" />
        <el-table-column prop="tagName" label="标签名称" width="150" />
        <el-table-column prop="ruleName" label="命中规则" width="180" />
        <el-table-column prop="action" label="操作" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.action === 'add' ? 'success' : 'danger'" size="small">
              {{ scope.row.action === 'add' ? '添加' : '移除' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="匹配原因" min-width="250" />
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>
      <div class="pagination-container" style="margin-top: 15px; display: flex; justify-content: flex-end;">
        <el-pagination
          background
          layout="total, sizes, prev, pager, next, jumper"
          :total="logTotal"
          :page-sizes="[10, 20, 50, 100]"
          :page-size="logPageSize"
          :current-page="logCurrentPage"
          @size-change="handleLogSizeChange"
          @current-change="handleLogCurrentChange"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { VideoPlay, RefreshRight, QuestionFilled, Loading, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { GetTaskBatches, RunTaggingTask, RollbackTask, GetDashboardStats, GetTaskLogs, GetTaskLogsPaged, ExportTaskLogsCSV, DeleteTaskBatches, GetAvailableSourceFiles, ListDatasets, GetRulesByDataset, GetDatasetTotalRecords } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const allToggleableColumns = [
  '任务描述',
  '目标数据集',
  '文件来源',
  '生效规则',
  '执行策略',
  '打标模式',
  '处理数据量',
  '耗时',
  '创建人',
  '创建时间'
]
const hiddenColumns = ref<string[]>([]) // 默认显示所有列

const toggleColumn = (col: string) => {
  const idx = hiddenColumns.value.indexOf(col)
  if (idx > -1) {
    hiddenColumns.value.splice(idx, 1)
  } else {
    hiddenColumns.value.push(col)
  }
}

const loadingBatches = ref(false)

const totalRecords = ref(0)
const availableRules = ref<model.SysMatchRule[]>([])
const availableSourceFiles = ref<model.SourceFileOption[]>([])
const availableDatasets = ref<model.SysDataset[]>([])

const taskForm = ref({
  batchName: '',
  datasetId: undefined as number | undefined,
  sourceFile: ['all'],
  rules: ['all'],
  execStrategy: 'append',
  tagMode: 'multiple',
  desc: ''
})

const handleSourceFileChange = (val: string[]) => {
  if (!val || val.length === 0) {
    taskForm.value.sourceFile = ['all']
    return
  }
  const lastSelected = val[val.length - 1]
  if (lastSelected === 'all') {
    taskForm.value.sourceFile = ['all']
  } else if (val.includes('all')) {
    taskForm.value.sourceFile = val.filter(v => v !== 'all')
  }
}

const handleRuleChange = (val: string[]) => {
  if (!val || val.length === 0) {
    taskForm.value.rules = ['all']
    return
  }
  const lastSelected = val[val.length - 1]
  if (lastSelected === 'all') {
    taskForm.value.rules = ['all']
  } else if (val.includes('all')) {
    taskForm.value.rules = val.filter(item => item !== 'all')
  }
}

const handleDatasetChange = async () => {
  taskForm.value.rules = ['all']
  taskForm.value.sourceFile = ['all']
  availableRules.value = []
  availableSourceFiles.value = []
  if (!taskForm.value.datasetId) {
    totalRecords.value = 0
    return
  }

  // 重新获取该数据集下的规则
  try {
    const sources = await GetAvailableSourceFiles(taskForm.value.datasetId)
    availableSourceFiles.value = sources || []
    
		const rules = await GetRulesByDataset(taskForm.value.datasetId)
		availableRules.value = rules || []

		const totalCount = await GetDatasetTotalRecords(taskForm.value.datasetId)
		totalRecords.value = totalCount
	} catch (e: any) {
		ElMessage.error('加载数据集相关信息失败: ' + String(e))
	}
}

const filterDataset = ref<string | number>('all')
const filterStatus = ref('all')
const filterTime = ref('all')
const isSubmitting = ref(false)

// 真实任务历史数据
const taskHistory = ref<any[]>([])

const currentPage = ref(1)
const pageSize = ref(10)

const handleSizeChange = (val: number) => {
  pageSize.value = val
  currentPage.value = 1
}

const handleCurrentChange = (val: number) => {
  currentPage.value = val
}

const filteredTaskHistory = computed(() => {
  let result = [...taskHistory.value]

  // 过滤数据集
  if (filterDataset.value !== 'all') {
    result = result.filter(item => item.datasetId === filterDataset.value)
  }

  // 过滤状态
  if (filterStatus.value !== 'all') {
    result = result.filter(item => {
      if (filterStatus.value === 'completed') {
        return item.statusType === 'completed' || item.statusType === 'rolled_back'
      }
      return item.statusType === filterStatus.value
    })
  }

  // 过滤时间
  const now = Date.now()
  if (filterTime.value === '7d') {
    result = result.filter(item => {
      return (now - item.rawTime) <= 7 * 24 * 60 * 60 * 1000
    })
  } else if (filterTime.value === '30d') {
    result = result.filter(item => {
      return (now - item.rawTime) <= 30 * 24 * 60 * 60 * 1000
    })
  }

  // 时间倒序排序（最新的在前面）
  return result.sort((a, b) => b.rawTime - a.rawTime)
})

const getDatasetName = (id: number) => {
  const ds = availableDatasets.value.find(d => d.id === id)
  return ds ? ds.name : `未知数据集(${id})`
}

const paginatedTaskHistory = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredTaskHistory.value.slice(start, end)
})

watch([filterDataset, filterStatus, filterTime], () => {
  currentPage.value = 1
})

// 日志弹窗相关状态
const logDialogVisible = ref(false)
const loadingLogs = ref(false)
const taskLogs = ref<model.TagTaskLogDto[]>([])
const logTotal = ref(0)
const logCurrentPage = ref(1)
const logPageSize = ref(20)
const currentLogBatchId = ref<number>(0)

// 批量删除相关
const selectedTaskIds = ref<number[]>([])

const handleSelectionChange = (selection: any[]) => {
  selectedTaskIds.value = selection.map(item => item.id)
}

const handleBatchDelete = async () => {
  if (selectedTaskIds.value.length === 0) return

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedTaskIds.value.length} 个打标任务吗？相关的日志和打标记录将被彻底清除，该操作不可恢复！`,
      '批量删除任务',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    loadingBatches.value = true
    await DeleteTaskBatches(selectedTaskIds.value)
    ElMessage.success('批量删除成功')
    selectedTaskIds.value = []
    await fetchBatches()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('批量删除失败: ' + String(e))
    }
  } finally {
    loadingBatches.value = false
  }
}

const handleSingleDelete = async (id: number) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除该打标任务吗？相关的日志和打标记录将被彻底清除，该操作不可恢复！`,
      '删除任务',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    loadingBatches.value = true
    await DeleteTaskBatches([id])
    ElMessage.success('删除成功')
    await fetchBatches()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败: ' + String(e))
    }
  } finally {
    loadingBatches.value = false
  }
}

const fetchLogs = async () => {
  loadingLogs.value = true
  try {
    const result = await GetTaskLogsPaged(currentLogBatchId.value, logCurrentPage.value, logPageSize.value)
    taskLogs.value = result.logs || []
    logTotal.value = result.total || 0
  } catch (e: any) {
    ElMessage.error('获取日志失败: ' + String(e))
  } finally {
    loadingLogs.value = false
  }
}

const viewLogs = async (batchId: number) => {
  currentLogBatchId.value = batchId
  logCurrentPage.value = 1
  logDialogVisible.value = true
  await fetchLogs()
}

const handleLogSizeChange = (val: number) => {
  logPageSize.value = val
  logCurrentPage.value = 1
  fetchLogs()
}

const handleLogCurrentChange = (val: number) => {
  logCurrentPage.value = val
  fetchLogs()
}

const exportLogs = async (batchId: number) => {
  try {
    const filepath = await ExportTaskLogsCSV(batchId)
    if (filepath) {
      ElMessage.success(`导出成功: ${filepath}`)
    }
  } catch (e: any) {
    ElMessage.error('导出日志失败: ' + String(e))
  }
}

const loadData = async () => {
  try {
    const ds = await ListDatasets()
    availableDatasets.value = ds || []
  } catch (e: any) {
    console.error('Failed to load datasets:', e)
  }
}

const fetchBatches = async () => {
  loadingBatches.value = true
  try {
    const batches = await GetTaskBatches()
    taskHistory.value = batches.map((b: model.TagTaskBatch) => {
      const isRunning = b.status === 'running'
      const isCompleted = b.status === 'completed' || b.status === 'rolled_back'
      const isFailed = b.status === 'failed'
      
      let statusText = '未知'
      if (isRunning) statusText = '执行中'
      else if (isCompleted) statusText = b.status === 'rolled_back' ? '已回退' : '已完成'
      else if (isFailed) statusText = '失败'

      let timeStr = '-'
      if (b.finished_at && b.created_at) {
        const diff = new Date(b.finished_at).getTime() - new Date(b.created_at).getTime()
        if (diff >= 0) {
          const seconds = Math.floor(diff / 1000)
          if (seconds < 60) timeStr = `${seconds > 0 ? seconds : '<1'}秒`
          else if (seconds < 3600) timeStr = `${Math.floor(seconds/60)}分${seconds%60}秒`
          else timeStr = `${Math.floor(seconds/3600)}小时${Math.floor((seconds%3600)/60)}分`
        }
      } else if (isRunning && b.created_at) {
        // 如果还在运行中，计算已运行时间
        const diff = Date.now() - new Date(b.created_at).getTime()
        if (diff >= 0) {
          const seconds = Math.floor(diff / 1000)
          if (seconds < 60) timeStr = `${seconds > 0 ? seconds : '<1'}秒`
          else if (seconds < 3600) timeStr = `${Math.floor(seconds/60)}分${seconds%60}秒`
          else timeStr = `${Math.floor(seconds/3600)}小时${Math.floor((seconds%3600)/60)}分`
        }
      }

      return {
        id: b.id,
        name: b.name,
        desc: b.desc || '-',
        sourceFile: b.source_file || '全部记录',
        rules: b.rules || '-',
        execStrategy: b.exec_strategy === 'overwrite' ? '覆盖模式' : '追加模式',
        tagMode: b.tag_mode === 'single' ? '单标签模式' : (b.tag_mode === 'mixed' ? '混合模式' : '多标签模式'),
        datasetId: b.dataset_id,
        statusType: b.status,
        statusText: statusText,
        progress: isCompleted ? 100 : (isRunning ? 0 : 0), // 运行中的进度交给WebSocket推送
        processed: `${b.total_processed} 条`,
        time: timeStr,
        creator: '系统',
        createTime: new Date(b.created_at || Date.now()).toLocaleString(),
        rawTime: new Date(b.created_at || Date.now()).getTime(),
        rawSourceFile: b.source_file,
        rawRules: b.rules,
        rawExecStrategy: b.exec_strategy,
        rawTagMode: b.tag_mode
      }
    })
  } catch (e: any) {
    ElMessage.error('获取任务历史失败: ' + String(e))
  } finally {
    loadingBatches.value = false
  }
}

const cloneTask = async (row: any) => {
  taskForm.value.datasetId = row.datasetId
  await handleDatasetChange()
  
  let parsedSources = ['all']
  try {
    if (row.rawSourceFile) {
      const parsed = JSON.parse(row.rawSourceFile)
      if (Array.isArray(parsed) && parsed.length > 0) {
        parsedSources = parsed
      } else if (typeof row.rawSourceFile === 'string') {
        parsedSources = row.rawSourceFile.split(',').filter((s: string) => s.trim() !== '')
        if (parsedSources.length === 0) parsedSources = ['all']
      }
    }
  } catch(e) {
    if (typeof row.rawSourceFile === 'string') {
      parsedSources = row.rawSourceFile.split(',').filter((s: string) => s.trim() !== '')
      if (parsedSources.length === 0) parsedSources = ['all']
    }
  }
  taskForm.value.sourceFile = parsedSources

  let parsedRules = ['all']
  try {
    if (row.rawRules) {
      const parsed = JSON.parse(row.rawRules)
      if (Array.isArray(parsed) && parsed.length > 0) {
        parsedRules = parsed.map((id: any) => String(id))
      } else if (typeof row.rawRules === 'string') {
        parsedRules = row.rawRules.split(',').filter((s: string) => s.trim() !== '')
        if (parsedRules.length === 0) parsedRules = ['all']
      }
    }
  } catch(e) {
     if (typeof row.rawRules === 'string') {
       parsedRules = row.rawRules.split(',').filter((s: string) => s.trim() !== '')
       if (parsedRules.length === 0) parsedRules = ['all']
     }
  }
  taskForm.value.rules = parsedRules
  
  taskForm.value.execStrategy = row.rawExecStrategy || 'append'
  taskForm.value.tagMode = row.rawTagMode || 'multiple'
  taskForm.value.desc = row.desc === '-' ? '' : row.desc
  
  taskForm.value.batchName = ''
  
  window.scrollTo({ top: 0, behavior: 'smooth' })
  ElMessage.success('已复制任务配置，请检查后重新发起')
}

const submitTask = async () => {
  if (!taskForm.value.batchName) {
    const datasetName = taskForm.value.datasetId ? getDatasetName(taskForm.value.datasetId) : '未命名数据集'
    const now = new Date()
    const pad = (n: number) => n.toString().padStart(2, '0')
    const dateStr = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
    const randomStr = Math.random().toString(36).substring(2, 5).toUpperCase()
    
    taskForm.value.batchName = `${datasetName}_${dateStr}_${randomStr}`
  }
  isSubmitting.value = true
  
  try {
    let ruleIDs: number[] = []
    if (taskForm.value.rules.includes('all')) {
      const rules = await GetRulesByDataset(taskForm.value.datasetId!)
      ruleIDs = rules.map(r => r.id)
    } else {
      ruleIDs = taskForm.value.rules.map(r => parseInt(r)).filter(id => !isNaN(id))
    }

    const isOverwrite = taskForm.value.execStrategy === 'overwrite'

    await RunTaggingTask(
      taskForm.value.datasetId!,
      ruleIDs,
      taskForm.value.batchName,
      taskForm.value.desc,
      isOverwrite,
      taskForm.value.tagMode,
      taskForm.value.sourceFile.includes('all') ? [] : taskForm.value.sourceFile
    )
    ElMessage.success(`任务提交成功`)
    
    taskForm.value.batchName = ''
    fetchBatches()
  } catch (e: any) {
    ElMessage.error('提交失败: ' + String(e))
  } finally {
    isSubmitting.value = false
  }
}

const handleRollback = async (batchId: number) => {
  try {
    await RollbackTask(batchId)
    ElMessage.success('回退成功')
    fetchBatches()
  } catch (e: any) {
    ElMessage.error('回退失败: ' + String(e))
  }
}

const getProgressColor = (statusType: string) => {
  if (statusType === 'running') return '#52c48f'
  if (statusType === 'completed') return '#3a8ee6'
  if (statusType === 'failed') return '#f56c6c'
  if (statusType === 'rolled_back') return '#909399'
  return '#e4e7ed'
}

onMounted(() => {
  fetchBatches()
  loadData()

  EventsOn('task_list_updated', fetchBatches)

  // 监听后端推送的任务进度事件
  EventsOn('taskProgress', (data: any) => {
    const batchIndex = taskHistory.value.findIndex(b => b.id === data.batchID)
    if (batchIndex !== -1) {
      const batch = taskHistory.value[batchIndex]
      const oldStatus = batch.statusType
      batch.statusType = data.status
      
      let statusText = '未知'
      if (data.status === 'running') statusText = '执行中'
      else if (data.status === 'completed') statusText = '已完成'
      else if (data.status === 'rolled_back') statusText = '已回退'
      else if (data.status === 'failed') statusText = '失败'

      batch.statusText = statusText
      batch.progress = data.progress
      batch.processed = `${data.processed} 条` // data.total 如果需要可以拼接

      if (data.status === 'running') {
        const diff = Date.now() - batch.rawTime
        if (diff >= 0) {
          const seconds = Math.floor(diff / 1000)
          if (seconds < 60) batch.time = `${seconds > 0 ? seconds : '<1'}秒`
          else if (seconds < 3600) batch.time = `${Math.floor(seconds/60)}分${seconds%60}秒`
          else batch.time = `${Math.floor(seconds/3600)}小时${Math.floor((seconds%3600)/60)}分`
        }
      }

      taskHistory.value[batchIndex] = { ...batch }
      
      if (oldStatus === 'running' && (data.status === 'completed' || data.status === 'failed')) {
        // Fetch to get real finished_at time
        fetchBatches()
      }
    } else {
      // 也有可能是新创建的任务（刚发起还没重新fetch的）
      if (data.status === 'running' && data.progress === 0) {
        fetchBatches()
      }
    }
  })
})

onUnmounted(() => {
  EventsOff('taskProgress')
  EventsOff('task_list_updated')
})
</script>

<style scoped lang="scss">
.column-settings {
  display: flex;
  flex-direction: column;
  max-height: 300px;
  overflow-y: auto;
  padding-right: 8px;
}

.column-settings::-webkit-scrollbar {
  width: 6px;
}
.column-settings::-webkit-scrollbar-thumb {
  background-color: var(--tm-border-color);
  border-radius: 3px;
}

.page-container {
  padding: 24px 32px 24px;
  flex: 1;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  min-height: 0;
}

/* --- 页面顶部 --- */
.page-header {
  margin-bottom: 16px;
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

/* --- 发起新任务区域 --- */
.launch-section {
  background: var(--tm-bg-main);
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  padding: 20px 24px;
  margin-bottom: 16px;
  flex-shrink: 0;

  .section-title {
    margin: 0 0 20px 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--tm-text-primary);
  }

  .launch-form {
    .form-item {
      label {
        display: block;
        font-size: 13px;
        color: var(--tm-text-secondary);
        margin-bottom: 8px;
      }
      .w-100 {
        width: 100%;
      }
    }

    .form-actions {
      display: flex;
      justify-content: flex-end;
      margin-top: 16px;
    }
  }
}

/* --- 任务历史区域 --- */
.history-section {
  background: var(--tm-bg-main);
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  padding: 16px 24px 20px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0; /* Important for nested flex scroll */

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    flex-shrink: 0;

    .section-title {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--tm-text-primary);
    }

    .header-filters {
      display: flex;
      gap: 12px;

      .filter-select {
        width: 120px;
      }
    }
  }
}

/* --- 表格样式 --- */
.custom-table {
  --el-table-border-color: transparent;
  --el-table-header-bg-color: var(--tm-bg-subtle);
  --el-table-header-text-color: var(--tm-text-secondary);
  flex: 1;
  min-height: 0;
  
  :deep(th.el-table__cell) {
    font-weight: 500;
    padding: 12px 0;
  }
  
  :deep(td.el-table__cell) {
    padding: 16px 0;
    font-size: 14px;
    color: var(--tm-text-regular);
    border-bottom: 1px solid var(--tm-border-color);
  }
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;

  .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }

  &.running {
    background-color: var(--tm-accent-light);
    color: var(--tm-accent-primary);
    .dot { background-color: var(--tm-accent-primary); }
  }

  &.completed {
    background-color: rgba(58, 142, 230, 0.1);
    color: #3a8ee6;
    .dot { background-color: #3a8ee6; }
  }

  &.failed {
    background-color: rgba(245, 108, 108, 0.1);
    color: #f56c6c;
    .dot { background-color: #f56c6c; }
  }

  &.rolled_back {
    background-color: var(--tm-bg-hover);
    color: #909399;
    .dot { background-color: #909399; }
  }

  &.pending {
    background-color: rgba(230, 162, 60, 0.1);
    color: #e6a23c;
    .dot { background-color: #e6a23c; }
  }
}

.progress-wrapper {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 160px;

  :deep(.el-progress-bar__outer) {
    background-color: var(--tm-bg-subtle);
  }

  .progress-text {
    font-size: 12px;
    color: var(--tm-text-secondary);
  }
}

.action-btn {
  border-color: var(--tm-border-color);
  color: var(--tm-text-regular);
  border-radius: 6px;

  &:hover {
    color: var(--tm-text-primary);
    border-color: #dcdfe6;
    background-color: var(--tm-bg-hover);
  }
  
  &.retry-btn {
    background-color: var(--tm-accent-primary);
    border-color: var(--tm-accent-primary);
    color: white;
    
    &:hover {
      background-color: var(--tm-accent-hover);
      border-color: var(--tm-accent-hover);
    }
  }

  &.btn-text-green {
    color: #67C23A;
    &:hover {
      color: #85ce61;
      border-color: #dcdfe6;
      background-color: var(--tm-bg-hover);
    }
  }

  &.btn-text-blue {
    color: #409EFF;
    &:hover {
      color: #66b1ff;
      border-color: #dcdfe6;
      background-color: var(--tm-bg-hover);
    }
  }

  &.btn-text-red {
    color: #F56C6C;
    &:hover {
      color: #f78989;
      border-color: #dcdfe6;
      background-color: var(--tm-bg-hover);
    }
  }
}

.action-btn-green {
  background-color: var(--tm-accent-primary);
  border-color: var(--tm-accent-primary);
  &:hover {
    background-color: var(--tm-accent-hover);
    border-color: var(--tm-accent-hover);
  }
}

/* --- 分页 --- */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  margin-top: 16px;
  flex-shrink: 0;

  :deep(.el-pagination.is-background .el-pager li:not(.is-disabled).is-active) {
    background-color: var(--tm-accent-primary);
  }
}
</style>