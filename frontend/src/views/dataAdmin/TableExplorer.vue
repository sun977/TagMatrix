<template>
  <div class="table-explorer-container">
    <div class="sidebar">
      <el-tabs v-model="tabMode" class="mode-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="系统物理表" name="system">
          <ul class="table-list">
            <li v-for="t in systemTables" :key="t.name" 
                :class="{ active: currentTable === t.name }"
                @click="selectTable(t.name)">
              <el-icon><DataBoard /></el-icon> {{ t.name }}
            </li>
          </ul>
        </el-tab-pane>
        <el-tab-pane label="业务数据集" name="dataset">
          <ul class="table-list">
            <li v-for="ds in datasets" :key="ds.id" 
                :class="{ active: currentDatasetId === ds.id }"
                @click="selectDataset(ds)">
              <el-icon><Folder /></el-icon> {{ ds.name }}
            </li>
          </ul>
        </el-tab-pane>
      </el-tabs>
    </div>

    <div class="main-content">
      <div class="toolbar">
        <div class="title-area">
          <h3 v-if="tabMode === 'system'">{{ currentTable || '未选择物理表' }}</h3>
          <h3 v-else>{{ currentDatasetName || '未选择数据集' }}</h3>
          <el-tag size="small" type="info" v-if="currentTable || currentDatasetId">共有 {{ totalRecords }} 条记录</el-tag>
        </div>
        <div class="actions">
          <el-button type="primary" size="small" @click="fetchData" :disabled="!currentTable && !currentDatasetId">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
<el-button type="success" size="small" @click="handleAddRow" :disabled="!currentTable && !currentDatasetId">
  <el-icon><Plus /></el-icon> 新增行
</el-button>
        </div>
      </div>

    <div class="table-area" v-loading="loading">
      <div class="table-absolute-wrapper">
        <el-table :data="tableData" style="width: 100%" height="100%" border stripe size="small" @row-dblclick="handleRowDblClick">
          <el-table-column v-for="col in columns" :key="col" :prop="col" :label="col" show-overflow-tooltip>
            <template #default="scope">
              <span v-if="!scope.row._editing || editingCell.rowId !== scope.row.id || editingCell.col !== col">{{ scope.row[col] }}</span>
              <el-input
                v-else
                v-model="scope.row[col]"
                size="small"
                ref="editInput"
                @blur="saveCellEdit(scope.row, col)"
                @keyup.enter="saveCellEdit(scope.row, col)"
              />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="scope">
              <el-button link type="danger" size="small" @click="deleteRecord(scope.row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <div class="pagination-area">
      <div class="table-actions-left">
        <el-upload
          class="inline-upload"
          action="#"
          :auto-upload="false"
          :show-file-list="false"
          accept=".csv"
          :on-change="handleImportCSV"
          :disabled="!currentTable && !currentDatasetId"
        >
          <el-button size="small" type="primary" plain :disabled="!currentTable && !currentDatasetId">
            <el-icon><Upload /></el-icon> 导入 CSV
          </el-button>
        </el-upload>
        <el-button size="small" plain @click="exportToCSV" :disabled="tableData.length === 0">
          <el-icon><Download /></el-icon> 导出当前页
        </el-button>
      </div>
      <el-pagination
        :current-page="currentPage"
        :page-size="pageSize"
        :page-sizes="[50, 100, 200]"
        layout="total, sizes, prev, pager, next"
        :total="totalRecords"
        @update:current-page="currentPage = $event"
        @update:page-size="pageSize = $event"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
    </div>

    <!-- 新增行对话框 -->
    <el-dialog v-model="addRowDialogVisible" :title="tabMode === 'system' ? '新增物理表记录' : '新增业务数据'" width="500px">
      <el-form :model="newRowForm" label-width="120px">
        <el-form-item v-for="col in editableColumns" :key="col" :label="col">
          <el-input v-model="newRowForm[col]" placeholder="请输入内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="addRowDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitAddRow" :loading="loading">保存</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type UploadFile } from 'element-plus'
import { DataBoard, Folder, Refresh, Plus, Download, Upload } from '@element-plus/icons-vue'
import { GetSystemTables, GetTableData, ListDatasets, GetVirtualDatasetData, UpdateVirtualRecord, DeleteVirtualRecord, UpdateSystemTableRecord, DeleteSystemTableRecord, InsertSystemTableRecord, InsertVirtualRecord } from '../../../wailsjs/go/main/App'

const tabMode = ref('system')
const systemTables = ref<{name: string}[]>([])
const datasets = ref<any[]>([])

const currentTable = ref('')
const currentDatasetId = ref(0)
const currentDatasetName = ref('')

const tableData = ref<any[]>([])
const columns = ref<string[]>([])
const totalRecords = ref(0)

const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(50)

const editingCell = ref({ rowId: null, col: '' })

const addRowDialogVisible = ref(false)
const newRowForm = ref<Record<string, any>>({})

// 过滤掉新增行时不需要用户填写的系统自动维护字段
const editableColumns = computed(() => {
  const readonlyFields = ['id', 'created_at', 'updated_at', 'deleted_at']
  return columns.value.filter(col => !readonlyFields.includes(col.toLowerCase()))
})

const handleAddRow = () => {
  newRowForm.value = {}
  for (const col of editableColumns.value) {
    newRowForm.value[col] = ''
  }
  addRowDialogVisible.value = true
}

const submitAddRow = async () => {
  loading.value = true
  try {
    const payload = { ...newRowForm.value }
    for (const k in payload) {
      if (payload[k] === '') {
        delete payload[k]
      }
    }
    
    if (tabMode.value === 'system') {
      await InsertSystemTableRecord(currentTable.value, payload)
    } else {
      await InsertVirtualRecord(currentDatasetId.value, payload)
    }
    ElMessage.success('新增成功')
    addRowDialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e.message || '新增失败')
  } finally {
    loading.value = false
  }
}

const loadSidebar = async () => {
  if (tabMode.value === 'system') {
    try {
      const res = await GetSystemTables()
      systemTables.value = res.map((name: string) => ({ name }))
      if (systemTables.value.length > 0 && !currentTable.value) {
        selectTable(systemTables.value[0].name)
      }
    } catch (e: any) {
      ElMessage.error(e.message || '加载系统表失败')
    }
  } else {
    try {
      const res = await ListDatasets()
      datasets.value = res || []
      if (datasets.value.length > 0 && !currentDatasetId.value) {
        selectDataset(datasets.value[0])
      }
    } catch (e: any) {
      ElMessage.error(e.message || '加载数据集失败')
    }
  }
}

const handleTabChange = () => {
  tableData.value = []
  columns.value = []
  totalRecords.value = 0
  currentPage.value = 1
  loadSidebar()
}

const selectTable = (name: string) => {
  currentTable.value = name
  currentPage.value = 1
  fetchData()
}

const selectDataset = (ds: any) => {
  currentDatasetId.value = ds.id
  currentDatasetName.value = ds.name
  currentPage.value = 1
  fetchData()
}

const fetchData = async () => {
  loading.value = true
  const offset = (currentPage.value - 1) * pageSize.value
  try {
    if (tabMode.value === 'system' && currentTable.value) {
      const res = await GetTableData(currentTable.value, offset, pageSize.value)
      columns.value = res.columns || []
      tableData.value = res.rows || []
      totalRecords.value = res.total || 0
    } else if (tabMode.value === 'dataset' && currentDatasetId.value) {
      const res = await GetVirtualDatasetData(currentDatasetId.value, offset, pageSize.value)
      columns.value = res.columns || []
      tableData.value = res.rows || []
      totalRecords.value = res.total || 0
    }
  } catch (e: any) {
    ElMessage.error(e.message || '加载数据失败')
  } finally {
    loading.value = false
  }
}

const handleSizeChange = () => {
  currentPage.value = 1
  fetchData()
}

const handleCurrentChange = () => {
  fetchData()
}

import { SaveCSVFile } from '../../../wailsjs/go/main/App'

// ========================
// 导入导出 CSV
// ========================

const exportToCSV = async () => {
  if (tableData.value.length === 0) {
    ElMessage.warning('当前没有数据可以导出')
    return
  }

  const escapeCSV = (field: any) => {
    if (field === null || field === undefined) return ''
    const str = String(field)
    if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
      return `"${str.replace(/"/g, '""')}"`
    }
    return str
  }

  const headers = columns.value.map(col => escapeCSV(col)).join(',')
  const body = tableData.value.map((row: any) => {
    return columns.value.map(col => escapeCSV(row[col])).join(',')
  }).join('\n')

  const csvContent = '\uFEFF' + headers + '\n' + body
  
  const now = new Date()
  const pad = (n: number) => n.toString().padStart(2, '0')
  const timestamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  const fileName = currentTable.value ? `sys_table_${currentTable.value}_${timestamp}.csv` : `dataset_${currentDatasetId.value}_${timestamp}.csv`
  
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

// 解析并插入记录（防脏数据注入）
const handleImportCSV = (file: UploadFile) => {
  if (!file.raw) return
  const reader = new FileReader()
  
  reader.onload = async (e) => {
    const text = e.target?.result as string
    if (!text) return
    
    // 简易 CSV 分行
    const lines = text.split('\n').map(l => l.trim()).filter(l => l.length > 0)
    if (lines.length < 2) {
      ElMessage.warning('CSV 文件内容为空或缺少表头')
      return
    }
    
    // 去除 UTF-8 BOM
    let headerLine = lines[0]
    if (headerLine.charCodeAt(0) === 0xFEFF) {
      headerLine = headerLine.substring(1)
    }
    
    const headers = headerLine.split(',').map(h => h.replace(/^"|"$/g, '').trim())
    
    let successCount = 0
    let errorCount = 0
    
    loading.value = true
    try {
      for (let i = 1; i < lines.length; i++) {
        const line = lines[i]
        // 简易逗号拆分，不涉及复杂的带逗号双引号字符串内包含的情况
        // 复杂场景需要 PapaParse 等库支持
        const values = line.split(',').map(v => v.replace(/^"|"$/g, '').trim())
        const record: Record<string, any> = {}
        
        headers.forEach((h, idx) => {
          // 清洗防护: 跳过 id、deleted_at。 created_at/updated_at 交给后端或者这里跳过。
          if (['id', 'created_at', 'updated_at', 'deleted_at'].includes(h.toLowerCase())) {
            return // 跳过内置 ID 和时间戳字段，屏蔽脏数据
          }
          const val = values[idx]
          if (val !== undefined && val !== '') {
            record[h] = val
          }
        })
        
        // 空记录拦截
        if (Object.keys(record).length === 0) continue

        if (tabMode.value === 'system' && currentTable.value) {
          await InsertSystemTableRecord(currentTable.value, record)
          successCount++
        } else if (tabMode.value === 'dataset' && currentDatasetId.value) {
          await InsertVirtualRecord(currentDatasetId.value, record)
          successCount++
        }
      }
      
      ElMessage.success(`导入完成: 成功插入 ${successCount} 条`)
    } catch (err: any) {
      console.error('Import error', err)
      ElMessage.error(`导入出现异常: ${err.message || err.toString()}`)
    } finally {
      loading.value = false
      fetchData()
    }
  }
  
  reader.readAsText(file.raw)
}

const handleRowDblClick = (row: any, column: any) => {
  if (tabMode.value === 'system' && (!row.id || !column.property || column.property === 'id')) {
    // Only warn if we cannot identify the row by id
    if (!row.id) {
      ElMessage.warning('当前物理表没有 id 列，暂不支持内联编辑')
      return
    }
  }
  // Ignore operation column and ID column
  if (!column.property || column.property === 'id') return
  
  row._editing = true
  row._originalValue = row[column.property]
  editingCell.value = { rowId: row.id, col: column.property }
  // Auto focus logic can be added here
}

const saveCellEdit = async (row: any, col: string) => {
  if (!row._editing) return
  row._editing = false
  editingCell.value = { rowId: null, col: '' }
  
  if (row[col] === row._originalValue) return // no change
  
  loading.value = true
  try {
    if (tabMode.value === 'system') {
      await UpdateSystemTableRecord(currentTable.value, row.id, { [col]: row[col] })
    } else {
      await UpdateVirtualRecord(row.id, { [col]: row[col] })
    }
    ElMessage.success('更新成功')
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
    row[col] = row._originalValue // revert
  } finally {
    loading.value = false
  }
}

const deleteRecord = (row: any) => {
  ElMessageBox.confirm('确定要删除这条记录吗？此操作不可恢复。', '警告', {
    type: 'warning'
  }).then(async () => {
    loading.value = true
    try {
      if (tabMode.value === 'system') {
        if (!row.id) {
          throw new Error("当前物理表没有 id 列，无法删除")
        }
        await DeleteSystemTableRecord(currentTable.value, row.id)
        ElMessage.success('删除成功')
        fetchData()
      } else {
        await DeleteVirtualRecord(row.id)
        ElMessage.success('删除成功')
        fetchData()
      }
    } catch (e: any) {
      ElMessage.error(e.message || '删除失败')
    } finally {
      loading.value = false
    }
  }).catch(() => {})
}

onMounted(() => {
  loadSidebar()
})
</script>

<style scoped lang="scss">
.table-explorer-container {
  display: flex;
  height: 100%;
  
  .sidebar {
    width: 250px;
    border-right: 1px solid var(--tm-border-color);
    background-color: var(--tm-bg-card);
    display: flex;
    flex-direction: column;
    
    .mode-tabs {
      flex: 1;
      display: flex;
      flex-direction: column;
      
      :deep(.el-tabs__header) {
        margin-bottom: 0;
      }
      
      :deep(.el-tabs__content) {
        flex: 1;
        overflow-y: auto;
        padding: 0;
      }
    }
    
    .table-list {
      list-style: none;
      padding: 0;
      margin: 0;
      
      li {
        padding: 12px 16px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        color: var(--tm-text-regular);
        border-bottom: 1px solid var(--tm-border-color-light);
        transition: all 0.2s ease;
        
        &:hover {
          background-color: var(--tm-bg-hover);
        }
        
        &.active {
          background-color: var(--tm-bg-active);
          color: var(--tm-text-primary);
          font-weight: 600;
          border-right: 3px solid var(--tm-accent-primary);
        }
      }
    }
  }
  
  .main-content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background-color: var(--tm-bg-main);
    
    .toolbar {
      padding: 12px 20px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      background-color: var(--tm-bg-card);
      border-bottom: 1px solid var(--tm-border-color);
      
      .title-area {
        display: flex;
        align-items: center;
        gap: 12px;
        
        h3 {
          margin: 0;
          font-size: 16px;
          color: var(--tm-text-primary);
        }
      }
      
      .actions {
        display: flex;
        gap: 8px;
      }
    }
    
  .table-area {
    flex: 1;
    position: relative;
    min-height: 0;
    overflow: hidden;
    
    .table-absolute-wrapper {
      position: absolute;
      top: 16px;
      left: 16px;
      right: 16px;
      bottom: 16px;
      display: flex;
      flex-direction: column;
    }

    :deep(.el-table) {
        border-radius: var(--tm-border-radius-sm);
        border: 1px solid var(--tm-border-color);
        
        .el-input__inner {
          padding: 0 4px;
        }
      }
    }
    
  .pagination-area {
    padding: 12px 20px;
    background-color: var(--tm-bg-card);
    border-top: 1px solid var(--tm-border-color);
    display: flex;
    justify-content: space-between;
    align-items: center;

    .table-actions-left {
      display: flex;
      gap: 12px;
      align-items: center;

      .inline-upload {
        display: flex;
        align-items: center;
      }
    }
  }
  }
}
</style>