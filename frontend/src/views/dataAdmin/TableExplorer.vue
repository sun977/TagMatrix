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
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[50, 100, 200]"
          layout="total, sizes, prev, pager, next"
          :total="totalRecords"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>

    <!-- 新增行对话框 -->
    <el-dialog v-model="addRowDialogVisible" :title="tabMode === 'system' ? '新增物理表记录' : '新增业务数据'" width="500px">
      <el-form :model="newRowForm" label-width="120px">
        <template v-for="col in columns" :key="col">
          <el-form-item :label="col" v-if="col !== 'id'">
            <el-input v-model="newRowForm[col]" placeholder="请输入内容" />
          </el-form-item>
        </template>
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
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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

const handleAddRow = () => {
  newRowForm.value = {}
  for (const col of columns.value) {
    if (col !== 'id') {
      newRowForm.value[col] = ''
    }
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
      justify-content: flex-end;
    }
  }
}
</style>