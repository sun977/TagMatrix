<template>
  <div class="dashboard-container">
    <el-row :gutter="24">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-header">总数据量</div>
          <div class="stat-value">0</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-header">已打标数据</div>
          <div class="stat-value">0</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-header">标签数量</div>
          <div class="stat-value">0</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-header">匹配规则数</div>
          <div class="stat-value">0</div>
        </el-card>
      </el-col>
    </el-row>

    <div class="quick-actions">
      <h3>快速入口</h3>
      <div class="actions-grid">
        <el-button size="large" @click="$router.push('/data-source')">导入数据源</el-button>
        <el-button size="large" @click="$router.push('/tag-rule')">配置标签规则</el-button>
        <el-button size="large" type="primary" @click="$router.push('/task-kanban')">发起打标任务</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 后续将接入 Wails 接口获取真实统计数据
</script>

<style scoped lang="scss">
.dashboard-container {
  .stat-card {
    border-radius: var(--tm-border-radius);
    border: none;
    background-color: var(--tm-bg-sidebar);
    
    .stat-header {
      color: var(--tm-text-secondary);
      font-size: 14px;
      margin-bottom: 12px;
    }
    
    .stat-value {
      font-size: 32px;
      font-weight: 600;
      color: var(--tm-accent-primary);
    }
  }

  .quick-actions {
    margin-top: 40px;
    
    h3 {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 20px;
      color: var(--tm-text-primary);
    }
    
    .actions-grid {
      display: flex;
      gap: 16px;
    }
  }
}
</style>
