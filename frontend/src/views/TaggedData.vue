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
            clearable
            class="input-with-select"
          >
            <template #prepend>
              <el-select v-model="filterForm.searchCol" placeholder="全部字段" style="width: 120px" clearable>
                <el-option label="全部字段" value="" />
                <el-option v-for="col in dynamicColumns" :key="col" :label="col" :value="col" />
              </el-select>
            </template>
            <template #append>
              <el-button :icon="Search" @click="handleSearch" />
            </template>
          </el-input>
        </el-form-item>
        
        <el-form-item label="标签分类">
          <el-select v-model="filterForm.tag" placeholder="全部标签" clearable class="w-150">
            <el-option v-for="tag in tagOptions" :key="tag.id" :label="tag.name" :value="String(tag.id)" />
          </el-select>
        </el-form-item>

        <el-form-item label="任务名称">
          <el-select v-model="filterForm.batch" placeholder="全部任务" clearable class="w-150">
            <el-option v-for="batch in batchOptions" :key="batch.id" :label="batch.name" :value="String(batch.id)" />
          </el-select>
        </el-form-item>

        <el-form-item label="数据来源">
          <el-select v-model="filterForm.dataSource" placeholder="全部来源" clearable class="w-150">
            <el-option v-for="ds in dataSourceOptions" :key="ds.source_name" :label="ds.source_name" :value="ds.source_name" />
          </el-select>
        </el-form-item>

        <el-form-item label="打标模式">
          <el-select v-model="filterForm.tagMode" placeholder="全部模式" clearable class="w-150">
            <el-option label="单标签" value="single" />
            <el-option label="多标签" value="multiple" />
            <el-option label="混合模式" value="mixed" />
          </el-select>
        </el-form-item>

        <el-form-item label="状态">
          <el-select v-model="filterForm.status" placeholder="全部状态" clearable class="w-150">
            <el-option label="已打标" value="success" />
            <el-option label="未命中" value="unmatched" />
          </el-select>
        </el-form-item>

        <el-form-item label="打标时间">
          <el-date-picker
            v-model="filterForm.dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            clearable
            style="width: 240px"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" class="mint-btn" :icon="Search" @click="handleSearch">查询</el-button>
          <el-button :icon="RefreshRight" @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 数据表格区 (卡片) -->
    <div class="table-card card-panel">
      <div class="table-header-actions" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
        <span style="font-size: 14px; color: var(--tm-text-secondary);">共 {{ totalItems }} 条数据</span>
        <div class="action-icons">
          <el-button circle @click="handleSearch">
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

      <el-table 
        :data="tableData" 
        style="width: 100%" 
        v-loading="loading"
        class="custom-table"
      >
        <el-table-column prop="id" label="数据 ID" width="100" fixed="left" />
        
        <!-- 动态列 (原始数据的所有字段) -->
        <el-table-column 
          v-for="col in visibleColumns" 
          :key="col" 
          :prop="col" 
          :label="col" 
          min-width="150" 
          show-overflow-tooltip 
        />

        <!-- 系统处理字段 -->
        <el-table-column label="打标模式" width="120">
          <template #default="scope">
            <el-tag size="small" type="info">
              {{ formatTagMode(scope.row.tagMode) }}
            </el-tag>
          </template>
        </el-table-column>

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

        <el-table-column label="命中主标签" min-width="120">
          <template #default="scope">
            <div v-if="scope.row.tagMode === 'mixed' && scope.row.primaryTag">
              <el-tag 
                :color="scope.row.primaryTag.color + '20'"
                :style="{ color: scope.row.primaryTag.color, borderColor: scope.row.primaryTag.color + '40' }"
                size="small"
                class="custom-tag"
                disable-transitions
              >
                {{ scope.row.primaryTag.name }}
              </el-tag>
            </div>
            <span v-else class="no-tag">-</span>
          </template>
        </el-table-column>

        <el-table-column prop="batchName" label="任务批次" min-width="150" show-overflow-tooltip />
        <el-table-column prop="dataSource" label="数据来源" min-width="120" show-overflow-tooltip />
        <el-table-column prop="updateTime" label="打标时间" width="160" />
          
          <el-table-column label="状态" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.status === 'success' ? 'success' : 'info'" size="small">
                {{ scope.row.status === 'success' ? '已打标' : '未命中' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="100" fixed="right" align="center">
            <template #default="scope">
              <el-button type="primary" link size="small" class="detail-btn" @click="handleViewDetail(scope.row)">查看详情</el-button>
            </template>
          </el-table-column>
        </el-table>

      <!-- 分页器 -->
      <div class="pagination-wrapper">
        <el-pagination
          :current-page="currentPage"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="totalItems"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>

    <!-- 查看详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="打标数据详情 (JSON)"
      width="600px"
    >
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { Search, Download, RefreshRight, Setting, DocumentCopy } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { GetTaggedDataList, ExportTaggedDataList, GetAllTags, GetTaskBatches, GetAvailableDataSources } from '../../wailsjs/go/main/App'

const loading = ref(false)

// 过滤表单状态
const filterForm = reactive({
  keyword: '',
  searchCol: '',
  tag: '',
  batch: '',
  dataSource: '',
  tagMode: '',
  status: '',
  dateRange: null as string[] | null
})

// 动态列名
const dynamicColumns = ref<string[]>([])
const hiddenColumns = ref<string[]>([])

// 查看详情相关的状态
const detailDialogVisible = ref(false)
const formattedDetailJson = ref('')

// 计算需要展示的列
const visibleColumns = computed(() => {
  return dynamicColumns.value.filter(col => !hiddenColumns.value.includes(col))
})

const toggleColumn = (col: string) => {
  const idx = hiddenColumns.value.indexOf(col)
  if (idx > -1) {
    hiddenColumns.value.splice(idx, 1)
  } else {
    hiddenColumns.value.push(col)
  }
}

const formatTagMode = (mode: string) => {
  switch (mode) {
    case 'single': return '单标签'
    case 'multiple': return '多标签'
    case 'mixed': return '混合模式'
    default: return '未知'
  }
}

// 下拉选项数据
const tagOptions = ref<any[]>([])
const batchOptions = ref<any[]>([])
const dataSourceOptions = ref<any[]>([])

// 分页状态
const currentPage = ref(1)
const pageSize = ref(20)
const totalItems = ref(0) 

// 表格数据
const tableData = ref<any[]>([])

const handleSearch = async () => {
  loading.value = true
  try {
    const res = await GetTaggedDataList(
      filterForm.keyword, 
      filterForm.tag, 
      filterForm.batch, 
      filterForm.searchCol, 
      filterForm.dataSource,
      filterForm.tagMode,
      filterForm.status,
      filterForm.dateRange && filterForm.dateRange.length === 2 ? filterForm.dateRange[0] : '',
      filterForm.dateRange && filterForm.dateRange.length === 2 ? filterForm.dateRange[1] : '',
      currentPage.value, 
      pageSize.value
    )
    if (res) {
      // 解析 JSON 数据并展平
      const parsedData = (res.records || []).map((r: any) => {
        let dataObj = {}
        try {
          if (r.content) {
            dataObj = JSON.parse(r.content)
          }
        } catch (e) {
          console.warn('Failed to parse record content:', r.content)
        }
        return {
          ...r,
          ...dataObj
        }
      })

      // 提取动态列头
      if (parsedData.length > 0) {
        const colSet = new Set<string>()
        parsedData.forEach((row: any) => {
          // 只把 JSON 内解析出来的 key 作为动态列
          let keys: string[] = []
          try {
            if (row.content) {
              keys = Object.keys(JSON.parse(row.content))
            }
          } catch(e) {}
          
          keys.forEach(k => {
            if (k !== 'id' && k !== '数据来源') {
              colSet.add(k)
            }
          })
        })
        let cols = Array.from(colSet)
        dynamicColumns.value = cols
      } else {
        dynamicColumns.value = []
      }

      tableData.value = parsedData
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
  filterForm.searchCol = ''
  filterForm.tag = ''
  filterForm.batch = ''
  filterForm.dataSource = ''
  filterForm.tagMode = ''
  filterForm.status = ''
  filterForm.dateRange = null
  currentPage.value = 1
  handleSearch()
}

const handleExport = async () => {
  try {
    await ExportTaggedDataList(
      filterForm.keyword, 
      filterForm.tag, 
      filterForm.batch, 
      filterForm.searchCol, 
      filterForm.dataSource,
      filterForm.tagMode,
      filterForm.status,
      filterForm.dateRange && filterForm.dateRange.length === 2 ? filterForm.dateRange[0] : '',
      filterForm.dateRange && filterForm.dateRange.length === 2 ? filterForm.dateRange[1] : ''
    )
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

const handleViewDetail = (row: any) => {
  // 排除掉不需要展示在 JSON 中的辅助字段
  const { id, updateTime, tags, primaryTag, ...rest } = row
  
  // 处理展示的标签名称
  const displayObj: any = { ...rest }
  
  if (tags && tags.length > 0) {
    displayObj['命中标签'] = tags.map((t: any) => t.name).join(', ')
  } else {
    displayObj['命中标签'] = '-'
  }

  if (row.tagMode === 'mixed' && primaryTag) {
    displayObj['命中主标签'] = primaryTag.name
  } else if (row.tagMode === 'mixed') {
    displayObj['命中主标签'] = '-'
  }
  
  // 格式化打标模式
  displayObj['打标模式'] = formatTagMode(row.tagMode)
  delete displayObj.tagMode
  
  // 重命名其它系统字段
  if (displayObj.batchName) {
    displayObj['任务批次'] = displayObj.batchName
    delete displayObj.batchName
  }
  if (displayObj.dataSource) {
    displayObj['数据来源'] = displayObj.dataSource
    delete displayObj.dataSource
  }
  if (displayObj.status) {
    displayObj['状态'] = displayObj.status === 'success' ? '已打标' : '未命中'
    delete displayObj.status
  }
  if (displayObj.content) {
    delete displayObj.content
  }

  formattedDetailJson.value = JSON.stringify(displayObj, null, 2)
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

const loadOptions = async () => {
  try {
    const tags = await GetAllTags()
    if (tags) tagOptions.value = tags

    const batches = await GetTaskBatches()
    if (batches) batchOptions.value = batches

    // TODO: 目前没有指定 dataset_id，获取所有的 data sources
    const ds = await GetAvailableDataSources(0)
    if (ds) dataSourceOptions.value = ds
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

.column-settings {
  display: flex;
  flex-direction: column;
  max-height: 300px;
  overflow-y: auto;
  padding-right: 8px;
}

.column-settings::-webkit-scrollbar {
  width: 6px;
}
.column-settings::-webkit-scrollbar-thumb {
  background-color: var(--tm-border-color);
  border-radius: 3px;
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

.detail-content-wrapper {
  max-height: 50vh;
  overflow-y: auto;
  background-color: #f5f7fa;
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
</style>