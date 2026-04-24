<template>
  <div class="task-kanban-container">
    <div class="header-actions">
      <el-button type="primary" size="large" icon="VideoPlay" @click="dialogVisible = true">
        新建打标任务
      </el-button>
    </div>

    <!-- 历史任务批次卡片 -->
    <div class="task-list">
      <el-card class="task-card" shadow="hover" v-for="i in 3" :key="i">
        <div class="card-header">
          <div class="batch-name">
            <el-icon><Tickets /></el-icon>
            Batch_20260424_{{ i }}
          </div>
          <el-tag :type="i === 1 ? 'success' : 'info'" effect="light" round>
            {{ i === 1 ? '已完成' : '已回退' }}
          </el-tag>
        </div>
        
        <div class="card-body">
          <div class="stat-item">
            <span class="label">处理数据量</span>
            <span class="value">10,234</span>
          </div>
          <div class="stat-item">
            <span class="label">命中标签数</span>
            <span class="value">8,901</span>
          </div>
          <div class="stat-item">
            <span class="label">耗时</span>
            <span class="value">12.5s</span>
          </div>
        </div>

        <div class="card-footer">
          <el-button type="primary" link>查看审计日志</el-button>
          <el-popconfirm title="确定要撤销此次打标结果吗？此操作不可逆。" v-if="i === 1">
            <template #reference>
              <el-button type="danger" link>一键回退</el-button>
            </template>
          </el-popconfirm>
        </div>
      </el-card>
    </div>

    <!-- 发起任务对话框 -->
    <el-dialog v-model="dialogVisible" title="发起自动打标任务" width="500px">
      <el-form label-position="top">
        <el-form-item label="任务批次名称 (可选)">
          <el-input placeholder="例如：4月24日活跃用户打标" />
        </el-form-item>
        <el-form-item label="选择打标模式">
          <el-radio-group v-model="tagMode">
            <el-radio value="primary" border>作为主标签覆盖</el-radio>
            <el-radio value="secondary" border>作为次标签追加</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="选择要执行的规则">
          <el-select multiple placeholder="请选择规则（可多选）" style="width: 100%">
            <el-option label="高净值规则 A" value="1" />
            <el-option label="活跃规则 B" value="2" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="dialogVisible = false" :loading="isSubmitting">
            开始执行
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { VideoPlay, Tickets } from '@element-plus/icons-vue'

const dialogVisible = ref(false)
const tagMode = ref('secondary')
const isSubmitting = ref(false)
</script>

<style scoped lang="scss">
.task-kanban-container {
  .header-actions {
    margin-bottom: 24px;
  }

  .task-list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 20px;
  }

  .task-card {
    border-radius: var(--tm-border-radius);
    border: 1px solid var(--tm-border-color);
    box-shadow: var(--tm-shadow-sm);
    transition: box-shadow 0.2s;

    &:hover {
      box-shadow: var(--tm-shadow-md);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid var(--tm-border-color);

      .batch-name {
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--tm-text-primary);
      }
    }

    .card-body {
      display: flex;
      flex-direction: column;
      gap: 12px;
      margin-bottom: 20px;

      .stat-item {
        display: flex;
        justify-content: space-between;
        font-size: 14px;

        .label {
          color: var(--tm-text-secondary);
        }
        
        .value {
          font-weight: 500;
          color: var(--tm-text-primary);
        }
      }
    }

    .card-footer {
      display: flex;
      justify-content: space-between;
      padding-top: 12px;
      border-top: 1px dashed var(--tm-border-color);
    }
  }
}
</style>
