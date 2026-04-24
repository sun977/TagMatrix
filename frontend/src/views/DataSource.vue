<template>
  <div class="page-container">
    <!-- 页面顶部 Header -->
    <header class="page-header">
      <div class="header-left">
        <h1 class="page-title">数据源管理</h1>
        <p class="page-subtitle">在这里管理所有待打标和已打标的数据源，支持上传、预览、导出数据。</p>
      </div>
    </header>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-button type="primary" @click="handleImportClick" :loading="isImporting" class="action-btn-green">
          <el-icon><Upload /></el-icon> 上传数据
        </el-button>
        <el-button @click="handleExportClick">
          <el-icon><Download /></el-icon> 导出当前结果
        </el-button>
        <el-button>
          <el-icon><Delete /></el-icon> 删除选中
        </el-button>
      </div>
      <div class="toolbar-right">
        <el-input
          v-model="searchQuery"
          placeholder="搜索数据内容"
          class="search-input"
          :prefix-icon="Search"
        />
        <el-button class="filter-btn">
          <el-icon><Filter /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- 上传区域 -->
    <div class="upload-area" @click="handleImportClick">
      <div class="upload-content">
        <div class="upload-icon-wrapper">
          <el-icon><UploadFilled /></el-icon>
        </div>
        <h3 class="upload-title">拖拽文件到此处上传，或点击选择文件</h3>
        <p class="upload-desc">支持 .xlsx, .xls, .csv 格式，单文件最大支持 100MB</p>
        <el-button type="primary" class="action-btn-green select-file-btn">选择文件</el-button>
      </div>
    </div>

    <!-- 数据预览区 -->
    <div class="table-section" v-loading="isLoading">
      <div class="table-header">
        <div class="table-title">
          <h4>用户行为数据_202404.xlsx</h4>
          <span class="count-pill">共 {{ totalRecords }} 条数据</span>
        </div>
        <div class="table-actions">
          <el-button circle @click="fetchTableData">
            <el-icon><RefreshRight /></el-icon>
          </el-button>
          <el-button circle>
            <el-icon><Setting /></el-icon>
          </el-button>
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
        
        <!-- 固定显示内部ID -->
        <el-table-column prop="id" label="用户ID" width="120" />
        
        <!-- 动态渲染数据列 -->
        <el-table-column 
          v-for="col in dynamicColumns" 
          :key="col" 
          :prop="col" 
          :label="col" 
          min-width="120"
          show-overflow-tooltip
        />

        <!-- Mock 的已打标签列 -->
        <el-table-column label="已打标签" min-width="200">
          <template #default="scope">
            <div class="tags-container">
              <!-- 为了 UI 演示，随机分配一些 mock 标签 -->
              <span class="mock-tag tag-yellow" v-if="scope.$index % 3 === 0">高价值用户</span>
              <span class="mock-tag tag-blue" v-if="scope.$index % 2 === 0">活跃用户</span>
              <span class="mock-tag tag-gray" v-if="scope.$index % 3 === 1">普通用户</span>
              <span class="mock-tag tag-red" v-if="scope.$index % 4 === 3">流失风险</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="scope">
            <el-button type="primary" link size="small" class="detail-btn">查看详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="totalRecords > 0">
        <span class="pagination-info">显示 {{ (currentPage - 1) * pageSize + 1 }} 到 {{ Math.min(currentPage * pageSize, totalRecords) }} 条，共 {{ totalRecords }} 条记录</span>
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
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
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Upload, Download, Delete, Search, Filter, UploadFilled, RefreshRight, Setting } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 引入 Wails 生成的 TS Bindings
import { ImportData, GetRawDataList, ExportData } from '../../wailsjs/go/main/App'

const isLoading = ref(false)
const isImporting = ref(false)
const searchQuery = ref('')

const tableData = ref<any[]>([])
const dynamicColumns = ref<string[]>([])
const selectedRows = ref<any[]>([])

const currentPage = ref(1)
const pageSize = ref(10)
const totalRecords = ref(0)

// 获取真实数据
const fetchTableData = async () => {
  isLoading.value = true
  try {
    const res = await GetRawDataList(currentPage.value, pageSize.value)
    
    // 解析 JSON 数据并展平
    const parsedData = res.Records.map((r: any) => {
      let dataObj = {}
      try {
        dataObj = JSON.parse(r.data)
      } catch (e) {
        console.warn('Failed to parse record data:', r.data)
      }
      return {
        id: r.id,
        batch_id: r.batch_id,
        ...dataObj
      }
    })

    // 提取动态列头
    if (parsedData.length > 0) {
      const cols = Object.keys(parsedData[0]).filter(k => k !== 'id' && k !== 'batch_id')
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

const handleImportClick = async () => {
  isImporting.value = true
  try {
    const count = await ImportData("") // 传入空字符串，Wails 后端会自动弹窗选择文件
    if (count > 0) {
      ElMessage.success(`成功导入 ${count} 条数据`)
      currentPage.value = 1
      fetchTableData()
    }
  } catch (error: any) {
    // 用户取消选择文件不报错
    if (error !== "cancelled") {
      ElMessage.error('导入失败: ' + String(error))
    }
  } finally {
    isImporting.value = false
  }
}

const handleExportClick = async () => {
  try {
    await ExportData(0, "") // 传入空字符串让后端弹窗，0 表示导出全部
    ElMessage.success('导出成功')
  } catch (error: any) {
    if (error !== "cancelled") {
      ElMessage.error('导出失败: ' + String(error))
    }
  }
}

const handleSelectionChange = (val: any[]) => {
  selectedRows.value = val
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  fetchTableData()
}

const handleCurrentChange = (val: number) => {
  currentPage.value = val
  fetchTableData()
}

onMounted(() => {
  fetchTableData()
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

/* --- 上传区域 --- */
.upload-area {
  border: 1px dashed #d9d9d9;
  border-radius: var(--tm-border-radius);
  background-color: #fafafa;
  padding: 40px 0;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  margin-bottom: 32px;

  &:hover {
    border-color: var(--tm-accent-primary);
    background-color: var(--tm-accent-light);
  }

  .upload-content {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .upload-icon-wrapper {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background-color: #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 16px;
    color: var(--tm-text-secondary);
    font-size: 24px;
  }

  .upload-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--tm-text-primary);
    margin: 0 0 8px 0;
  }

  .upload-desc {
    font-size: 13px;
    color: var(--tm-text-secondary);
    margin: 0 0 20px 0;
  }

  .select-file-btn {
    background-color: var(--tm-accent-primary);
    border-color: var(--tm-accent-primary);
    border-radius: var(--tm-border-radius-sm);
    padding: 8px 24px;
    font-weight: 500;
    
    &:hover {
      background-color: var(--tm-accent-hover);
      border-color: var(--tm-accent-hover);
    }
  }
}

/* --- 数据预览区 --- */
.table-section {
  background-color: #ffffff;
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
      background-color: #f5f5f5;
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
  --el-table-header-bg-color: #ffffff;
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
    border-bottom: 1px solid #fafafa;
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
    background-color: #fdf5e6;
    color: #e6a23c;
  }
  &.tag-blue {
    background-color: #e6f0fa;
    color: #3a8ee6;
  }
  &.tag-gray {
    background-color: #f4f4f5;
    color: #909399;
  }
  &.tag-red {
    background-color: #fef0f0;
    color: #f56c6c;
  }
}

.detail-btn {
  color: var(--tm-text-secondary);
  font-weight: 500;
  background-color: #f5f5f5;
  padding: 6px 12px;
  border-radius: 6px;

  &:hover {
    color: var(--tm-text-primary);
    background-color: #e5e5e5;
  }
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
</style>