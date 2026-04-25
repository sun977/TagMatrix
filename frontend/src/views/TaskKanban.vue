<template>
  <div class="page-container">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <div class="header-left">
        <h1 class="page-title">打标任务看板</h1>
        <p class="page-subtitle">在这里发起和管理打标任务，查看执行进度和历史记录，支持任务回滚和日志查看。</p>
      </div>
    </header>

    <!-- 发起新任务区域 -->
    <div class="launch-section">
      <h3 class="section-title">发起新的打标任务</h3>
      <div class="launch-form">
        <el-row :gutter="24">
          <el-col :span="4">
            <div class="form-item">
              <label>任务名称</label>
              <el-input v-model="taskForm.batchName" placeholder="请输入任务名称" />
            </div>
          </el-col>
          <el-col :span="6">
            <div class="form-item">
              <label>选择数据源</label>
              <el-select v-model="taskForm.dataSource" placeholder="请选择数据源" class="w-100">
                <el-option :label="`全量库内数据 (${totalRecords}条)`" value="ds1" />
              </el-select>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="form-item">
              <label>选择要执行的标签规则</label>
              <el-select v-model="taskForm.rules" placeholder="请选择规则" class="w-100">
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
                    <div style="line-height: 1.6; max-width: 320px;">
                      <div style="margin-bottom: 4px;"><b>多标签模式</b>：数据命中几条规则，就打上几个平级的标签。</div>
                      <div style="margin-bottom: 4px;"><b>单标签模式</b>：命中多条规则时，仅取优先级最高的一个标签。</div>
                      <div style="margin-bottom: 4px;"><b>混合模式</b>：命中的所有标签均入库，但优先级最高的一个设为主标签。</div>
                      <div style="margin-top: 8px; color: #a0cfff;">
                        <i>* 注：后续将引入智能“主标签推导策略”（如基于业务线权重、ML打分）来更精准地推导和选取主/单标签。</i>
                      </div>
                    </div>
                  </template>
                  <el-icon style="cursor: pointer; color: var(--el-text-color-secondary);"><QuestionFilled /></el-icon>
                </el-tooltip>
              </label>
              <el-select v-model="taskForm.tagMode" class="w-100">
                <el-option label="多标签模式 (允许多个标签)" value="multiple" />
                <el-option label="单标签模式 (仅取最高优先级)" value="single" />
                <el-option label="混合模式 (最高优先级为主标签)" value="mixed" />
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
        <div class="form-actions">
          <el-button type="primary" class="action-btn-green" @click="submitTask" :loading="isSubmitting">
            <el-icon><VideoPlay /></el-icon> 开始执行任务
          </el-button>
        </div>
      </div>
    </div>

    <!-- 任务历史区域 -->
    <div class="history-section">
      <div class="section-header">
        <h3 class="section-title">任务历史</h3>
        <div class="header-filters">
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
        </div>
      </div>

      <el-table :data="paginatedTaskHistory" style="width: 100%" class="custom-table" v-loading="loadingBatches">
        <el-table-column prop="name" label="任务名称" min-width="180" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="scope">
            <div class="status-pill" :class="scope.row.statusType">
              <span class="dot"></span>
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
        <el-table-column prop="processed" label="处理数据量" width="160" />
        <el-table-column prop="time" label="耗时" width="100" />
        <el-table-column prop="creator" label="创建人" width="100" />
        <el-table-column prop="createTime" label="创建时间" width="160" />
        <el-table-column label="操作" width="220" align="right">
          <template #default="scope">
            <template v-if="scope.row.statusType === 'running'">
              <el-button size="small" class="action-btn">查看详情</el-button>
              <el-button type="danger" link size="small">终止</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'completed'">
              <el-button size="small" class="action-btn" @click="viewLogs(scope.row.id)">查看日志</el-button>
              <el-button size="small" class="action-btn" @click="exportLogs(scope.row.id)">导出</el-button>
              <el-button type="danger" link size="small" @click="handleRollback(scope.row.id)">回退</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'failed'">
              <el-button type="danger" link size="small">查看错误日志</el-button>
              <el-button type="success" size="small" class="action-btn retry-btn">重试</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'rolled_back'">
              <el-button size="small" class="action-btn" @click="viewLogs(scope.row.id)">查看日志</el-button>
              <el-button size="small" class="action-btn" @click="exportLogs(scope.row.id)">导出</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'pending'">
              <el-button size="small" class="action-btn">编辑</el-button>
              <el-button type="success" size="small" class="action-btn retry-btn">立即执行</el-button>
              <el-button type="danger" link size="small">删除</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <span class="pagination-info">共 {{ filteredTaskHistory.length }} 条记录</span>
        <el-pagination
          background
          layout="prev, pager, next"
          :total="filteredTaskHistory.length"
          :page-size="pageSize"
          :current-page="currentPage"
          @update:current-page="currentPage = $event"
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
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { VideoPlay, RefreshRight, QuestionFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { GetTaskBatches, RunTaggingTask, RollbackTask, GetAllRules, GetDashboardStats, GetTaskLogs, ExportTaskLogsCSV } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const loadingBatches = ref(false)

const totalRecords = ref(0)
const availableRules = ref<model.SysMatchRule[]>([])

const taskForm = ref({
  batchName: '',
  dataSource: 'ds1',
  rules: 'all',
  execStrategy: 'append',
  tagMode: 'multiple',
  desc: ''
})

const filterStatus = ref('all')
const filterTime = ref('all')
const isSubmitting = ref(false)

// 真实任务历史数据
const taskHistory = ref<any[]>([])

const currentPage = ref(1)
const pageSize = ref(10)

const filteredTaskHistory = computed(() => {
  let result = [...taskHistory.value]

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

const paginatedTaskHistory = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredTaskHistory.value.slice(start, end)
})

watch([filterStatus, filterTime], () => {
  currentPage.value = 1
})

// 日志弹窗相关状态
const logDialogVisible = ref(false)
const loadingLogs = ref(false)
const taskLogs = ref<model.TagTaskLogDto[]>([])

const viewLogs = async (batchId: number) => {
  logDialogVisible.value = true
  loadingLogs.value = true
  try {
    const logs = await GetTaskLogs(batchId)
    taskLogs.value = logs || []
  } catch (e: any) {
    ElMessage.error('获取日志失败: ' + String(e))
  } finally {
    loadingLogs.value = false
  }
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
    const stats = await GetDashboardStats()
    totalRecords.value = stats.totalRecords || 0

    const rules = await GetAllRules()
    availableRules.value = rules || []
  } catch (e: any) {
    console.error("加载前置数据失败", e)
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

      return {
        id: b.id,
        name: b.name,
        statusType: b.status,
        statusText: statusText,
        progress: isCompleted ? 100 : (isRunning ? 0 : 0), // 运行中的进度交给WebSocket推送
        processed: `${b.total_processed} 条`,
        time: '-',
        creator: '系统',
        createTime: new Date(b.created_at || Date.now()).toLocaleString(),
        rawTime: new Date(b.created_at || Date.now()).getTime()
      }
    })
  } catch (e: any) {
    ElMessage.error('获取任务历史失败: ' + String(e))
  } finally {
    loadingBatches.value = false
  }
}

const submitTask = async () => {
  if (!taskForm.value.batchName) {
    taskForm.value.batchName = '新建打标任务'
  }
  isSubmitting.value = true
  
  try {
    let ruleIDs: number[] = []
    if (taskForm.value.rules === 'all') {
      const rules = await GetAllRules()
      ruleIDs = rules.map(r => r.id)
    } else {
      ruleIDs = [parseInt(taskForm.value.rules)]
    }

    const isOverwrite = taskForm.value.execStrategy === 'overwrite'

    await RunTaggingTask(ruleIDs, taskForm.value.batchName, isOverwrite, taskForm.value.tagMode)
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

  // 监听后端推送的任务进度事件
  EventsOn('taskProgress', (data: any) => {
    const batchIndex = taskHistory.value.findIndex(b => b.id === data.batchID)
    if (batchIndex !== -1) {
      const batch = taskHistory.value[batchIndex]
      batch.statusType = data.status
      
      let statusText = '未知'
      if (data.status === 'running') statusText = '执行中'
      else if (data.status === 'completed') statusText = '已完成'
      else if (data.status === 'rolled_back') statusText = '已回退'
      else if (data.status === 'failed') statusText = '失败'

      batch.statusText = statusText
      batch.progress = data.progress
      batch.processed = `${data.processed} 条` // data.total 如果需要可以拼接

      taskHistory.value[batchIndex] = { ...batch }
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
})
</script>

<style scoped lang="scss">
.page-container {
  padding: 24px 32px 40px;
}

/* --- 页面顶部 --- */
.page-header {
  margin-bottom: 24px;

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
  background: #ffffff;
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  padding: 24px;
  margin-bottom: 32px;

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
  background: #ffffff;
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  padding: 24px;

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;

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
  --el-table-header-bg-color: #f9fafc;
  --el-table-header-text-color: var(--tm-text-secondary);
  
  :deep(th.el-table__cell) {
    font-weight: 500;
    padding: 12px 0;
  }
  
  :deep(td.el-table__cell) {
    padding: 16px 0;
    font-size: 14px;
    color: var(--tm-text-regular);
    border-bottom: 1px solid #fafafa;
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
    background-color: #e6f0fa;
    color: #3a8ee6;
    .dot { background-color: #3a8ee6; }
  }

  &.failed {
    background-color: #fef0f0;
    color: #f56c6c;
    .dot { background-color: #f56c6c; }
  }

  &.rolled_back {
    background-color: #f4f4f5;
    color: #909399;
    .dot { background-color: #909399; }
  }

  &.pending {
    background-color: #fdf5e6;
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
    background-color: #f5f5f5;
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
    background-color: #f5f7fa;
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
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;

  .pagination-info {
    font-size: 13px;
    color: var(--tm-text-secondary);
  }

  :deep(.el-pagination.is-background .el-pager li:not(.is-disabled).is-active) {
    background-color: var(--tm-accent-primary);
  }
}
</style>