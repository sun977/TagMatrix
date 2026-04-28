<template>
  <div class="sql-console-container">
    <div class="editor-section">
      <div class="editor-toolbar">
        <div class="toolbar-left">
          <el-button type="primary" @click="executeSQL" :loading="isExecuting">
            <el-icon><VideoPlay /></el-icon> 执行查询
          </el-button>
          <el-button @click="clearSQL">
            <el-icon><Delete /></el-icon> 清空
          </el-button>
          <!-- <el-button @click="formatSQL">
            <el-icon><Document /></el-icon> 格式化
          </el-button> -->
          <el-button @click="openSaveDialog">
            <el-icon><Download /></el-icon> 保存查询
          </el-button>
        </div>
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
    
    <div class="bottom-section">
      <div class="templates-panel">
        <div class="panel-header">
          常用 SQL 模板
        </div>
        <div class="templates-list">
          <div 
            v-for="tpl in sqlTemplates" 
            :key="tpl.id" 
            class="template-item"
            @click="useTemplate(tpl)"
          >
            <div class="tpl-content">
              <div class="tpl-name">{{ tpl.name }}</div>
              <div class="tpl-query">{{ tpl.query }}</div>
            </div>
            <div class="tpl-actions" @click.stop>
              <el-button link type="danger" size="small" @click="deleteTemplate(tpl.id)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <el-empty v-if="sqlTemplates.length === 0" description="暂无保存的模板" :image-size="60" />
        </div>
      </div>

      <div class="result-panel">
        <div class="panel-header result-header-row">
          <div class="result-title">
            <el-icon><List /></el-icon> 查询结果
            <span v-if="lastDuration" class="duration-badge">耗时: {{ lastDuration }}</span>
          </div>
        </div>
        <div class="result-content" v-loading="isExecuting">
          <el-alert
            v-if="errorMessage"
            :title="errorMessage"
            type="error"
            show-icon
            :closable="false"
            class="error-alert"
          />
          <div v-else-if="resultData" class="result-data-wrapper">
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
          <el-empty v-else description="暂无查询结果" class="empty-state" :image-size="80" />
        </div>
      </div>
    </div>

    <!-- 保存查询对话框 -->
    <el-dialog v-model="saveDialogVisible" title="保存 SQL 查询" width="400px">
      <el-form :model="saveForm" label-width="80px">
        <el-form-item label="模板名称">
          <el-input v-model="saveForm.name" placeholder="请输入模板名称" maxlength="50" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="saveDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveSQL" :loading="isSaving">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Codemirror } from 'vue-codemirror'
import { sql } from '@codemirror/lang-sql'
import { ExecuteRawSQL, GetSqlTemplates, SaveSqlTemplate, DeleteSqlTemplate } from '../../../wailsjs/go/main/App'

const sqlQuery = ref('SELECT * FROM sys_datasets;')
const extensions = [sql()]
const isExecuting = ref(false)
const errorMessage = ref('')
const resultData = ref<any>(null)
const lastDuration = ref('')

const sqlTemplates = ref<any[]>([])
const saveDialogVisible = ref(false)
const isSaving = ref(false)
const saveForm = ref({ name: '' })

const loadTemplates = async () => {
  try {
    const res = await GetSqlTemplates()
    sqlTemplates.value = res || []
  } catch (e: any) {
    ElMessage.error('获取模板失败: ' + e.message)
  }
}

const clearSQL = () => {
  sqlQuery.value = ''
}

const useTemplate = (tpl: any) => {
  sqlQuery.value = tpl.query
}

const openSaveDialog = () => {
  if (!sqlQuery.value.trim()) {
    ElMessage.warning('查询语句为空，无法保存')
    return
  }
  saveForm.value.name = ''
  saveDialogVisible.value = true
}

const saveSQL = async () => {
  if (!saveForm.value.name.trim()) {
    ElMessage.warning('请输入模板名称')
    return
  }
  isSaving.value = true
  try {
    await SaveSqlTemplate(0, saveForm.value.name, sqlQuery.value)
    ElMessage.success('保存成功')
    saveDialogVisible.value = false
    loadTemplates()
  } catch (e: any) {
    ElMessage.error('保存失败: ' + e.message)
  } finally {
    isSaving.value = false
  }
}

const deleteTemplate = (id: number) => {
  ElMessageBox.confirm('确定要删除这个查询模板吗？', '提示', { type: 'warning' })
    .then(async () => {
      try {
        await DeleteSqlTemplate(id)
        ElMessage.success('删除成功')
        loadTemplates()
      } catch (e: any) {
        ElMessage.error('删除失败: ' + e.message)
      }
    })
    .catch(() => {})
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

onMounted(() => {
  loadTemplates()
})
</script>

<style scoped lang="scss">
.sql-console-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  
  .editor-section {
    flex: 0 0 40%;
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--tm-border-color);
    background-color: var(--tm-bg-card);
    
    .editor-toolbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 8px 16px;
      border-bottom: 1px solid var(--tm-border-color);
      
      .toolbar-left {
        display: flex;
        gap: 8px;
      }
    }
    
    .codemirror-wrapper {
      flex: 1;
      overflow: hidden;
      
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
  
  .bottom-section {
    flex: 1;
    display: flex;
    overflow: hidden;
    
    .templates-panel {
      flex: 0 0 280px;
      border-right: 1px solid var(--tm-border-color);
      display: flex;
      flex-direction: column;
      background-color: var(--tm-bg-main);
      
      .panel-header {
        padding: 12px 16px;
        font-size: 14px;
        font-weight: 600;
        color: var(--tm-text-primary);
        border-bottom: 1px solid var(--tm-border-color);
        background-color: var(--tm-bg-hover);
      }
      
      .templates-list {
        flex: 1;
        overflow-y: auto;
        padding: 8px 0;
        
        .template-item {
          padding: 12px 16px;
          cursor: pointer;
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          transition: background-color 0.2s;
          border-bottom: 1px solid var(--tm-border-color-light);
          
          &:hover {
            background-color: var(--tm-bg-hover);
          }
          
          .tpl-content {
            flex: 1;
            overflow: hidden;
            
            .tpl-name {
              font-size: 14px;
              font-weight: 500;
              color: var(--tm-text-primary);
              margin-bottom: 4px;
            }
            
            .tpl-query {
              font-size: 12px;
              color: var(--tm-text-secondary);
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
              font-family: monospace;
            }
          }
          
          .tpl-actions {
            margin-left: 8px;
            opacity: 0;
            transition: opacity 0.2s;
          }
          
          &:hover .tpl-actions {
            opacity: 1;
          }
        }
      }
    }
    
    .result-panel {
      flex: 1;
      display: flex;
      flex-direction: column;
      background-color: var(--tm-bg-card);
      overflow: hidden;
      
      .result-header-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        font-size: 14px;
        font-weight: 600;
        color: var(--tm-text-primary);
        border-bottom: 1px solid var(--tm-border-color);
        background-color: var(--tm-bg-hover);
        
        .result-title {
          display: flex;
          align-items: center;
          gap: 6px;
        }
        
        .duration-badge {
          font-weight: normal;
          font-size: 12px;
          color: var(--tm-accent-primary);
          background-color: rgba(82, 196, 143, 0.1);
          padding: 2px 8px;
          border-radius: 12px;
          margin-left: 8px;
        }
      }
      
      .result-content {
        flex: 1;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        
        .result-data-wrapper {
          flex: 1;
          overflow: hidden;
          display: flex;
          flex-direction: column;
        }
        
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
}
</style>