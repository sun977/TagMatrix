<template>
  <div class="sql-console-container">
    <div class="editor-section" :style="{ height: editorHeight + 'px' }">
      <div class="editor-toolbar">
        <div class="toolbar-left">
          <el-button type="primary" @click="executeSQL" :loading="isExecuting" class="toolbar-btn">
            <el-icon><CaretRight /></el-icon> 执行语句
          </el-button>
          <el-button @click="clearSQL" class="toolbar-btn">
            <el-icon><Delete /></el-icon> 清空
          </el-button>
          <el-button @click="formatSQL" class="toolbar-btn">
            <el-icon><Document /></el-icon> 格式化
          </el-button>
          <el-button @click="openSaveDialog" class="toolbar-btn">
            <el-icon><FolderAdd /></el-icon> 保存查询
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-button @click="openTemplateDialog" class="toolbar-btn">
            <el-icon><DocumentCopy /></el-icon> 快捷模板
          </el-button>
          <el-button @click="openHistoryDialog" class="toolbar-btn">
            <el-icon><Clock /></el-icon> 查询历史
          </el-button>
        </div>
      </div>
      
      <div class="editor-header">
        <div class="editor-title">
          SQL 编辑器
          <span v-if="lastDuration" class="duration-info">
            耗时: {{ lastDuration }}
          </span>
        </div>
        <div class="editor-options">
          <span class="option-label">语法高亮</span>
          <el-switch v-model="syntaxHighlight" size="small" />
          <el-button link class="fullscreen-btn" @click="toggleFullscreen">
            <el-icon><FullScreen /></el-icon>
          </el-button>
        </div>
      </div>

      <div class="codemirror-wrapper" :class="{ 'is-fullscreen': isFullscreen }">
        <div v-if="isFullscreen" class="fullscreen-header">
          <div class="header-title">
            <el-icon><Monitor /></el-icon> SQL 编辑器 (全屏模式)
          </div>
          <el-button link @click="toggleFullscreen" class="exit-btn">
            退出
          </el-button>
        </div>
        <codemirror
          v-model="sqlQuery"
          placeholder="请输入 SQL 语句..."
          :style="{ height: '100%' }"
          :autofocus="true"
          :indent-with-tab="true"
          :tab-size="2"
          :extensions="activeExtensions"
          @keydown.f5.prevent="executeSQL"
        />
      </div>
    </div>
    
    <div class="resize-handle" @mousedown="startDrag">
    </div>

    <div class="bottom-section">
      <div class="result-panel">
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
            <template v-if="resultData.is_select">
              <div class="table-wrapper">
                <el-table :data="paginatedRows" style="width: 100%" height="100%" border stripe size="small">
                  <el-table-column
                    v-for="col in resultData.columns"
                    :key="col"
                    :prop="col"
                    :label="col"
                    min-width="150"
                    show-overflow-tooltip
                  />
                </el-table>
              </div>
              <div class="pagination-wrapper">
                <el-button 
                  size="small" 
                  plain 
                  @click="exportToCSV" 
                  :disabled="!resultData?.rows || resultData.rows.length === 0"
                >
                  <el-icon><Download /></el-icon> 导出 CSV
                </el-button>
                <el-pagination
                  :current-page="currentPage"
                  :page-size="pageSize"
                  :page-sizes="[10, 20, 50, 100, 200, 500]"
                  layout="total, sizes, prev, pager, next"
                  :total="totalRows"
                  @update:current-page="currentPage = $event"
                  @update:page-size="pageSize = $event"
                />
              </div>
            </template>
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
    <!-- 历史记录管理对话框 -->
    <el-dialog v-model="historyDialogVisible" title="查询历史" width="700px" destroy-on-close>
      <div class="template-dialog-toolbar" style="margin-bottom: 16px; display: flex; justify-content: flex-end;">
        <el-button type="danger" plain @click="clearHistory" :disabled="queryHistory.length === 0">
          清空历史记录
        </el-button>
      </div>

      <el-table :data="queryHistory" style="width: 100%" max-height="400" border stripe>
        <el-table-column prop="time" label="执行时间" width="180" />
        <el-table-column prop="query" label="SQL 语句" show-overflow-tooltip>
          <template #default="scope">
            <span style="font-family: monospace; font-size: 12px;">{{ scope.row.query }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="scope">
            <el-button type="primary" link size="small" @click="applyHistory(scope.row.query)">应用</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="historyDialogVisible = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { List, Search, VideoPlay, Delete, Download, CaretRight, Document, FolderAdd, DocumentCopy, Clock, FullScreen, Monitor, Timer, Close } from '@element-plus/icons-vue'
import { Codemirror } from 'vue-codemirror'
import { sql } from '@codemirror/lang-sql'
import { tags as t } from '@lezer/highlight'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { EditorView } from '@codemirror/view'
import { ExecuteRawSQL, GetSqlTemplates, SaveSqlTemplate, DeleteSqlTemplate } from '../../../wailsjs/go/main/App'

const sqlQuery = ref('SELECT * FROM sys_datasets;')

const customTheme = EditorView.theme({
  "&": {
    fontSize: "15px",
  },
  ".cm-content": {
    fontFamily: "'Consolas', 'Courier New', monospace",
    letterSpacing: "0.5px",
    lineHeight: "1.6"
  }
})

const myHighlightStyle = HighlightStyle.define([
  { tag: t.keyword, color: '#005cc5', fontWeight: 'bold' }, 
  { tag: t.operator, color: '#d73a49', fontWeight: 'bold' }, 
  { tag: t.string, color: '#22863a', fontWeight: 'bold' }, 
  { tag: t.number, color: '#005cc5', fontWeight: 'bold' }, 
  { tag: t.comment, color: '#6a737d', fontStyle: 'italic' }, 
  { tag: t.punctuation, color: '#24292e', fontWeight: 'bold' }, 
  { tag: t.variableName, color: '#e36209', fontWeight: 'bold' }, 
  { tag: t.function(t.variableName), color: '#6f42c1', fontWeight: 'bold' }, 
  { tag: t.typeName, color: '#6f42c1', fontWeight: 'bold' },
  { tag: t.null, color: '#005cc5', fontWeight: 'bold' }
])

const baseExtensions = [sql(), customTheme]
const highlightExtensions = [syntaxHighlighting(myHighlightStyle)]

const isExecuting = ref(false)
const errorMessage = ref('')
const resultData = ref<any>(null)
const lastDuration = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(50)
const totalRows = computed(() => resultData.value?.rows?.length || 0)
const paginatedRows = computed(() => {
  if (!resultData.value?.rows) return []
  const start = (currentPage.value - 1) * pageSize.value
  return resultData.value.rows.slice(start, start + pageSize.value)
})

import { SaveCSVFile } from '../../../wailsjs/go/main/App'

// 导出 CSV 功能
const exportToCSV = async () => {
  if (!resultData.value?.rows || resultData.value.rows.length === 0) {
    ElMessage.warning('暂无数据可导出')
    return
  }

  const columns = resultData.value.columns
  const rows = resultData.value.rows

  // 格式化 CSV 单元格内容
  const escapeCSV = (field: any) => {
    if (field === null || field === undefined) return ''
    const str = String(field)
    // 包含逗号、双引号或换行符的字段，需要用双引号包围，并将内部双引号转义
    if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
      return `"${str.replace(/"/g, '""')}"`
    }
    return str
  }

  const headers = columns.map((col: string) => escapeCSV(col)).join(',')
  const body = rows.map((row: any) => {
    return columns.map((col: string) => escapeCSV(row[col])).join(',')
  }).join('\n')

  // 加入 BOM (\uFEFF) 防止 Excel 乱码
  const csvContent = '\uFEFF' + headers + '\n' + body

  // 生成时间戳文件名
  const now = new Date()
  const pad = (n: number) => n.toString().padStart(2, '0')
  const timestamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  const fileName = `query_result_${timestamp}.csv`
  
  try {
    const savedPath = await SaveCSVFile(fileName, csvContent)
    if (savedPath) {
      ElMessage.success(`导出成功：${savedPath}`)
    } else {
      ElMessage.info('已取消导出')
    }
  } catch (err: any) {
    ElMessage.error(`导出失败：${err.message || err}`)
  }
}

const sqlTemplates = ref<any[]>([])
const saveDialogVisible = ref(false)
const isSaving = ref(false)
const saveForm = ref({ name: '' })

// 语法高亮
const syntaxHighlight = ref(true)
const activeExtensions = computed(() => {
  return syntaxHighlight.value ? [...baseExtensions, ...highlightExtensions] : baseExtensions
})

// 全屏
const isFullscreen = ref(false)
const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  if (isFullscreen.value) {
    window.addEventListener('keydown', handleEsc)
  } else {
    window.removeEventListener('keydown', handleEsc)
  }
}

const handleEsc = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    toggleFullscreen()
  }
}

onUnmounted(() => {
  window.removeEventListener('keydown', handleEsc)
})

// 格式化
const formatSQL = () => {
  let formatted = sqlQuery.value
    .replace(/\s+/g, ' ')
    .replace(/\b(select|from|where|and|or|order by|group by|limit|insert into|update|set|delete from)\b/gi, match => '\n' + match.toUpperCase())
  sqlQuery.value = formatted.trim()
}

// 历史记录
const historyDialogVisible = ref(false)
const queryHistory = ref<any[]>([])

const loadHistory = () => {
  const h = localStorage.getItem('sql_query_history')
  if (h) {
    try { queryHistory.value = JSON.parse(h) } catch (e) {}
  }
}

const saveToHistory = (q: string) => {
  if (!q.trim()) return
  queryHistory.value = queryHistory.value.filter(item => item.query !== q)
  queryHistory.value.unshift({ query: q, time: new Date().toLocaleString() })
  if (queryHistory.value.length > 50) {
    queryHistory.value = queryHistory.value.slice(0, 50)
  }
  localStorage.setItem('sql_query_history', JSON.stringify(queryHistory.value))
}

const openHistoryDialog = () => {
  loadHistory()
  historyDialogVisible.value = true
}

const applyHistory = (q: string) => {
  sqlQuery.value = q
  historyDialogVisible.value = false
}

const clearHistory = () => {
  queryHistory.value = []
  localStorage.removeItem('sql_query_history')
}

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
    currentPage.value = 1
    lastDuration.value = res.duration
    saveToHistory(sqlQuery.value)
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
    background-color: var(--tm-bg-card);
    flex-shrink: 0;
    
    .editor-toolbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      border-bottom: 1px solid var(--tm-border-color);
      
      .toolbar-left, .toolbar-right {
        display: flex;
        gap: 12px;
        align-items: center;
      }
      
      .toolbar-btn {
        border-radius: 6px;
        padding: 8px 16px;
        font-weight: 500;
        
        :deep(.el-icon) {
          margin-right: 4px;
        }
      }
    }
    
      .editor-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 16px;
      background-color: var(--tm-bg-hover);
      border-bottom: 1px solid var(--tm-border-color);
      
      .editor-title {
        font-size: 13px;
        font-weight: 600;
        color: var(--tm-text-primary);
        display: flex;
        align-items: center;
        gap: 6px;

        .duration-info {
          margin-left: 8px;
          font-size: 12px;
          font-weight: normal;
          color: var(--el-color-success);
          display: flex;
          align-items: center;
          gap: 4px;
          background: var(--el-color-success-light-9);
          padding: 2px 8px;
          border-radius: 4px;
        }
      }
      
      .editor-options {
        display: flex;
        align-items: center;
        gap: 12px;

        .option-label {
          font-size: 13px;
          color: var(--tm-text-secondary);
        }
        
        .fullscreen-btn {
          color: var(--tm-text-secondary);
          &:hover {
            color: var(--tm-text-primary);
          }
        }
      }
    }
    
    .codemirror-wrapper {
      flex: 1;
      overflow: hidden;
      border-bottom: 1px solid var(--tm-border-color);
      
      &.is-fullscreen {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 9999;
        height: 100vh !important;
        background: var(--tm-bg-main);
        display: flex;
        flex-direction: column;
        
        .fullscreen-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 16px;
          background-color: var(--tm-bg-card);
          border-bottom: 1px solid var(--tm-border-color);
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
          
          .header-title {
            font-size: 14px;
            font-weight: 600;
            color: var(--tm-text-primary);
            display: flex;
            align-items: center;
            gap: 6px;
          }
          
          .exit-btn {
            font-size: 14px;
            color: var(--tm-text-secondary);
            
            &:hover {
              color: var(--el-color-danger);
            }
          }
        }
      }
      
      :deep(.cm-editor) {
        height: 100%;
        outline: none;
        background-color: var(--tm-bg-card) !important;
        color: var(--tm-text-primary) !important;
        
        .cm-scroller {
          font-family: 'Consolas', 'Courier New', monospace;
          font-size: 14px;
          line-height: 1.5;
        }
        
        .cm-gutters {
          background-color: var(--tm-bg-subtle) !important;
          color: var(--tm-text-secondary) !important;
          border-right: 1px solid var(--tm-border-color) !important;
        }

        .cm-activeLineGutter, .cm-activeLine {
          background-color: var(--tm-bg-hover) !important;
        }
      }
    }
  }

  .resize-handle {
    height: 6px;
    background-color: var(--tm-bg-main);
    cursor: row-resize;
    transition: background-color 0.2s;
    
    &:hover {
      background-color: var(--el-color-primary-light-8);
    }
  }
  
    .bottom-section {
      flex: 1;
      height: 0;
      min-height: 0;
      display: flex;
      overflow: hidden;
      position: relative;

      .result-panel {
        flex: 1;
        min-height: 0;
        height: 100%;
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
        position: relative;
        overflow: hidden;
        
        .result-data-wrapper {
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          display: flex;
          flex-direction: column;
          
          .table-wrapper {
            flex: 1;
            overflow: hidden;
          }
      
      .pagination-wrapper {
        padding: 8px 12px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        background-color: var(--tm-bg-card);
        border-top: 1px solid var(--tm-border-color);
      }
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