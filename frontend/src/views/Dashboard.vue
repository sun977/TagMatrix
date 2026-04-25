<template>
  <div class="dashboard-page">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <h1 class="page-title">概览控制台</h1>
      <div class="header-right">
        <div class="task-status-pill" v-if="runningTask">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在执行打标任务: {{ runningTask.name }}</span>
        </div>
      </div>
    </header>

    <!-- 欢迎语 -->
    <div class="welcome-section">
      <h2>欢迎回来，数据管理员</h2>
      <p>这里是 TagMatrix 智能标签管理系统，你可以在这里管理数据、配置标签规则和执行打标任务。</p>
    </div>

    <!-- 数据统计卡片 -->
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <div class="stat-card">
          <div class="card-top">
            <span class="card-title">总数据量</span>
            <div class="icon-wrapper green-bg">
              <el-icon><Coin /></el-icon>
            </div>
          </div>
          <div class="card-value">{{ stats.totalRecords || 0 }}</div>
          <div class="card-trend green-text">当前库内记录总数</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="card-top">
            <span class="card-title">已打标数据量</span>
            <div class="icon-wrapper blue-bg">
              <el-icon><PriceTag /></el-icon>
            </div>
          </div>
          <div class="card-value">{{ stats.taggedRecords || 0 }}</div>
          <div class="card-trend green-text">打标覆盖率 {{ stats.totalRecords ? Math.round((stats.taggedRecords / stats.totalRecords) * 100) : 0 }}%</div>
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

    <!-- 最近打标任务 -->
    <div class="section-container">
      <div class="section-header">
        <h3 class="section-title">最近打标任务</h3>
        <el-button type="primary" link class="view-all-btn" @click="$router.push('/task-kanban')">
          查看全部 <el-icon class="el-icon--right"><ArrowRight /></el-icon>
        </el-button>
      </div>
      
      <el-table :data="recentTasks" style="width: 100%" class="custom-table">
        <el-table-column prop="name" label="任务名称" min-width="200" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="scope">
            <div class="status-pill" :class="scope.row.statusType">
              <span class="dot"></span>
              {{ scope.row.statusText }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="processed" label="处理数量" width="180" />
        <el-table-column prop="time" label="耗时" width="120" />
        <el-table-column prop="createTime" label="创建时间" width="180" />
        <el-table-column label="操作" width="220" align="right">
          <template #default="scope">
            <el-button v-if="scope.row.statusType === 'running'" size="small" class="action-btn">查看详情</el-button>
            <template v-else-if="scope.row.statusType === 'completed' || scope.row.statusType === 'rolled_back'">
              <el-button size="small" class="action-btn" @click="viewLogs(scope.row.id)">查看</el-button>
              <el-button size="small" class="action-btn" @click="exportLogs(scope.row.id)">导出</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'failed'">
              <el-button type="danger" link size="small">查看错误日志</el-button>
              <el-button type="success" size="small" class="action-btn retry-btn">重试</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
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
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Loading, Setting, Coin, PriceTag, Collection, Document, UploadFilled, ArrowRight } from '@element-plus/icons-vue'
import { GetDashboardStats, GetTaskBatches, GetAllTags, GetAllRules, GetTaskLogs, ExportTaskLogsCSV } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'
import { ElMessage } from 'element-plus'

const stats = ref<model.DashboardStats>({
  totalRecords: 0,
  taggedRecords: 0,
  totalTags: 0,
  totalRules: 0
} as any)

const recentTasks = ref<any[]>([])
const loadingTasks = ref(false)

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
    // 同时获取规则和标签数据（用于匹配标签名称）
    const [rules, tags] = await Promise.all([
      GetAllRules(),
      GetAllTags()
    ])
    
    // 过滤出已生效的规则
    const enabledRules = rules.filter(r => r.is_enabled)
    
    // 组装关联的标签信息
    const tagsMap = new Map(tags.map(t => [t.id, t]))
    
    rulesList.value = enabledRules.map(r => {
      const tag = tagsMap.get(r.tag_id)
      return {
        ...r,
        tagName: tag ? tag.name : '未知标签',
        tagColor: tag ? tag.color : ''
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
    // 只取前5条
    const recent = batches.slice(0, 5)
    recentTasks.value = recent.map((b: model.TagTaskBatch) => {
      return {
        id: b.id,
        name: b.name,
        statusType: b.status,
        statusText: b.status === 'running' ? '执行中' : (b.status === 'completed' ? '已完成' : (b.status === 'failed' ? '失败' : '未知')),
        processed: `${b.total_processed}`,
        time: '-',
        createTime: new Date(b.created_at || Date.now()).toLocaleString()
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
})
</script>

<style scoped lang="scss">
.dashboard-page {
  padding: 24px 32px 40px;
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

/* --- 页面顶部 --- */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;

  .page-title {
    font-size: 20px;
    font-weight: 600;
    color: var(--tm-text-primary);
    margin: 0;
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

/* --- 欢迎语 --- */
.welcome-section {
  margin-bottom: 32px;

  h2 {
    font-size: 24px;
    font-weight: 700;
    margin: 0 0 8px 0;
    color: var(--tm-text-primary);
  }

  p {
    margin: 0;
    font-size: 14px;
    color: var(--tm-text-secondary);
  }
}

/* --- 统计卡片 --- */
.stat-cards {
  margin-bottom: 40px;

  .stat-card {
    background: #ffffff;
    border: 1px solid var(--tm-border-color);
    border-radius: var(--tm-border-radius);
    padding: 20px 24px;
    transition: box-shadow 0.2s ease;

    &:hover {
      box-shadow: var(--tm-shadow-sm);
    }

    .card-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

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
          background-color: #e8f7f0;
          color: var(--tm-accent-primary);
        }
        &.blue-bg {
          background-color: #e6f0fa;
          color: #3a8ee6;
        }
        &.yellow-bg {
          background-color: #fdf5e6;
          color: #e6a23c;
        }
        &.purple-bg {
          background-color: #f3e8ff;
          color: #9d5cb8;
        }
      }
    }

    .card-value {
      font-size: 32px;
      font-weight: 700;
      color: var(--tm-text-primary);
      margin-bottom: 8px;
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

/* --- 通用区块 --- */
.section-container {
  margin-bottom: 40px;

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    margin: 0 0 20px 0;
    color: var(--tm-text-primary);
  }

  .view-all-btn {
    font-weight: 500;
  }
}

/* --- 快速操作 --- */
.quick-action-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 24px;
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

/* --- 表格样式 --- */
.custom-table {
  --el-table-border-color: var(--tm-border-color);
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

  &.success {
    background-color: #e6f0fa;
    color: #3a8ee6;
    .dot { background-color: #3a8ee6; }
  }

  &.error {
    background-color: #fef0f0;
    color: #f56c6c;
    .dot { background-color: #f56c6c; }
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
</style>