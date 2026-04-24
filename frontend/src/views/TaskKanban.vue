<template>
  <div class="task-kanban-container">
    <div class="header-actions">
      <el-button type="primary" size="large" icon="VideoPlay" @click="dialogVisible = true">
        新建打标任务
      </el-button>
    </div>

    <!-- 历史任务批次卡片 -->
    <div class="task-list" v-loading="loadingBatches">
      <el-empty v-if="taskBatches.length === 0" description="暂无打标任务" style="grid-column: 1 / -1;" />

      <el-card class="task-card" shadow="hover" v-for="batch in taskBatches" :key="batch.id">
        <div class="card-header">
          <div class="batch-name">
            <el-icon><Tickets /></el-icon>
            {{ batch.name || `Batch_${batch.id}` }}
          </div>
          <el-tag :type="getStatusType(batch.status)" effect="light" round>
            {{ getStatusText(batch.status) }}
          </el-tag>
        </div>
        
        <div class="card-body">
          <div class="stat-item">
            <span class="label">处理数据量</span>
            <span class="value">{{ batch.total_processed }}</span>
          </div>
          <!-- TODO: 后续可在后端增加返回更详细的耗时和命中统计 -->
          <div class="stat-item">
            <span class="label">完成时间</span>
            <span class="value">{{ formatDate(batch.finished_at) }}</span>
          </div>
        </div>

        <div class="card-footer">
          <el-button type="primary" link>查看审计日志</el-button>
          <el-popconfirm 
            title="确定要撤销此次打标结果吗？此操作不可逆。" 
            v-if="batch.status === 'completed'"
            @confirm="handleRollback(batch.id)"
          >
            <template #reference>
              <el-button type="danger" link>一键回退</el-button>
            </template>
          </el-popconfirm>
        </div>
      </el-card>
    </div>

    <!-- 发起任务对话框 -->
    <el-dialog v-model="dialogVisible" title="发起自动打标任务" width="500px">
      <el-form label-position="top">
        <el-form-item label="任务批次名称 (可选)">
          <el-input v-model="taskForm.batchName" placeholder="例如：4月24日活跃用户打标" />
        </el-form-item>
        <el-form-item label="选择打标模式">
          <el-radio-group v-model="taskForm.tagMode">
            <el-radio :value="true" border>作为主标签覆盖</el-radio>
            <el-radio :value="false" border>作为次标签追加</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="输入要执行的规则 ID (逗号分隔)">
          <!-- 暂时用输入框代替，实际应用中可以接入下拉多选框选择真实存在的规则 -->
          <el-input v-model="taskForm.ruleIdsStr" placeholder="例如: 1,2,3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitTask" :loading="isSubmitting">
            开始执行
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { VideoPlay, Tickets } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { GetTaskBatches, RunTaggingTask, RollbackTask } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const loadingBatches = ref(false)
const taskBatches = ref<model.TagTaskBatch[]>([])

const dialogVisible = ref(false)
const isSubmitting = ref(false)

const taskForm = ref({
  batchName: '',
  tagMode: false,
  ruleIdsStr: ''
})

const fetchBatches = async () => {
  loadingBatches.value = true
  try {
    const res = await GetTaskBatches()
    taskBatches.value = res || []
  } catch (error) {
    ElMessage.error('获取任务批次失败: ' + String(error))
  } finally {
    loadingBatches.value = false
  }
}

const submitTask = async () => {
  if (!taskForm.value.ruleIdsStr) {
    ElMessage.warning('请输入规则 ID')
    return
  }

  // 解析逗号分隔的数字
  const ids = taskForm.value.ruleIdsStr.split(',').map(s => parseInt(s.trim())).filter(n => !isNaN(n))
  if (ids.length === 0) {
    ElMessage.warning('解析规则 ID 失败，请输入有效的数字')
    return
  }

  isSubmitting.value = true
  try {
    const batchId = await RunTaggingTask(ids, taskForm.value.batchName, taskForm.value.tagMode)
    ElMessage.success(`任务提交成功，批次号: ${batchId}`)
    dialogVisible.value = false
    taskForm.value.batchName = ''
    taskForm.value.ruleIdsStr = ''
    fetchBatches() // 刷新列表
  } catch (error) {
    ElMessage.error('提交任务失败: ' + String(error))
  } finally {
    isSubmitting.value = false
  }
}

const handleRollback = async (batchId: number) => {
  try {
    await RollbackTask(batchId)
    ElMessage.success('回退成功')
    fetchBatches()
  } catch (error) {
    ElMessage.error('回退失败: ' + String(error))
  }
}

const getStatusType = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'running': return 'primary'
    case 'rolled_back': return 'info'
    case 'failed': return 'danger'
    default: return 'info'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'completed': return '已完成'
    case 'running': return '执行中'
    case 'rolled_back': return '已回退'
    case 'failed': return '执行失败'
    default: return status
  }
}

const formatDate = (dateStr: string | null) => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString()
}

onMounted(() => {
  fetchBatches()
})
</script>

<style scoped lang="scss">
.task-kanban-container {
  .header-actions {
    margin-bottom: 24px;
  }

  .task-list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 20px;
  }

  .task-card {
    border-radius: var(--tm-border-radius);
    border: 1px solid var(--tm-border-color);
    box-shadow: var(--tm-shadow-sm);
    transition: box-shadow 0.2s;

    &:hover {
      box-shadow: var(--tm-shadow-md);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid var(--tm-border-color);

      .batch-name {
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--tm-text-primary);
      }
    }

    .card-body {
      display: flex;
      flex-direction: column;
      gap: 12px;
      margin-bottom: 20px;

      .stat-item {
        display: flex;
        justify-content: space-between;
        font-size: 14px;

        .label {
          color: var(--tm-text-secondary);
        }
        
        .value {
          font-weight: 500;
          color: var(--tm-text-primary);
        }
      }
    }

    .card-footer {
      display: flex;
      justify-content: space-between;
      padding-top: 12px;
      border-top: 1px dashed var(--tm-border-color);
    }
  }
}
</style>
