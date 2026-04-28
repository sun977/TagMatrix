<template>
  <div class="backup-restore-container">
    <div class="header-actions">
      <el-button type="primary" @click="handleCreateBackup" :loading="isCreating">
        <el-icon><CopyDocument /></el-icon> 创建快照
      </el-button>
      <el-button type="success" @click="handleImportDB" :loading="isImporting">
        <el-icon><Upload /></el-icon> 导入数据库 (.db)
      </el-button>
    </div>

    <el-table :data="backups" style="width: 100%" v-loading="isLoading" border stripe>
      <el-table-column prop="name" label="文件名" min-width="200" />
      <el-table-column prop="size" label="文件大小" width="120">
        <template #default="scope">
          {{ formatSize(scope.row.size) }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column prop="note" label="备注" min-width="150" show-overflow-tooltip />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="scope">
          <el-button size="small" type="primary" plain @click="handleRestore(scope.row)">恢复</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createDialogVisible" title="创建备份" width="400px">
      <el-form :model="createForm" label-width="60px">
        <el-form-item label="备注">
          <el-input v-model="createForm.note" placeholder="请输入备份说明..." maxlength="50" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitCreateBackup" :loading="isCreating">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ListBackups, CreateBackup, RestoreDatabase, DeleteBackup, ImportExternalDatabase } from '../../../wailsjs/go/main/App'

const backups = ref<any[]>([])
const isLoading = ref(false)
const isCreating = ref(false)
const isImporting = ref(false)
const createDialogVisible = ref(false)

const createForm = ref({
  note: ''
})

const fetchBackups = async () => {
  isLoading.value = true
  try {
    const res = await ListBackups()
    backups.value = res || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载备份列表失败')
  } finally {
    isLoading.value = false
  }
}

const handleCreateBackup = () => {
  createForm.value.note = ''
  createDialogVisible.value = true
}

const submitCreateBackup = async () => {
  isCreating.value = true
  try {
    await CreateBackup(createForm.value.note)
    ElMessage.success('备份创建成功')
    createDialogVisible.value = false
    fetchBackups()
  } catch (e: any) {
    ElMessage.error(e.message || '备份创建失败')
  } finally {
    isCreating.value = false
  }
}

const handleRestore = (row: any) => {
  ElMessageBox.confirm(
    `高危操作：此操作将使用选中的备份文件 [${row.name}] 覆盖当前所有的系统数据，覆盖后应用将自动重新加载且无法撤销！是否继续？`,
    '高危警告',
    {
      confirmButtonText: '强制恢复',
      cancelButtonText: '取消',
      type: 'error',
      center: true
    }
  ).then(async () => {
    try {
      await RestoreDatabase(row.path || row.name)
      ElMessage.success({
        message: '数据恢复成功，即将重新加载应用...',
        duration: 2000,
        onClose: () => {
          window.location.reload()
        }
      })
    } catch (e: any) {
      ElMessage.error(`恢复失败：${e.message}`)
    }
  }).catch(() => {})
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定要删除此备份文件吗？', '提示', {
    type: 'warning'
  }).then(async () => {
    try {
      await DeleteBackup(row.path || row.name)
      ElMessage.success('删除成功')
      fetchBackups()
    } catch (e: any) {
      ElMessage.error(e.message || '删除失败')
    }
  }).catch(() => {})
}

const handleImportDB = async () => {
  isImporting.value = true
  try {
    await ImportExternalDatabase()
    ElMessage.success({
      message: '数据恢复成功，即将重新加载应用...',
      duration: 2000,
      onClose: () => {
        window.location.reload()
      }
    })
  } catch (e: any) {
    if (e.message === 'cancelled') {
      ElMessage.info('已取消导入')
    } else {
      ElMessage.error(`导入失败：${e.message || e}`)
    }
  } finally {
    isImporting.value = false
  }
}

const formatSize = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(() => {
  fetchBackups()
})
</script>

<style scoped lang="scss">
.backup-restore-container {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--tm-bg-main);

  .header-actions {
    display: flex;
    gap: 12px;
    margin-bottom: 20px;
  }
}
</style>