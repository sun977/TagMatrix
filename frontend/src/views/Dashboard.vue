<template>
  <div class="dashboard-page">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <div class="header-left">
        <h1 class="page-title">仪表板</h1>
        <p class="page-subtitle">这里是 TagMatrix 智能标签管理系统，你可以在这里管理数据、配置标签规则和执行打标任务。</p>
      </div>
      <div class="header-right">
        <div class="task-status-pill" v-if="runningTask">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在执行打标任务: {{ runningTask.name }}</span>
        </div>
      </div>
    </header>

    <!-- 数据统计卡片 -->
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <div class="stat-card clickable-card" @click="showDatasetRecordsDialog">
          <div class="card-top">
            <span class="card-title">数据总量</span>
            <div class="icon-wrapper green-bg">
              <el-icon><Coin /></el-icon>
            </div>
          </div>
          <div class="card-value">{{ stats.totalRecords || 0 }}</div>
          <div class="card-trend green-text" style="cursor: pointer;">当前库内记录总数</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card clickable-card" @click="showDatasetTaggedDialog">
          <div class="card-top">
            <span class="card-title">已打标数据量</span>
            <div class="icon-wrapper blue-bg">
              <el-icon><PriceTag /></el-icon>
            </div>
          </div>
          <div class="card-value">{{ stats.taggedRecords || 0 }}</div>
          <div class="card-trend green-text" style="cursor: pointer;">打标覆盖率 {{ stats.totalRecords ? Math.round((stats.taggedRecords / stats.totalRecords) * 100) : 0 }}%</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card clickable-card" @click="showTagsDialog">
          <div class="card-top">
            <span class="card-title">标签总数</span>
            <div class="icon-wrapper yellow-bg">
              <el-icon><Collection /></el-icon>
            </div>
          </div>
          <div class="card-value">{{ stats.totalTags || 0 }}</div>
          <div class="card-trend green-text" style="cursor: pointer;">系统标签详情</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card clickable-card" @click="showRulesDialog">
          <div class="card-top">
            <span class="card-title">规则总数</span>
            <div class="icon-wrapper purple-bg">
              <el-icon><Document /></el-icon>
            </div>
          </div>
          <div class="card-value">{{ stats.totalRules || 0 }}</div>
          <div class="card-trend green-text" style="cursor: pointer;">系统规则详情</div>
        </div>
      </el-col>
    </el-row>

    <!-- 快速操作 -->
    <div class="section-container">
      <h3 class="section-title">快速操作</h3>
      <el-row :gutter="20">
        <el-col :span="12">
          <div class="quick-action-card" @click="$router.push('/data-source')">
            <div class="action-icon green-light-bg">
              <el-icon><UploadFilled /></el-icon>
            </div>
            <div class="action-content">
              <h4>导入新数据</h4>
              <p>支持 Excel、CSV 格式文件，快速导入待打标数据</p>
            </div>
          </div>
        </el-col>
        <el-col :span="12">
          <div class="quick-action-card" @click="$router.push('/tag-rule')">
            <div class="action-icon green-light-bg">
              <el-icon><PriceTag /></el-icon>
            </div>
            <div class="action-content">
              <h4>新建标签</h4>
              <p>创建新的标签分类，配置相关匹配规则</p>
            </div>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- 最近任务 -->
    <div class="section-container tasks-section">
      <div class="section-header">
        <h3 class="section-title">最近任务</h3>
        <el-button type="primary" link class="view-all-btn" @click="$router.push('/task-kanban')">
          查看全部 <el-icon class="el-icon--right"><ArrowRight /></el-icon>
        </el-button>
      </div>
      
      <el-table :data="paginatedRecentTasks" style="width: 100%" height="100%" class="custom-table" v-loading="loadingTasks">
        <el-table-column prop="name" label="任务名称" min-width="180" />
        <el-table-column prop="desc" label="任务描述" min-width="180" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="scope">
            <div class="status-pill" :class="scope.row.statusClass">
              <span class="dot"></span>
              {{ scope.row.statusText }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="processed" label="处理数量(命中)" width="180" />
        <el-table-column prop="time" label="用时" width="120" />
        <el-table-column prop="createTime" label="创建时间" width="180" />
        <el-table-column label="操作" width="160" align="center">
          <template #default="scope">
            <div class="action-buttons-group">
              <el-button v-if="scope.row.statusType === 'running'" size="small" class="action-btn">查看详情</el-button>
              <template v-else-if="scope.row.statusType === 'completed' || scope.row.statusType === 'rolled_back'">
                <el-button size="small" class="action-btn" @click="viewLogs(scope.row.id)">查看</el-button>
                <el-button size="small" class="action-btn" @click="exportLogs(scope.row.id)">导出</el-button>
              </template>
              <template v-else-if="scope.row.statusType === 'failed'">
                <el-button type="danger" link size="small">查看错误日志</el-button>
                <el-button type="success" size="small" class="action-btn retry-btn">重试</el-button>
              </template>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="recentTasks.length > 0">
        <el-pagination
          :current-page="currentPage"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="recentTasks.length"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
    <!-- 标签列表弹窗 -->
    <el-dialog v-model="tagsDialogVisible" title="系统标签列表" width="800px">
      <el-table :data="tagsList" style="width: 100%" height="400" v-loading="loadingTags" class="custom-table">
        <el-table-column prop="name" label="标签名称" width="180">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 8px;">
              <span class="tag-color-dot" :style="{ backgroundColor: row.color || 'var(--tm-accent-primary)' }"></span>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" />
        <el-table-column prop="description" label="描述" show-overflow-tooltip />
      </el-table>
    </el-dialog>

    <!-- 规则列表弹窗 -->
    <el-dialog v-model="rulesDialogVisible" title="规则列表" width="900px">
      <el-table :data="rulesList" style="width: 100%" height="400" v-loading="loadingRules" class="custom-table">
        <el-table-column prop="name" label="规则名称" min-width="180" />
        <el-table-column prop="datasetName" label="所属数据集" width="150" />
        <el-table-column prop="tagName" label="关联标签" width="150">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 8px;">
              <span class="tag-color-dot" :style="{ backgroundColor: row.tagColor || 'var(--tm-accent-primary)' }"></span>
              <span>{{ row.tagName }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="100" align="center" />
        <el-table-column prop="is_enabled" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.is_enabled ? 'success' : 'danger'">
              {{ row.is_enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

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

    <!-- 数据总量（按数据集）弹窗 -->
    <el-dialog v-model="datasetRecordsDialogVisible" title="数据量统计" width="700px">
      <el-table :data="stats.datasetStats || []" style="width: 100%" height="400" class="custom-table">
        <el-table-column prop="datasetName" label="数据集名称" min-width="200" />
        <el-table-column prop="totalRecords" label="数据量" width="150" align="right" />
        <el-table-column label="占比" width="150" align="right">
          <template #default="{ row }">
            {{ stats.totalRecords ? Math.round((row.totalRecords / stats.totalRecords) * 100) : 0 }}%
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 已打标数据量（按数据集）弹窗 -->
    <el-dialog v-model="datasetTaggedDialogVisible" title="打标覆盖率统计" width="700px">
      <el-table :data="stats.datasetStats || []" style="width: 100%" height="400" class="custom-table">
        <el-table-column prop="datasetName" label="数据集名称" min-width="200" />
        <el-table-column prop="taggedRecords" label="已打标数量" width="150" align="right" />
        <el-table-column label="打标覆盖率" width="150" align="right">
          <template #default="{ row }">
            {{ row.totalRecords ? Math.round((row.taggedRecords / row.totalRecords) * 100) : 0 }}%
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { Loading, Setting, Coin, PriceTag, Collection, Document, UploadFilled, ArrowRight } from '@element-plus/icons-vue'
import { GetDashboardStats, GetTaskBatches, GetAllTags, GetAllRules, GetTaskLogs, GetTaskLogsPaged, ExportTaskLogsCSV, ListDatasets } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { ElMessage } from 'element-plus'

const stats = ref<model.DashboardStats>({
  totalRecords: 0,
  taggedRecords: 0,
  totalTags: 0,
  totalRules: 0,
  datasetStats: []
} as any)

const datasetRecordsDialogVisible = ref(false)
const datasetTaggedDialogVisible = ref(false)

const showDatasetRecordsDialog = () => {
  datasetRecordsDialogVisible.value = true
}

const showDatasetTaggedDialog = () => {
  datasetTaggedDialogVisible.value = true
}

const recentTasks = ref<any[]>([])
const loadingTasks = ref(false)

const currentPage = ref(1)
const pageSize = ref(10)

const paginatedRecentTasks = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return recentTasks.value.slice(start, end)
})

const handleSizeChange = (val: number) => {
  pageSize.value = val
  currentPage.value = 1
}

const handleCurrentChange = (val: number) => {
  currentPage.value = val
}

const runningTask = computed(() => {
  return recentTasks.value.find(t => t.statusType === 'running')
})

const tagsDialogVisible = ref(false)
const tagsList = ref<model.SysTag[]>([])
const loadingTags = ref(false)

const showTagsDialog = async () => {
  tagsDialogVisible.value = true
  loadingTags.value = true
  try {
    tagsList.value = await GetAllTags()
  } catch (e) {
    console.error('Failed to load tags:', e)
  } finally {
    loadingTags.value = false
  }
}

const rulesDialogVisible = ref(false)
const rulesList = ref<any[]>([])
const loadingRules = ref(false)

const showRulesDialog = async () => {
  rulesDialogVisible.value = true
  loadingRules.value = true
  try {
    // 同时获取规则和标签数据（用于匹配标签名称）以及数据集（用于匹配数据集名称）
    const [rules, tags, datasets] = await Promise.all([
      GetAllRules(),
      GetAllTags(),
      ListDatasets()
    ])
    
    // 过滤出已生效的规则
    const enabledRules = rules.filter(r => r.is_enabled)
    
    // 组装关联的标签信息和数据集信息
    const tagsMap = new Map(tags.map(t => [t.id, t]))
    const datasetsMap = new Map(datasets.map(d => [d.id, d]))
    
    rulesList.value = enabledRules.map(r => {
      const tag = tagsMap.get(r.tag_id)
      const ds = datasetsMap.get(r.dataset_id)
      return {
        ...r,
        tagName: tag ? tag.name : '未知标签',
        tagColor: tag ? tag.color : '',
        datasetName: ds ? ds.name : '未知数据集'
      }
    })
  } catch (e) {
    console.error('Failed to load rules:', e)
  } finally {
    loadingRules.value = false
  }
}

// 日志弹窗相关状态
const logDialogVisible = ref(false)
const loadingLogs = ref(false)
const taskLogs = ref<model.TagTaskLogDto[]>([])
const logTotal = ref(0)
const logCurrentPage = ref(1)
const logPageSize = ref(20)
const currentLogBatchId = ref<number>(0)

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

const loadDashboardData = async () => {
  try {
    const s = await GetDashboardStats()
    if (s) stats.value = s
  } catch (e) {
    console.error('Failed to load stats:', e)
  }

  try {
    loadingTasks.value = true
    const batches = await GetTaskBatches()
    recentTasks.value = batches.map((b: model.TagTaskBatch) => {
      const isRunning = b.status === 'running'
      let timeStr = '-'
      if (b.finished_at && b.created_at) {
        const diff = new Date(b.finished_at).getTime() - new Date(b.created_at).getTime()
        if (diff >= 0) {
          const totalSeconds = diff / 1000
          if (totalSeconds < 60) timeStr = `${Number(totalSeconds.toFixed(1))}秒`
          else if (totalSeconds < 3600) timeStr = `${Math.floor(totalSeconds/60)}分${Math.floor(totalSeconds%60)}秒`
          else timeStr = `${Math.floor(totalSeconds/3600)}小时${Math.floor((totalSeconds%3600)/60)}分`
        }
      } else if (isRunning && b.created_at) {
        const diff = Date.now() - new Date(b.created_at).getTime()
        if (diff >= 0) {
          const totalSeconds = diff / 1000
          if (totalSeconds < 60) timeStr = `${Number(totalSeconds.toFixed(1))}秒`
          else if (totalSeconds < 3600) timeStr = `${Math.floor(totalSeconds/60)}分${Math.floor(totalSeconds%60)}秒`
          else timeStr = `${Math.floor(totalSeconds/3600)}小时${Math.floor((totalSeconds%3600)/60)}分`
        }
      }

      return {
        id: b.id,
        name: b.name,
        desc: b.desc || '-',
        statusType: b.status,
        statusText: isRunning ? '执行中' : (b.status === 'completed' ? '已完成' : (b.status === 'failed' ? '失败' : (b.status === 'rolled_back' ? '已回退' : '未知'))),
        statusClass: b.status === 'running' ? 'status-running' : 
                     b.status === 'completed' ? 'status-completed' :
                     b.status === 'failed' ? 'status-failed' :
                     b.status === 'rolled_back' ? 'status-rolled-back' : '',
        processed: `${b.total_processed}`,
        time: timeStr,
        createTime: new Date(b.created_at || Date.now()).toLocaleString(),
        rawTime: new Date(b.created_at || Date.now()).getTime()
      }
    })
  } catch (e) {
    console.error('Failed to load recent tasks:', e)
  } finally {
    loadingTasks.value = false
  }
}

onMounted(() => {
  loadDashboardData()

  EventsOn('task_list_updated', loadDashboardData)

  // 监听后端推送的任务进度事件，保持与任务看板同步的实时更新
  EventsOn('taskProgress', (data: any) => {
    const batchIndex = recentTasks.value.findIndex(b => b.id === data.batchID)
    if (batchIndex !== -1) {
      const batch = recentTasks.value[batchIndex]
      const oldStatus = batch.statusType
      batch.statusType = data.status
      
      let statusText = '未知'
      if (data.status === 'running') statusText = '执行中'
      else if (data.status === 'completed') statusText = '已完成'
      else if (data.status === 'rolled_back') statusText = '已回退'
      else if (data.status === 'failed') statusText = '失败'

      let statusClass = ''
      if (data.status === 'running') statusClass = 'status-running'
      else if (data.status === 'completed') statusClass = 'status-completed'
      else if (data.status === 'rolled_back') statusClass = 'status-rolled-back'
      else if (data.status === 'failed') statusClass = 'status-failed'

      batch.statusText = statusText
      batch.statusClass = statusClass
      batch.processed = `${data.processed}`

      if (data.status === 'running') {
        const diff = Date.now() - batch.rawTime
        if (diff >= 0) {
          const totalSeconds = diff / 1000
          if (totalSeconds < 60) batch.time = `${Number(totalSeconds.toFixed(1))}秒`
          else if (totalSeconds < 3600) batch.time = `${Math.floor(totalSeconds/60)}分${Math.floor(totalSeconds%60)}秒`
          else batch.time = `${Math.floor(totalSeconds/3600)}小时${Math.floor((totalSeconds%3600)/60)}分`
        }
      }

      recentTasks.value[batchIndex] = { ...batch }

      if (oldStatus === 'running' && (data.status === 'completed' || data.status === 'failed')) {
        // 完成后刷新面板数据，获取最终时间并更新统计卡片
        loadDashboardData()
      }
    } else {
      // 刚发起的任务
      if (data.status === 'running' && data.progress === 0) {
        loadDashboardData()
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
.dashboard-page {
  /* 调整 padding 可以改变整个页面的外框留白边距 (上方高度、左右宽度等) */
  padding: 16px 24px;
  flex: 1;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  min-height: 0;
}

.clickable-card {
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  }
}

.tag-color-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

  /* --- 页面顶部 (Header) 对应UI最上方标题和副标题 --- */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  /* 调整这里改变 Header 距离下方【数据统计卡片】的垂直间距 */
  margin-bottom: 16px;

  .header-left {
    .page-title {
      font-size: 20px;
      font-weight: 600;
      color: var(--tm-text-primary);
      margin: 0 0 8px 0;
    }
    
    .page-subtitle {
      margin: 0;
      font-size: 14px;
      color: var(--tm-text-secondary);
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;

    .task-status-pill {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 16px;
      background-color: var(--tm-accent-light);
      color: var(--tm-accent-primary);
      border-radius: 20px;
      font-size: 13px;
      font-weight: 500;
    }

    .settings-btn {
      border: 1px solid var(--tm-border-color);
      color: var(--tm-text-secondary);
      &:hover {
        color: var(--tm-text-primary);
        border-color: #dcdfe6;
      }
    }
  }
}

  /* --- 统计卡片 (对应UI上方四个指标展示框) --- */
.stat-cards {
  /* 调整这里改变 统计卡片 整体与下方【快速操作】的垂直间距 */
  margin-bottom: 16px;

  .stat-card {
    background: var(--tm-bg-main);
    border: 1px solid var(--tm-border-color);
    border-radius: var(--tm-border-radius);
    /* 调整这里改变单个统计卡片内部的上下、左右留白（影响卡片高度和胖瘦） */
    padding: 12px 16px;
    transition: box-shadow 0.2s ease;

    &:hover {
      box-shadow: var(--tm-shadow-sm);
    }

    .card-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      /* 调整这里改变卡片内标题行距离下方大数字的垂直间距 */
      margin-bottom: 8px;

      .card-title {
        font-size: 14px;
        color: var(--tm-text-secondary);
      }

      .icon-wrapper {
        width: 32px;
        height: 32px;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        
        .el-icon {
          font-size: 16px;
        }

        &.green-bg {
          background-color: var(--tm-accent-light);
          color: var(--tm-accent-primary);
        }
        &.blue-bg {
          background-color: rgba(58, 142, 230, 0.1);
          color: #3a8ee6;
        }
        &.yellow-bg {
          background-color: rgba(230, 162, 60, 0.1);
          color: #e6a23c;
        }
        &.purple-bg {
          background-color: rgba(157, 92, 184, 0.1);
          color: #9d5cb8;
        }
      }
    }

    .card-value {
      /* 调整这里改变卡片内展示的大数字大小 */
      font-size: 28px;
      font-weight: 700;
      color: var(--tm-text-primary);
      /* 调整这里改变大数字距离最下方描述文字的垂直间距 */
      margin-bottom: 4px;
    }

    .card-trend {
      font-size: 13px;
      font-weight: 500;

      &.green-text {
        color: var(--tm-accent-primary);
      }
      &.red-text {
        color: #f56c6c;
      }
    }
  }
}

/* --- 通用区块外层 (主要作用于【快速操作】和【最近任务】的外层间距) --- */
.section-container {
  /* 调整这里改变区块距离下方其他区块的间距 */
  margin-bottom: 16px;
  flex-shrink: 0;

  &.tasks-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    margin-bottom: 0;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    /* 调整这里改变区块标题和内部内容（如表格）的垂直间距 */
    margin-bottom: 12px;
    flex-shrink: 0;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    /* 调整下方 margin-bottom 数值可控制纯标题（无右上角按钮时）与下方内容的间距 */
    margin: 0 0 12px 0;
    color: var(--tm-text-primary);
  }

  .view-all-btn {
    font-weight: 500;
  }
}

/* --- 快速操作 (对应UI中间那两个带图标的操作按钮框) --- */
.quick-action-card {
  display: flex;
  align-items: center;
  /* gap 调整左右图标和文字之间的间距 */
  gap: 16px;
  /* 调整 padding 改变【快速操作】卡片内部的上下左右留白高度 */
  padding: 12px 16px;
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--tm-accent-primary);
    box-shadow: var(--tm-shadow-sm);
  }

  .action-icon {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    color: var(--tm-accent-primary);

    &.green-light-bg {
      background-color: var(--tm-accent-light);
    }
  }

  .action-content {
    h4 {
      margin: 0 0 8px 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--tm-text-primary);
    }
    p {
      margin: 0;
      font-size: 13px;
      color: var(--tm-text-secondary);
    }
  }
}

/* --- 表格样式 (对应最近任务等处的表格) --- */
.custom-table {
  --el-table-border-color: transparent;
  --el-table-header-bg-color: var(--tm-bg-subtle);
  --el-table-header-text-color: var(--tm-text-secondary);
  flex: 1;
  min-height: 0;
  
  :deep(th.el-table__cell) {
    font-weight: 500;
    /* 调整这里改变表头上下间距 */
    padding: 8px 0;
  }
  
  :deep(td.el-table__cell) {
    /* 调整这里改变表格数据行的上下间距（让表格更紧凑） */
    padding: 10px 0;
    font-size: 14px;
    color: var(--tm-text-regular);
    border-bottom: 1px solid var(--tm-border-color);
  }
}

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

  &.status-running {
    background-color: var(--tm-accent-light);
    color: var(--tm-accent-primary);
    .dot { background-color: var(--tm-accent-primary); }
  }

  &.status-completed {
    background-color: rgba(58, 142, 230, 0.1);
    color: #3a8ee6;
    .dot { background-color: #3a8ee6; }
  }

  &.status-failed {
    background-color: rgba(245, 108, 108, 0.1);
    color: #f56c6c;
    .dot { background-color: #f56c6c; }
  }

  &.status-rolled-back {
    background-color: #f4f4f5;
    color: #909399;
    .dot { background-color: #909399; }
  }
}

.action-buttons-group {
  display: flex;
  justify-content: center;
  gap: 6px; /* 调整这里可以控制按钮之间的间距 */
  
  /* 去除 Element Plus 默认的兄弟按钮左边距，完全由 gap 控制 */
  :deep(.el-button + .el-button) {
    margin-left: 0;
  }
}

.action-btn {
  border-color: var(--tm-border-color);
  color: var(--tm-text-regular);
  border-radius: 6px;

  &:hover {
    color: var(--tm-text-primary);
    border-color: var(--tm-border-color);
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
}
</style>