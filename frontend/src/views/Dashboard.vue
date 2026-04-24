<template>
  <div class="dashboard-page">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <h1 class="page-title">概览控制台</h1>
      <div class="header-right">
        <div class="task-status-pill">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在执行打标任务: 2024年Q1用户数据打标 (已完成67%)</span>
        </div>
        <el-button circle class="settings-btn">
          <el-icon><Setting /></el-icon>
        </el-button>
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
          <div class="card-value">128,567</div>
          <div class="card-trend green-text">较上周增长 12.5%</div>
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
          <div class="card-value">97,234</div>
          <div class="card-trend green-text">打标覆盖率 75.6%</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="card-top">
            <span class="card-title">标签总数</span>
            <div class="icon-wrapper yellow-bg">
              <el-icon><Collection /></el-icon>
            </div>
          </div>
          <div class="card-value">42</div>
          <div class="card-trend green-text">较上月新增 8 个标签</div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card">
          <div class="card-top">
            <span class="card-title">规则总数</span>
            <div class="icon-wrapper purple-bg">
              <el-icon><Document /></el-icon>
            </div>
          </div>
          <div class="card-value">68</div>
          <div class="card-trend red-text">3 条规则待优化</div>
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
            <template v-else-if="scope.row.statusType === 'success'">
              <el-button size="small" class="action-btn">查看日志</el-button>
              <el-button size="small" class="action-btn">导出</el-button>
            </template>
            <template v-else-if="scope.row.statusType === 'error'">
              <el-button type="danger" link size="small">查看错误日志</el-button>
              <el-button type="success" size="small" class="action-btn retry-btn">重试</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Loading, Setting, Coin, PriceTag, Collection, Document, UploadFilled, ArrowRight } from '@element-plus/icons-vue'

const recentTasks = ref([
  {
    name: '2024年Q1用户数据打标',
    statusType: 'running',
    statusText: '执行中',
    processed: '16,892 / 25,000',
    time: '00:12:34',
    createTime: '2024-04-24 13:23'
  },
  {
    name: '高价值用户标签更新',
    statusType: 'success',
    statusText: '已完成',
    processed: '42,568 / 42,568',
    time: '00:34:12',
    createTime: '2024-04-23 15:42'
  },
  {
    name: '用户行为标签批量更新',
    statusType: 'error',
    statusText: '失败',
    processed: '12,345 / 38,900',
    time: '00:18:45',
    createTime: '2024-04-22 09:17'
  },
  {
    name: '新用户注册标签初始化',
    statusType: 'success',
    statusText: '已完成',
    processed: '15,678 / 15,678',
    time: '00:08:23',
    createTime: '2024-04-21 11:05'
  },
  {
    name: '历史数据全量打标',
    statusType: 'success',
    statusText: '已完成',
    processed: '89,432 / 89,432',
    time: '01:23:45',
    createTime: '2024-04-20 16:30'
  }
])
</script>

<style scoped lang="scss">
.dashboard-page {
  padding: 24px 32px 40px;
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