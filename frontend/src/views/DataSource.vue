<template>
  <div class="datasource-container">
    <!-- 顶部操作区 -->
    <div class="toolbar">
      <el-button type="primary" @click="handleImportClick" :loading="isImporting">
        <el-icon><Upload /></el-icon> 导入 CSV / Excel
      </el-button>
      <el-button @click="handleExportClick">
        <el-icon><Download /></el-icon> 导出当前数据
      </el-button>
      <span class="tip-text" v-if="totalRecords > 0">当前库中共有 {{ totalRecords }} 条记录</span>
    </div>

    <!-- 数据预览区 (空状态与加载状态) -->
    <div class="table-container" v-loading="isLoading">
      <el-empty 
        v-if="!isLoading && tableData.length === 0" 
        description="暂无数据，请先导入数据源" 
        :image-size="160"
      >
        <el-button type="primary" @click="handleImportClick">立即导入</el-button>
      </el-empty>

      <!-- 动态列数据表格 -->
      <el-table 
        v-else 
        :data="tableData" 
        style="width: 100%" 
        height="calc(100vh - 280px)"
        border
        stripe
      >
        <!-- 固定显示内部ID -->
        <el-table-column prop="id" label="系统ID" width="100" fixed />
        
        <!-- 动态渲染数据列 -->
        <el-table-column 
          v-for="col in dynamicColumns" 
          :key="col" 
          :prop="col" 
          :label="col" 
          min-width="150"
          show-overflow-tooltip
        />
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper" v-if="totalRecords > 0">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[50, 100, 200, 500]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="totalRecords"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Upload, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 引入 Wails 生成的 TS Bindings
import { ImportData, GetRawDataList, ExportData } from '../../wailsjs/go/main/App'

const isLoading = ref(false)
const isImporting = ref(false)

const tableData = ref<any[]>([])
const dynamicColumns = ref<string[]>([])

const currentPage = ref(1)
const pageSize = ref(50)
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
.datasource-container {
  display: flex;
  flex-direction: column;
  height: 100%;

  .toolbar {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;

    .tip-text {
      color: var(--tm-text-secondary);
      font-size: 13px;
      margin-left: auto;
    }
  }

  .table-container {
    flex: 1;
    background-color: var(--tm-bg-main);
    border-radius: var(--tm-border-radius);
    
    // 覆盖 element table 的边框颜色
    :deep(.el-table) {
      --el-table-border-color: var(--tm-border-color);
      border-radius: var(--tm-border-radius-sm);
      overflow: hidden;
      
      th.el-table__cell {
        background-color: var(--tm-bg-sidebar);
        color: var(--tm-text-primary);
        font-weight: 600;
      }
    }
  }

  .pagination-wrapper {
    margin-top: 20px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
