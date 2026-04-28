<template>
  <div class="sql-console-container">
    <div class="editor-section">
      <div class="editor-toolbar">
        <el-button type="primary" size="small" @click="executeSQL" :loading="isExecuting" class="run-btn">
          <el-icon><VideoPlay /></el-icon> 执行 (F5)
        </el-button>
        <el-dropdown trigger="click" @command="insertSnippet">
          <el-button size="small" class="snippet-btn">
            常用模板 <el-icon class="el-icon--right"><arrow-down /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="SELECT * FROM raw_data_records LIMIT 10;">查询所有原始数据</el-dropdown-item>
              <el-dropdown-item command="SELECT json_extract(data, '$.title') AS title FROM raw_data_records LIMIT 10;">JSON提取示例</el-dropdown-item>
              <el-dropdown-item command="SELECT name FROM sqlite_master WHERE type='table';">查询所有系统表</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <span class="duration-text" v-if="lastDuration">耗时: {{ lastDuration }}</span>
      </div>
      <div class="codemirror-wrapper">
        <codemirror
          v-model="sqlQuery"
          placeholder="请输入 SQL 语句..."
          :style="{ height: '100%' }"
          :autofocus="true"
          :indent-with-tab="true"
          :tab-size="2"
          :extensions="extensions"
          @keydown.f5.prevent="executeSQL"
        />
      </div>
    </div>
    
    <div class="result-section">
      <div class="result-header">执行结果</div>
      <div class="result-content" v-loading="isExecuting">
        <el-alert
          v-if="errorMessage"
          :title="errorMessage"
          type="error"
          show-icon
          :closable="false"
          class="error-alert"
        />
        <div v-else-if="resultData">
          <div v-if="resultData.is_select" class="table-wrapper">
            <el-table :data="resultData.rows" style="width: 100%" height="100%" border stripe size="small">
              <el-table-column 
                v-for="col in resultData.columns" 
                :key="col" 
                :prop="col" 
                :label="col" 
                show-overflow-tooltip 
              />
            </el-table>
          </div>
          <el-alert
            v-else
            :title="`执行成功。受影响行数: ${resultData.affected}`"
            type="success"
            show-icon
            :closable="false"
            class="success-alert"
          />
        </div>
        <el-empty v-else description="暂无数据" class="empty-state" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { sql } from '@codemirror/lang-sql'
import { ExecuteRawSQL } from '../../../wailsjs/go/main/App'

const sqlQuery = ref('SELECT * FROM sys_datasets;')
const extensions = [sql()]
const isExecuting = ref(false)
const errorMessage = ref('')
const resultData = ref<any>(null)
const lastDuration = ref('')

const insertSnippet = (cmd: string) => {
  sqlQuery.value = cmd
}

const executeSQL = async () => {
  if (!sqlQuery.value.trim()) return
  
  isExecuting.value = true
  errorMessage.value = ''
  lastDuration.value = ''
  resultData.value = null
  
  try {
    const res = await ExecuteRawSQL(sqlQuery.value)
    resultData.value = res
    lastDuration.value = res.duration
  } catch (err: any) {
    errorMessage.value = err.message || err.toString()
  } finally {
    isExecuting.value = false
  }
}
</script>

<style scoped lang="scss">
.sql-console-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  
  .editor-section {
    flex: 1;
    min-height: 250px;
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--tm-border-color);
    
    .editor-toolbar {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 8px 16px;
      background-color: var(--tm-bg-hover);
      border-bottom: 1px solid var(--tm-border-color);
      
      .run-btn {
        display: flex;
        align-items: center;
        gap: 4px;
        font-weight: 600;
      }
      
      .duration-text {
        font-size: 13px;
        color: var(--tm-text-secondary);
        margin-left: auto;
      }
    }
    
    .codemirror-wrapper {
      flex: 1;
      overflow: hidden;
      background-color: var(--tm-bg-card);
      
      :deep(.cm-editor) {
        height: 100%;
        outline: none;
        
        .cm-scroller {
          font-family: 'Consolas', 'Courier New', monospace;
          font-size: 14px;
          line-height: 1.5;
        }
      }
    }
  }
  
  .result-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: var(--tm-bg-main);
    
    .result-header {
      padding: 8px 16px;
      font-size: 14px;
      font-weight: 600;
      color: var(--tm-text-primary);
      border-bottom: 1px solid var(--tm-border-color);
      background-color: var(--tm-bg-hover);
    }
    
    .result-content {
      flex: 1;
      overflow: hidden;
      display: flex;
      flex-direction: column;
      
      .table-wrapper {
        flex: 1;
        overflow: hidden;
      }
      
      .error-alert, .success-alert {
        margin: 16px;
      }
      
      .empty-state {
        margin: auto;
      }
    }
  }
}
</style>