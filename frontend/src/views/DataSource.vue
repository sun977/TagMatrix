<template>
  <div class="page-container">
    <!-- 数据集列表视图 -->
    <div v-if="viewMode === 'list'">
      <header class="page-header">
        <div class="header-left">
          <h1 class="page-title">
            数据集管理
          </h1>
          <p class="page-subtitle">管理您的数据集，支持创建、导入数据、预览和导出。</p>
        </div>
      </header>

      <div class="toolbar">
        <div class="toolbar-left">
          <el-button type="primary" @click="handleCreateDataset" class="action-btn-green">
            <el-icon><Plus /></el-icon> 新建数据集
          </el-button>
          <el-button type="success" @click="handleImportClick()" :loading="isImporting">
            <el-icon><Upload /></el-icon> 导入数据
          </el-button>
          <el-button type="info" @click="handleImportBusinessAsset">
            <el-icon><Upload /></el-icon> 导入数据集结构和规则
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-button class="filter-btn" @click="fetchDatasetList" circle>
            <el-icon><RefreshRight /></el-icon>
          </el-button>
        </div>
      </div>

      <div class="table-section" v-loading="isDatasetLoading">
        <el-table :data="datasetList" style="width: 100%" class="custom-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="数据集名称" min-width="150" show-overflow-tooltip />
          <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
          <el-table-column label="包含字段" min-width="200">
            <template #default="scope">
              <div class="tags-container" v-if="scope.row.schema_keys">
                <span class="mock-tag tag-gray" v-for="col in parseSchema(scope.row.schema_keys).slice(0, 3)" :key="col">{{ col }}</span>
                <span class="mock-tag tag-gray" v-if="parseSchema(scope.row.schema_keys).length > 3">...</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" align="right">
            <template #header>
              <div style="display: flex; align-items: center; justify-content: flex-end;">
                操作
                <el-tooltip effect="dark" placement="top-end">
                  <template #content>
                    <div style="line-height: 1.8;">
                      <div><strong>数据</strong>：浏览和管理该数据集内的数据记录。</div>
                      <div><strong>编辑</strong>：修改数据集的名称和描述信息。</div>
                      <div><strong>导出</strong>：导出该数据集的结构和绑定规则。</div>
                      <div><strong style="color: #F56C6C;">删除</strong>：彻底删除该数据集数据和绑定规则。</div>
                    </div>
                  </template>
                  <el-icon style="font-size: 14px; margin-left: 4px; color: #909399; cursor: help;"><QuestionFilled /></el-icon>
                </el-tooltip>
              </div>
            </template>
            <template #default="scope">
              <div style="margin-bottom: 6px;">
                <el-button size="small" class="action-btn" @click="handleViewDataset(scope.row)">数据</el-button>
                <el-button size="small" class="action-btn" @click="handleEditDataset(scope.row)">编辑</el-button>
              </div>
              <div style="display: flex; align-items: center; justify-content: flex-end;">
                <el-button size="small" class="action-btn" @click="handleExportBusinessAsset(scope.row)">导出</el-button>
                <el-button size="small" class="action-btn" @click="handleDeleteDataset(scope.row)" style="color: #F56C6C; border-color: var(--tm-border-color); background-color: var(--el-button-bg-color);">删除</el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 数据详情视图 -->
    <div v-else-if="viewMode === 'detail'">
      <header class="page-header">
        <div class="header-left" style="display: flex; align-items: center; gap: 16px;">
          <el-button @click="handleBackToList" circle>
            <el-icon><Back /></el-icon>
          </el-button>
          <div>
            <h1 class="page-title">
              {{ currentDataset?.name }}
            </h1>
            <p class="page-subtitle">{{ currentDataset?.description }}</p>
          </div>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <el-button type="success" @click="handleImportClick(currentDataset?.id)" :loading="isImporting">
            <el-icon><Upload /></el-icon> 导入数据
          </el-button>
          <el-button @click="handleExportClick">
            <el-icon><Download /></el-icon> 导出数据
          </el-button>
          <el-button @click="handleDeleteClick" type="danger" plain :disabled="selectedRows.length === 0">
            <el-icon><Delete /></el-icon> 删除选中
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-select v-model="searchCol" placeholder="全部字段" clearable style="width: 140px">
            <el-option label="全部字段" value="" />
            <el-option v-for="col in dynamicColumns" :key="col" :label="col" :value="col" />
          </el-select>
          <el-input
            v-model="searchQuery"
            placeholder="搜索数据内容"
            class="search-input"
            :prefix-icon="Search"
            clearable
            @keyup.enter="fetchTableData"
          />
          <el-button class="filter-btn" @click="fetchTableData">
            <el-icon><Filter /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 数据预览区 -->
      <div class="table-section" v-loading="isLoading">
        <div class="table-header">
          <div class="table-title">
            <h4>数据列表</h4>
            <span class="count-pill">共 {{ totalRecords }} 条数据</span>
          </div>
          <div class="table-actions">
            <el-button circle @click="fetchTableData">
              <el-icon><RefreshRight /></el-icon>
            </el-button>
            
            <el-popover
              placement="bottom-end"
              title="展示列设置"
              :width="200"
              trigger="click"
            >
              <template #reference>
                <el-button circle>
                  <el-icon><Setting /></el-icon>
                </el-button>
              </template>
              <div class="column-settings">
                <el-checkbox 
                  v-for="col in dynamicColumns" 
                  :key="col" 
                  :model-value="!hiddenColumns.includes(col)"
                  @change="toggleColumn(col)"
                >
                  {{ col }}
                </el-checkbox>
              </div>
            </el-popover>
          </div>
        </div>

        <el-empty 
          v-if="!isLoading && tableData.length === 0" 
          description="暂无数据" 
          :image-size="120"
        />

        <el-table 
          v-else 
          :data="tableData" 
          style="width: 100%" 
          class="custom-table"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="55" />
          
          <el-table-column prop="id" label="内部ID" width="100" />
          
          <el-table-column 
            v-for="col in visibleColumns" 
            :key="col" 
            :prop="col" 
            :label="col" 
            min-width="120"
            show-overflow-tooltip
          />

          <el-table-column label="操作" width="100" fixed="right" align="center">
            <template #default="scope">
              <el-button type="primary" link size="small" class="detail-btn" @click="handleViewDetail(scope.row)">查看详情</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="pagination-wrapper" v-if="totalRecords > 0">
          <span class="pagination-info">显示 {{ (currentPage - 1) * pageSize + 1 }} 到 {{ Math.min(currentPage * pageSize, totalRecords) }} 条，共 {{ totalRecords }} 条记录</span>
          <el-pagination
            :current-page="currentPage"
            :page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            layout="prev, pager, next, jumper"
            :total="totalRecords"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
            background
          />
        </div>
      </div>
    </div>

    <!-- 查看详情对话框 -->
    <el-dialog v-model="detailDialogVisible" title="数据详情 (JSON)" width="600px">
      <div class="detail-content-wrapper">
        <pre class="json-preview">{{ formattedDetailJson }}</pre>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="detailDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="copyDetailJson" :icon="DocumentCopy">
            复制 JSON
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Sheet 选择与导入对话框 -->
    <el-dialog v-model="importDialogVisible" title="导入数据" width="500px">
      <el-form label-position="top">
        <el-form-item label="数据文件" required>
          <div style="display: flex; gap: 10px; width: 100%;">
            <el-input v-model="pendingImportFileName" disabled placeholder="未选择文件" />
            <el-button type="primary" @click="handleSelectFile">选择文件</el-button>
          </div>
        </el-form-item>

        <el-form-item label="选择要导入的工作表 (Sheet)" v-if="availableSheets.length > 0">
          <el-checkbox-group v-model="selectedSheets" class="sheet-checkbox-group">
            <el-checkbox v-for="sheet in availableSheets" :key="sheet" :value="sheet" :label="sheet" class="sheet-checkbox">
              {{ sheet }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-form-item label="目标数据集" required>
          <el-radio-group v-model="importTargetMode">
            <el-radio value="existing" :disabled="!hasExistingDatasets">现有数据集</el-radio>
            <el-radio value="new">新建数据集</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="importTargetMode === 'existing'" label="选择数据集">
          <el-select v-model="importDatasetId" placeholder="请选择目标数据集" style="width: 100%;">
            <el-option v-for="ds in datasetList" :key="ds.id" :label="ds.name" :value="ds.id" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="importTargetMode === 'new'" label="数据集名称" required>
          <el-input v-model="importNewDatasetName" placeholder="请输入新建数据集名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="importDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmImport" :loading="isImporting" :disabled="!pendingImportFilePath">
            确定导入
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 新建/编辑数据集对话框 -->
    <el-dialog v-model="datasetDialogVisible" :title="editingDatasetId ? '编辑数据集' : '新建数据集'" width="450px">
      <el-form :model="datasetForm" label-position="top">
        <el-form-item label="数据集名称" required>
          <el-input v-model="datasetForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="datasetForm.description" type="textarea" :rows="3" placeholder="请输入描述信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="datasetDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveDataset" :loading="isSavingDataset">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Upload, Download, Delete, Search, Filter, RefreshRight, Setting, DocumentCopy, Back, Plus, MoreFilled, QuestionFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 引入 Wails 生成的 TS Bindings
import { 
  AnalyzeDataFile, ImportData, GetRawDataList, ExportData, DeleteRawData, 
  ListDatasets, CreateDataset, UpdateDataset, DeleteDataset,
  ExportDatasetWithRules, ImportDatasetWithRules
} from '../../wailsjs/go/main/App'

const viewMode = ref<'list' | 'detail'>('list')
const isDatasetLoading = ref(false)
const datasetList = ref<any[]>([])
const currentDataset = ref<any>(null)

const isLoading = ref(false)
const isImporting = ref(false)

const searchQuery = ref('')
const searchCol = ref('')

const tableData = ref<any[]>([])
const dynamicColumns = ref<string[]>([])
const selectedRows = ref<any[]>([])
const hiddenColumns = ref<string[]>([])

const currentPage = ref(1)
const pageSize = ref(10)
const totalRecords = ref(0)

// Dataset CRUD state
const datasetDialogVisible = ref(false)
const isSavingDataset = ref(false)
const editingDatasetId = ref<number | null>(null)
const datasetForm = ref({ name: '', description: '' })

// Import Dialog state
const importDialogVisible = ref(false)
const pendingImportFilePath = ref('')
const pendingImportFileName = ref('')
const availableSheets = ref<string[]>([])
const selectedSheets = ref<string[]>([])
const importTargetMode = ref<'existing' | 'new'>('new')
const importDatasetId = ref<number | null>(null)
const importNewDatasetName = ref('')

// 查看详情相关的状态
const detailDialogVisible = ref(false)
const formattedDetailJson = ref('')

const hasExistingDatasets = computed(() => datasetList.value.length > 0)

// 计算需要展示的列
const visibleColumns = computed(() => {
  return dynamicColumns.value.filter(col => !hiddenColumns.value.includes(col))
})

const parseSchema = (schemaStr: string) => {
  try {
    return JSON.parse(schemaStr) || []
  } catch (e) {
    return []
  }
}

const toggleColumn = (col: string) => {
  const idx = hiddenColumns.value.indexOf(col)
  if (idx > -1) {
    hiddenColumns.value.splice(idx, 1)
  } else {
    hiddenColumns.value.push(col)
  }
}

// 添加导出数据集结构和规则功能
const handleExportBusinessAsset = async (dataset: any) => {
  try {
    const result = await ExportDatasetWithRules(dataset.id, "")
    if (result !== undefined && result !== null) {
      ElMessage.success(`数据集结构和规则导出成功 (${dataset.name})`)
    }
  } catch (error: any) {
    if (error !== "cancelled") ElMessage.error('数据集结构和规则导出失败: ' + String(error))
  }
}

// 添加导入数据集结构和规则功能
const handleImportBusinessAsset = async () => {
  try {
    const result = await ImportDatasetWithRules("")
    if (result) {
      ElMessage.success(`数据集结构和规则导入成功: ${result.dataset_name}, 导入规则${result.rule_imported}个, 跳过规则${result.rule_skipped}个`)
      fetchDatasetList() // 刷新数据集列表
    }
  } catch (error: any) {
    if (error !== "cancelled") ElMessage.error('数据集结构和规则导入失败: ' + String(error))
  }
}

// ----------------- 数据集列表相关方法 -----------------
const fetchDatasetList = async () => {
  isDatasetLoading.value = true
  try {
    const list = await ListDatasets()
    datasetList.value = list || []
  } catch (error) {
    ElMessage.error('获取数据集失败: ' + String(error))
  } finally {
    isDatasetLoading.value = false
  }
}

const handleCreateDataset = () => {
  editingDatasetId.value = null
  datasetForm.value = { name: '', description: '' }
  datasetDialogVisible.value = true
}

const handleEditDataset = (row: any) => {
  editingDatasetId.value = row.id
  datasetForm.value = { name: row.name, description: row.description }
  datasetDialogVisible.value = true
}

const saveDataset = async () => {
  if (!datasetForm.value.name.trim()) {
    ElMessage.warning('请输入数据集名称')
    return
  }
  isSavingDataset.value = true
  try {
    if (editingDatasetId.value) {
      await UpdateDataset(editingDatasetId.value, datasetForm.value.name, datasetForm.value.description)
      ElMessage.success('更新成功')
    } else {
      await CreateDataset(datasetForm.value.name, datasetForm.value.description)
      ElMessage.success('创建成功')
    }
    datasetDialogVisible.value = false
    fetchDatasetList()
  } catch (error) {
    ElMessage.error('保存失败: ' + String(error))
  } finally {
    isSavingDataset.value = false
  }
}

const handleDeleteDataset = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要删除数据集 [${row.name}] 及其所有数据吗？此操作不可恢复。`, '危险操作', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await DeleteDataset(row.id)
    ElMessage.success('删除成功')
    fetchDatasetList()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败: ' + String(e))
    }
  }
}

const handleViewDataset = (row: any) => {
  currentDataset.value = row
  viewMode.value = 'detail'
  currentPage.value = 1
  searchQuery.value = ''
  searchCol.value = ''
  fetchTableData()
}

const handleBackToList = () => {
  viewMode.value = 'list'
  currentDataset.value = null
  fetchDatasetList()
}

// ----------------- 数据导入相关方法 -----------------
const handleImportClick = (prefillDatasetId?: number) => {
  pendingImportFilePath.value = ''
  pendingImportFileName.value = ''
  availableSheets.value = []
  selectedSheets.value = []
  importNewDatasetName.value = ''
  
  if (prefillDatasetId) {
    importTargetMode.value = 'existing'
    importDatasetId.value = prefillDatasetId
  } else {
    importTargetMode.value = datasetList.value.length > 0 ? 'existing' : 'new'
    importDatasetId.value = datasetList.value.length > 0 ? datasetList.value[0].id : null
  }
  
  importDialogVisible.value = true
}

const handleSelectFile = async () => {
  try {
    const analysis = await AnalyzeDataFile()
    if (!analysis) return

    pendingImportFilePath.value = analysis.filePath
    pendingImportFileName.value = analysis.fileName

    if (analysis.fileType === 'excel' && analysis.sheetNames && analysis.sheetNames.length > 0) {
      availableSheets.value = analysis.sheetNames
      selectedSheets.value = [analysis.sheetNames[0]]
    } else {
      availableSheets.value = []
      selectedSheets.value = []
    }
    
    if (importTargetMode.value === 'new' && !importNewDatasetName.value) {
      importNewDatasetName.value = analysis.fileName.split('.').slice(0, -1).join('.') || analysis.fileName
    }
  } catch (error: any) {
    if (error !== "cancelled") {
      ElMessage.error('文件分析失败: ' + String(error))
    }
  }
}

const confirmImport = async () => {
  if (!pendingImportFilePath.value) {
    ElMessage.warning('请先选择数据文件')
    return
  }
  if (availableSheets.value.length > 0 && selectedSheets.value.length === 0) {
    ElMessage.warning('请至少选择一个 Sheet')
    return
  }
  
  let targetDatasetId = 0
  let targetNewName = ""
  
  if (importTargetMode.value === 'existing') {
    if (!importDatasetId.value) {
      ElMessage.warning('请选择目标数据集')
      return
    }
    targetDatasetId = importDatasetId.value
  } else {
    if (!importNewDatasetName.value.trim()) {
      ElMessage.warning('请输入新建数据集名称')
      return
    }
    targetNewName = importNewDatasetName.value.trim()
  }

  isImporting.value = true
  try {
    const count = await ImportData(pendingImportFilePath.value, selectedSheets.value, targetDatasetId, targetNewName)
    if (count > 0) {
      ElMessage.success(`成功导入 ${count} 条数据`)
      importDialogVisible.value = false
      
      if (viewMode.value === 'detail') {
        fetchTableData()
      } else {
        fetchDatasetList()
      }
    }
  } catch (error: any) {
    ElMessage.error('导入失败: ' + String(error))
  } finally {
    isImporting.value = false
  }
}

// ----------------- 数据列表相关方法 -----------------
const fetchTableData = async () => {
  if (!currentDataset.value) return
  isLoading.value = true
  try {
    const res = await GetRawDataList(currentDataset.value.id, currentPage.value, pageSize.value, searchCol.value, searchQuery.value)
    
    const parsedData = res.Records.map((r: any) => {
      let dataObj = {}
      try {
        dataObj = JSON.parse(r.data)
      } catch (e) {}
      return { id: r.id, batch_id: r.batch_id, ...dataObj }
    })

    if (parsedData.length > 0) {
      const colSet = new Set<string>()
      parsedData.forEach((row: any) => {
        Object.keys(row).forEach(k => {
          if (k !== 'id' && k !== 'batch_id') colSet.add(k)
        })
      })
      let cols = Array.from(colSet)
      if (cols.includes('来源文件')) {
        cols = cols.filter(c => c !== '来源文件')
        cols.push('来源文件')
      }
      dynamicColumns.value = cols
    } else {
      dynamicColumns.value = []
    }
    
    tableData.value = parsedData
    totalRecords.value = res.Total
  } catch (error) {
    console.error(error)
    ElMessage.error('获取数据失败: ' + String(error))
  } finally {
    isLoading.value = false
  }
}

const handleDeleteClick = async () => {
  if (selectedRows.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${selectedRows.value.length} 条数据吗？此操作不可恢复。`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    const ids = selectedRows.value.map(row => row.id)
    await DeleteRawData(ids)
    ElMessage.success('删除成功')
    
    if (tableData.value.length === ids.length && currentPage.value > 1) {
      currentPage.value -= 1
    }
    fetchTableData()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('删除失败: ' + String(e))
  }
}

const handleExportClick = async () => {
  if (!currentDataset.value) return
  try {
    await ExportData(currentDataset.value.id, "") 
    ElMessage.success('导出成功')
  } catch (error: any) {
    if (error !== "cancelled") ElMessage.error('导出失败: ' + String(error))
  }
}

const handleSelectionChange = (val: any[]) => selectedRows.value = val
const handleSizeChange = (val: number) => { pageSize.value = val; fetchTableData() }
const handleCurrentChange = (val: number) => { currentPage.value = val; fetchTableData() }

const handleViewDetail = (row: any) => {
  const { id, batch_id, ...rest } = row
  formattedDetailJson.value = JSON.stringify(rest, null, 2)
  detailDialogVisible.value = true
}

const copyDetailJson = async () => {
  try {
    await navigator.clipboard.writeText(formattedDetailJson.value)
    ElMessage.success('JSON 数据已复制到剪贴板')
  } catch (err) {
    ElMessage.error('复制失败，您的浏览器可能不支持该功能')
  }
}

onMounted(() => {
  fetchDatasetList()
})
</script>

<style scoped lang="scss">
.page-container {
  padding: 24px 32px 40px;
}

/* --- 页面顶部 --- */
.page-header {
  margin-bottom: 24px;

  .page-title {
    font-size: 20px;
    font-weight: 600;
    color: var(--tm-text-primary);
    margin: 0 0 8px 0;
  }

  .page-subtitle {
    font-size: 14px;
    color: var(--tm-text-secondary);
    margin: 0;
  }
}

/* --- 工具栏 --- */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;

  .toolbar-left {
    display: flex;
    gap: 12px;

    .el-button {
      border-radius: var(--tm-border-radius-sm);
      font-weight: 500;
    }

    .action-btn-green {
      background-color: var(--tm-accent-primary);
      border-color: var(--tm-accent-primary);
      &:hover {
        background-color: var(--tm-accent-hover);
        border-color: var(--tm-accent-hover);
      }
    }
  }

  .toolbar-right {
    display: flex;
    gap: 12px;

    .search-input {
      width: 240px;
      :deep(.el-input__wrapper) {
        border-radius: var(--tm-border-radius-sm);
      }
    }

    .filter-btn {
      border-radius: var(--tm-border-radius-sm);
      padding: 8px 12px;
    }
  }
}

/* --- 数据预览区 --- */
.table-section {
  background-color: var(--tm-bg-main);
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius);
  padding: 20px 24px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;

  .table-title {
    display: flex;
    align-items: center;
    gap: 12px;

    h4 {
      margin: 0;
      font-size: 15px;
      font-weight: 600;
      color: var(--tm-text-primary);
    }

    .count-pill {
      background-color: var(--tm-bg-hover);
      color: var(--tm-text-secondary);
      font-size: 12px;
      padding: 2px 10px;
      border-radius: 12px;
    }
  }

  .table-actions {
    display: flex;
    gap: 8px;
    
    .el-button {
      border-color: var(--tm-border-color);
      color: var(--tm-text-secondary);
      &:hover {
        color: var(--tm-text-primary);
        border-color: #dcdfe6;
      }
    }
  }
}

/* --- 表格样式 --- */
.custom-table {
  --el-table-border-color: transparent;
  --el-table-header-bg-color: var(--tm-bg-main);
  --el-table-header-text-color: var(--tm-text-secondary);
  
  :deep(th.el-table__cell) {
    font-weight: 500;
    padding: 12px 0;
    border-bottom: 1px solid var(--tm-border-color);
  }
  
  :deep(td.el-table__cell) {
    padding: 16px 0;
    font-size: 14px;
    color: var(--tm-text-regular);
    border-bottom: 1px solid var(--tm-border-color);
  }
}

/* --- 标签样式 --- */
.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.mock-tag {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 12px;
  font-weight: 500;

  &.tag-yellow {
    background-color: rgba(230, 162, 60, 0.1);
    color: #e6a23c;
  }
  &.tag-blue {
    background-color: rgba(58, 142, 230, 0.1);
    color: #3a8ee6;
  }
  &.tag-gray {
    background-color: var(--tm-bg-hover);
    color: #909399;
  }
  &.tag-red {
    background-color: rgba(245, 108, 108, 0.1);
    color: #f56c6c;
  }
}

.detail-btn {
  color: var(--tm-text-secondary);
  font-weight: 500;
  background-color: var(--tm-bg-hover);
  padding: 6px 12px;
  border-radius: 6px;

  &:hover {
    color: var(--tm-text-primary);
    background-color: var(--tm-bg-active);
  }
}

.column-settings {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
}

/* --- 分页 --- */
.pagination-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;

  .pagination-info {
    font-size: 13px;
    color: var(--tm-text-secondary);
  }

  :deep(.el-pagination.is-background .el-pager li:not(.is-disabled).is-active) {
    background-color: var(--tm-accent-primary);
  }
}

.detail-content-wrapper {
  max-height: 50vh;
  overflow-y: auto;
  background-color: var(--tm-bg-subtle);
  border-radius: var(--tm-border-radius-sm);
  padding: 16px;
  border: 1px solid var(--tm-border-color);
}

.json-preview {
  margin: 0;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 14px;
  color: var(--tm-text-primary);
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>
