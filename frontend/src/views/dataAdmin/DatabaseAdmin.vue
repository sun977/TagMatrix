<template>
  <div class="database-admin-container">
    <div class="page-header">
      <div class="header-left">
        <h2>系统数据库管理 <el-tag type="danger" effect="dark" size="small">开发者模式</el-tag></h2>
        <p class="subtitle">高危操作：此界面提供对底层 SQLite 数据库及系统文件的直接控制。</p>
      </div>
    </div>
    
    <div class="page-content">
      <el-tabs v-model="activeTab" class="admin-tabs">
        <el-tab-pane label="SQL 查询终端" name="sql">
          <SqlConsole />
        </el-tab-pane>
        <el-tab-pane label="可视化表结构与数据" name="table">
          <TableExplorer />
        </el-tab-pane>
        <el-tab-pane label="备份与还原" name="backup">
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
    padding: 16px;
    background-color: #fff2f0;
    border: 1px solid #ffccc7;
    border-radius: var(--tm-border-radius-md);

    .header-left {
      h2 {
        margin: 0 0 8px;
        font-size: 20px;
        color: #cf1322;
        display: flex;
        align-items: center;
        gap: 12px;
      }
      .subtitle {
        margin: 0;
        font-size: 14px;
        color: #cf1322;
        opacity: 0.8;
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