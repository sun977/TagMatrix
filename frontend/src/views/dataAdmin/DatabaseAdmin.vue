<template>
  <div class="page-container">
    <header class="page-header">
      <div class="header-top-row">
        <div class="header-left">
          <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 8px;">
            <h1 class="page-title" style="margin: 0;">系统数据库管理</h1>
            <span class="dev-mode-text">开发者模式</span>
          </div>
          <p class="page-subtitle">管理底层 SQLite 数据库及系统文件，支持 SQL 查询、表结构浏览和备份还原。</p>
        </div>
        <el-alert
          v-if="showWarning"
          title="提示：此界面具有直接修改底层数据的最高权限，请谨慎操作以免造成数据损坏或丢失。"
          type="warning"
          show-icon
          @close="handleWarningClose"
          class="inline-warning"
        />
      </div>
    </header>
    
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
import { ref, onMounted } from 'vue'
import SqlConsole from './SqlConsole.vue'
import TableExplorer from './TableExplorer.vue'
import BackupRestore from './BackupRestore.vue'

const activeTab = ref('sql')
const showWarning = ref(false)

onMounted(() => {
  if (!localStorage.getItem('sys_db_warning_dismissed')) {
    showWarning.value = true
  }
})

const handleWarningClose = () => {
  showWarning.value = false
  localStorage.setItem('sys_db_warning_dismissed', 'true')
}
</script>

<style scoped lang="scss">
.page-container {
  padding: 24px 32px 40px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overflow: hidden;

.page-header {
  margin-bottom: 24px;
  flex-shrink: 0;

  .header-top-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 20px;
    width: 100%;
  }

  .header-left {
    display: flex;
    flex-direction: column;
  }

  .inline-warning {
    margin: 0;
    width: auto;
    flex-shrink: 1;
    max-width: 60%;
  }

  .page-title {
      font-size: 20px;
      font-weight: 600;
      color: var(--tm-text-primary);
    }

    .page-subtitle {
      font-size: 14px;
      color: var(--tm-text-secondary);
      margin: 0;
    }

    .dev-mode-text {
      font-size: 13px;
      color: #e6a23c;
      background-color: #fdf5e6;
      padding: 2px 8px;
      border-radius: 4px;
      border: 1px solid #faecd8;
    }
  }

  .page-content {
    flex: 1;
    background-color: #ffffff;
    border-radius: var(--tm-border-radius);
    border: 1px solid var(--tm-border-color);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .admin-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;

    :deep(.el-tabs__header) {
      margin-bottom: 0;
      padding: 0 20px;
      background-color: #ffffff;
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