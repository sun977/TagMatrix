<template>
  <div class="database-admin-container">
    <div class="page-header">
      <div class="header-top">
        <h2>系统数据库管理</h2>
        <span class="dev-mode-text">开发者模式</span>
      </div>
      <el-alert
        v-if="showWarning"
        title="注意：此界面提供对底层 SQLite 数据库及系统文件的直接控制。"
        type="warning"
        show-icon
        @close="showWarning = false"
      />
    </div>
    
    <div class="page-content">
      <el-tabs v-model="activeTab" class="admin-tabs">
        <el-tab-pane label="SQL查询终端" name="sql">
          <SqlConsole />
        </el-tab-pane>
        <el-tab-pane label="表结构与数据管理" name="table">
          <TableExplorer />
        </el-tab-pane>
        <el-tab-pane label="备份与还原中心" name="backup">
          <BackupRestore />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SqlConsole from './SqlConsole.vue'
import TableExplorer from './TableExplorer.vue'
import BackupRestore from './BackupRestore.vue'

const activeTab = ref('sql')
const showWarning = ref(true)
</script>

<style scoped lang="scss">
.database-admin-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  box-sizing: border-box;
  background-color: var(--tm-bg-main);
  overflow: hidden;

  .page-header {
    margin-bottom: 24px;

    .header-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;

      h2 {
        margin: 0;
        font-size: 20px;
        color: var(--tm-text-primary);
      }

      .dev-mode-text {
        font-size: 13px;
        color: var(--tm-text-secondary);
        background-color: var(--tm-bg-hover);
        padding: 4px 12px;
        border-radius: 12px;
      }
    }
  }

  .page-content {
    flex: 1;
    background-color: var(--tm-bg-card);
    border-radius: var(--tm-border-radius-md);
    border: 1px solid var(--tm-border-color);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.02);
  }

  .admin-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;

    :deep(.el-tabs__header) {
      margin-bottom: 0;
      padding: 0 20px;
      background-color: var(--tm-bg-card);
      border-bottom: 1px solid var(--tm-border-color);
    }
    
    :deep(.el-tabs__content) {
      flex: 1;
      padding: 0;
      overflow: hidden;
      
      .el-tab-pane {
        height: 100%;
      }
    }
  }
}
</style>