<template>
  <div class="tag-rule-container">
    <!-- 左侧标签树区域 -->
    <div class="left-pane">
      <div class="pane-header">
        <h3>标签体系</h3>
        <el-button type="primary" size="small" circle>
          <el-icon><Plus /></el-icon>
        </el-button>
      </div>
      
      <div class="pane-content">
        <el-input
          v-model="filterText"
          placeholder="搜索标签"
          prefix-icon="Search"
          clearable
        />
        
        <el-tree
          ref="treeRef"
          class="tag-tree"
          :data="tagData"
          :props="defaultProps"
          default-expand-all
          :filter-node-method="filterNode"
          @node-click="handleNodeClick"
        >
          <template #default="{ node, data }">
            <span class="custom-tree-node">
              <span class="tag-color-dot" :style="{ backgroundColor: data.color || 'var(--tm-accent-primary)' }"></span>
              <span>{{ node.label }}</span>
            </span>
          </template>
        </el-tree>
      </div>
    </div>

    <!-- 分割线 -->
    <div class="divider"></div>

    <!-- 右侧规则配置区域 -->
    <div class="right-pane">
      <template v-if="selectedTag">
        <div class="pane-header">
          <h3>配置规则: <el-tag effect="plain" :color="selectedTag.color" round>{{ selectedTag.label }}</el-tag></h3>
          <el-button type="success" size="small">保存规则</el-button>
        </div>

        <div class="pane-content rule-editor">
          <!-- 规则可视化编辑器占位 -->
          <el-alert title="规则构建器区域" type="info" description="这里将渲染复杂的布尔逻辑下拉框组合" show-icon />

          <!-- 试运行面板 -->
          <div class="dry-run-section">
            <el-button type="warning" plain icon="VideoPlay">试运行 (Dry Run)</el-button>
            <p class="help-text">在当前数据库中抽取 10 条真实数据进行匹配测试，不影响实际结果。</p>
            
            <el-table :data="[]" style="width: 100%; margin-top: 16px;" empty-text="点击上方按钮进行试运行">
              <el-table-column prop="id" label="记录ID" width="100" />
              <el-table-column prop="data" label="数据预览" />
              <el-table-column prop="matched" label="是否命中" width="120" />
            </el-table>
          </div>
        </div>
      </template>

      <!-- 未选择标签时的空状态 -->
      <div class="empty-state" v-else>
        <el-empty description="请在左侧选择一个标签以配置打标规则" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Plus, Search, VideoPlay } from '@element-plus/icons-vue'

const filterText = ref('')
const treeRef = ref<any>()

const selectedTag = ref<any>(null)

interface Tree {
  id: number
  label: string
  color?: string
  children?: Tree[]
}

// 模拟标签数据
const tagData: Tree[] = [
  {
    id: 1,
    label: '用户画像',
    children: [
      { id: 4, label: '高净值用户', color: '#ff4d4f' },
      { id: 9, label: '活跃用户', color: '#52c41a' },
    ],
  },
  {
    id: 2,
    label: '风险等级',
    children: [
      { id: 5, label: '高风险', color: '#f5222d' },
      { id: 6, label: '低风险', color: '#52c41a' },
    ],
  },
]

const defaultProps = {
  children: 'children',
  label: 'label',
}

watch(filterText, (val) => {
  treeRef.value!.filter(val)
})

const filterNode = (value: string, data: Tree) => {
  if (!value) return true
  return data.label.includes(value)
}

const handleNodeClick = (data: Tree) => {
  if (!data.children || data.children.length === 0) {
    selectedTag.value = data
  }
}
</script>

<style scoped lang="scss">
.tag-rule-container {
  display: flex;
  height: 100%;
  background: var(--tm-bg-main);
  border-radius: var(--tm-border-radius);
  border: 1px solid var(--tm-border-color);
  overflow: hidden;

  .pane-header {
    height: 56px;
    padding: 0 20px;
    border-bottom: 1px solid var(--tm-border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      display: flex;
      align-items: center;
      gap: 12px;
    }
  }

  .pane-content {
    padding: 20px;
    overflow-y: auto;
    flex: 1;
  }

  .left-pane {
    width: 280px;
    display: flex;
    flex-direction: column;
    background-color: var(--tm-bg-sidebar);

    .tag-tree {
      margin-top: 16px;
      background: transparent;

      .custom-tree-node {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;

        .tag-color-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }
      }
    }
  }

  .divider {
    width: 1px;
    background-color: var(--tm-border-color);
  }

  .right-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;

    .rule-editor {
      display: flex;
      flex-direction: column;
      gap: 32px;
    }

    .dry-run-section {
      margin-top: auto;
      padding-top: 24px;
      border-top: 1px dashed var(--tm-border-color);

      .help-text {
        font-size: 12px;
        color: var(--tm-text-secondary);
        margin-top: 8px;
      }
    }

    .empty-state {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
    }
  }
}
</style>
