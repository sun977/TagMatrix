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
              <label>打标模式</label>
              <el-select v-model="taskForm.tagMode" class="w-100">
                <el-option label="追加模式 (保留原有标签)" value="append" />
                <el-option label="覆盖模式 (清除原有标签)" value="overwrite" />
              </el-select>
            </div>
          </el-col>
          <el-col :span="4">
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
            <el-option label="近7天" value="7d" />
            <el-option label="近30天" value="30d" />
          </el-select>
          <el-button circle @click="fetchBatches" :loading="loadingBatches">
            <el-icon><RefreshRight /></el-icon>
          </el-button>
        </div>
      </div>

      <el-table :data="taskHistory" style="width: 100%" class="custom-table" v-loading="loadingBatches">
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
              <el-button size="small" class="action-btn">查看日志</el-button>
              <el-button size="small" class="action-btn">导出</el-button>
              <el-button type="danger" link size="small" @click="handleRollback(scope.row.id)">回退</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'failed'">
              <el-button type="danger" link size="small">查看错误日志</el-button>
              <el-button type="success" size="small" class="action-btn retry-btn">重试</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'rolled_back'">
              <el-button size="small" class="action-btn">查看日志</el-button>
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
        <span class="pagination-info">共 {{ taskHistory.length }} 条记录</span>
        <el-pagination
          background
          layout="prev, pager, next"
          :total="taskHistory.length"
          :page-size="10"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { VideoPlay, RefreshRight } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { GetTaskBatches, RunTaggingTask, RollbackTask, GetAllRules, GetDashboardStats } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const loadingBatches = ref(false)

const totalRecords = ref(0)
const availableRules = ref<model.SysMatchRule[]>([])

const taskForm = ref({
  batchName: '',
  dataSource: 'ds1',
  rules: 'all',
  tagMode: 'append',
  desc: ''
})

const filterStatus = ref('all')
const filterTime = ref('7d')
const isSubmitting = ref(false)

// 真实任务历史数据
const taskHistory = ref<any[]>([])

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
      return {
        id: b.id,
        name: b.name,
        statusType: b.status,
        statusText: b.status === 'running' ? '执行中' : (b.status === 'completed' ? '已完成' : (b.status === 'failed' ? '失败' : (b.status === 'rolled_back' ? '已回退' : '未知'))),
        progress: (b.status === 'completed' || b.status === 'rolled_back') ? 100 : (b.status === 'running' ? 50 : 0),
        processed: `${b.total_processed} 条`,
        time: '-',
        creator: '系统',
        createTime: new Date(b.created_at || Date.now()).toLocaleString()
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

    const isPrimary = taskForm.value.tagMode === 'overwrite'

    await RunTaggingTask(ruleIDs, taskForm.value.batchName, isPrimary)
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