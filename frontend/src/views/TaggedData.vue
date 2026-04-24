<template>
  <div class="tagged-data-container">
    <!-- 页面标题与操作区 -->
    <div class="page-header">
      <h2>打标数据看板</h2>
      <div class="header-actions">
        <el-button type="primary" :icon="Download" class="mint-btn" @click="handleExport">导出当前数据</el-button>
      </div>
    </div>

    <!-- 筛选过滤栏 (卡片) -->
    <div class="filter-card card-panel">
      <el-form :inline="true" :model="filterForm" class="filter-form" @submit.prevent>
        <el-form-item label="关键字">
          <el-input 
            v-model="filterForm.keyword" 
            placeholder="搜索文本内容" 
            :prefix-icon="Search"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="标签分类">
          <el-select v-model="filterForm.tag" placeholder="全部标签" clearable class="w-150">
            <el-option v-for="tag in tagOptions" :key="tag.id" :label="tag.name" :value="String(tag.id)" />
          </el-select>
        </el-form-item>

        <el-form-item label="打标批次">
          <el-select v-model="filterForm.batch" placeholder="全部批次" clearable class="w-150">
            <el-option v-for="batch in batchOptions" :key="batch.id" :label="batch.name" :value="String(batch.id)" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" class="mint-btn" :icon="Search" @click="handleSearch">查询</el-button>
          <el-button :icon="RefreshRight" @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 数据表格区 (卡片) -->
    <div class="table-card card-panel">
      <el-table 
        :data="tableData" 
        style="width: 100%" 
        v-loading="loading"
        class="custom-table"
      >
        <el-table-column prop="id" label="数据 ID" width="100" />
        <el-table-column prop="content" label="原始数据内容" min-width="300" show-overflow-tooltip />
        <el-table-column label="命中标签" min-width="200">
          <template #default="scope">
            <div class="tags-wrapper">
              <el-tag 
                v-for="(tag, index) in scope.row.tags" 
                :key="index"
                :color="tag.color + '20'"
                :style="{ color: tag.color, borderColor: tag.color + '40' }"
                size="small"
                class="custom-tag"
                disable-transitions
              >
                {{ tag.name }}
              </el-tag>
              <span v-if="!scope.row.tags || scope.row.tags.length === 0" class="no-tag">-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="batchName" label="来源批次" min-width="180" show-overflow-tooltip />
        <el-table-column prop="updateTime" label="打标时间" width="160" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'success' ? 'success' : 'info'" size="small">
              {{ scope.row.status === 'success' ? '已打标' : '未命中' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页器 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="totalItems"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, Download, RefreshRight } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { GetTaggedDataList, ExportData, GetAllTags, GetTaskBatches } from '../../wailsjs/go/main/App'

const loading = ref(false)

// 过滤表单状态
const filterForm = reactive({
  keyword: '',
  tag: '',
  batch: ''
})

// 下拉选项数据
const tagOptions = ref<any[]>([])
const batchOptions = ref<any[]>([])

// 分页状态
const currentPage = ref(1)
const pageSize = ref(20)
const totalItems = ref(0) 

// 表格数据
const tableData = ref<any[]>([])

const handleSearch = async () => {
  loading.value = true
  try {
    const res = await GetTaggedDataList(filterForm.keyword, filterForm.tag, filterForm.batch, currentPage.value, pageSize.value)
    if (res) {
      tableData.value = res.records || []
      totalItems.value = res.total || 0
    }
  } catch (e) {
    ElMessage.error('查询失败: ' + String(e))
  } finally {
    loading.value = false
  }
}

const resetFilter = () => {
  filterForm.keyword = ''
  filterForm.tag = ''
  filterForm.batch = ''
  currentPage.value = 1
  handleSearch()
}

const handleExport = async () => {
  try {
    const batchId = filterForm.batch ? parseInt(filterForm.batch) : 0
    await ExportData(batchId, '')
    ElMessage.success('导出成功')
  } catch (e: any) {
    if (e !== 'cancelled') {
      ElMessage.error('导出失败: ' + String(e))
    }
  }
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  handleSearch()
}

const handleCurrentChange = (val: number) => {
  currentPage.value = val
  handleSearch()
}

const loadOptions = async () => {
  try {
    const tags = await GetAllTags()
    if (tags) tagOptions.value = tags

    const batches = await GetTaskBatches()
    if (batches) batchOptions.value = batches
  } catch (e) {
    console.error('加载选项失败', e)
  }
}

onMounted(() => {
  loadOptions()
  handleSearch()
})
</script>

<style scoped lang="scss">
.tagged-data-container {
  padding: 24px;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;

  h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 600;
    color: var(--tm-text-primary);
  }
}

.card-panel {
  background-color: var(--tm-bg-main);
  border-radius: var(--tm-border-radius, 12px);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 20px;
}

.filter-card {
  flex-shrink: 0;
  
  .filter-form {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    
    .el-form-item {
      margin-bottom: 0;
      margin-right: 0;
    }
  }

  .w-150 {
    width: 150px;
  }
}

.table-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0; // 移除 padding，让表格占满卡片

  .custom-table {
    flex: 1;
    
    :deep(th.el-table__cell) {
      background-color: var(--tm-bg-sidebar);
      color: var(--tm-text-secondary);
      font-weight: 500;
      border-bottom: 1px solid var(--tm-border-color);
    }

    :deep(td.el-table__cell) {
      border-bottom: 1px solid var(--tm-border-color-light);
    }
    
    :deep(.el-table__inner-wrapper::before) {
      display: none;
    }
  }
}

.tags-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  
  .custom-tag {
    border-radius: 4px;
    font-weight: 500;
  }
  
  .no-tag {
    color: var(--tm-text-secondary);
  }
}

.pagination-wrapper {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid var(--tm-border-color-light);
  background-color: var(--tm-bg-main);
}

.mint-btn {
  background-color: var(--tm-accent-primary, #52c48f);
  border-color: var(--tm-accent-primary, #52c48f);
  
  &:hover {
    background-color: #45b07e;
    border-color: #45b07e;
  }
}
</style>