<template>
  <div class="sql-console-container">
    <div class="editor-section" :style="{ height: editorHeight + 'px' }">
      <div class="editor-toolbar">
        <div class="toolbar-left">
          <el-button type="primary" @click="executeSQL" :loading="isExecuting">
            <el-icon><VideoPlay /></el-icon> 执行查询
          </el-button>
          <el-button @click="clearSQL">
            <el-icon><Delete /></el-icon> 清空
          </el-button>
          <el-button @click="openSaveDialog">
            <el-icon><Download /></el-icon> 保存查询
          </el-button>

          <el-button @click="openTemplateDialog" type="info" plain style="margin-left: 8px;">
            常用模板 <el-icon class="el-icon--right"><List /></el-icon>
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
    
    <div class="resize-handle" @mousedown="startDrag">
      <div class="handle-line"></div>
    </div>

    <div class="bottom-section">
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
    <!-- 常用模板管理对话框 -->
    <el-dialog v-model="templateDialogVisible" title="常用 SQL 模板管理" width="700px" destroy-on-close>
      <div class="template-dialog-toolbar" style="margin-bottom: 16px; display: flex; justify-content: flex-end;">
        <el-input v-model="templateSearchKeyword" placeholder="模糊搜索模板名称或内容" style="width: 250px" clearable>
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <el-table :data="filteredTemplates" style="width: 100%" max-height="400" border stripe>
        <el-table-column prop="name" label="模板名称" width="150" show-overflow-tooltip />
        <el-table-column prop="query" label="SQL 语句" show-overflow-tooltip>
          <template #default="scope">
            <span style="font-family: monospace; font-size: 12px;">{{ scope.row.query }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="scope">
            <el-button type="primary" link size="small" @click="applyTemplate(scope.row)">应用</el-button>
            <el-button type="warning" link size="small" @click="editTemplate(scope.row)">编辑</el-button>
            <el-button type="danger" link size="small" @click="deleteTemplate(scope.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="templateDialogVisible = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 编辑模板对话框 -->
    <el-dialog v-model="editTemplateDialogVisible" title="编辑 SQL 模板" width="500px">
      <el-form :model="editTemplateForm" label-width="80px">
        <el-form-item label="模板名称">
          <el-input v-model="editTemplateForm.name" placeholder="请输入模板名称" maxlength="50" show-word-limit />
        </el-form-item>
        <el-form-item label="SQL 语句">
          <el-input v-model="editTemplateForm.query" type="textarea" :rows="6" placeholder="请输入 SQL 语句" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editTemplateDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveEditTemplate" :loading="isSavingEdit">保存</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { List, Search, VideoPlay, Delete, Download } from '@element-plus/icons-vue'
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

// 模板管理相关状态
const templateDialogVisible = ref(false)
const templateSearchKeyword = ref('')
const editTemplateDialogVisible = ref(false)
const isSavingEdit = ref(false)
const editTemplateForm = ref({ id: 0, name: '', query: '' })

const filteredTemplates = computed(() => {
  if (!templateSearchKeyword.value) {
    return sqlTemplates.value
  }
  const keyword = templateSearchKeyword.value.toLowerCase()
  return sqlTemplates.value.filter(tpl => 
    tpl.name.toLowerCase().includes(keyword) || 
    tpl.query.toLowerCase().includes(keyword)
  )
})

const openTemplateDialog = () => {
  templateSearchKeyword.value = ''
  templateDialogVisible.value = true
}

const applyTemplate = (tpl: any) => {
  sqlQuery.value = tpl.query
  templateDialogVisible.value = false
  ElMessage.success('已应用模板')
}

const editTemplate = (tpl: any) => {
  editTemplateForm.value = { ...tpl }
  editTemplateDialogVisible.value = true
}

const saveEditTemplate = async () => {
  if (!editTemplateForm.value.name.trim() || !editTemplateForm.value.query.trim()) {
    ElMessage.warning('名称和内容不能为空')
    return
  }
  isSavingEdit.value = true
  try {
    await SaveSqlTemplate(editTemplateForm.value.id, editTemplateForm.value.name, editTemplateForm.value.query)
    ElMessage.success('保存成功')
    editTemplateDialogVisible.value = false
    loadTemplates()
  } catch (e: any) {
    ElMessage.error('保存失败: ' + e.message)
  } finally {
    isSavingEdit.value = false
  }
}

const editorHeight = ref(300)
let isDragging = false
let startY = 0
let startHeight = 0

const startDrag = (e: MouseEvent) => {
  isDragging = true
  startY = e.clientY
  startHeight = editorHeight.value
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
}

const onDrag = (e: MouseEvent) => {
  if (!isDragging) return
  const dy = e.clientY - startY
  const newHeight = startHeight + dy
  if (newHeight > 100 && newHeight < window.innerHeight - 200) {
    editorHeight.value = newHeight
  }
}

const stopDrag = () => {
  isDragging = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
})

const handleTemplateCommand = (tpl: any) => {
  useTemplate(tpl)
}

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
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--tm-border-color);
    background-color: var(--tm-bg-card);
    flex-shrink: 0;
    
    .editor-toolbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 8px 16px;
      border-bottom: 1px solid var(--tm-border-color);
      
      .toolbar-left {
        display: flex;
        gap: 8px;
        align-items: center;
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

  .resize-handle {
    height: 10px;
    background-color: var(--tm-bg-main);
    cursor: row-resize;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background-color 0.2s;

    &:hover {
      background-color: #e4e7ed;
    }

    .handle-line {
      width: 40px;
      height: 4px;
      background-color: #dcdfe6;
      border-radius: 2px;
    }
  }
  
  .bottom-section {
    flex: 1;
    display: flex;
    overflow: hidden;
    
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

.tpl-dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 200px;

  .tpl-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--tm-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}
</style>