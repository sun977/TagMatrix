<template>
  <div class="rule-group">
    <div class="group-header">
      <el-select v-model="modelValue.logic" class="logic-select" size="small">
        <el-option label="AND (满足所有)" value="and" />
        <el-option label="OR (满足任一)" value="or" />
      </el-select>
      <div class="group-actions">
        <el-button type="primary" link size="small" @click="addCondition">
          <el-icon><Plus /></el-icon> 添加条件
        </el-button>
        <el-button type="success" link size="small" @click="addGroup">
          <el-icon><FolderAdd /></el-icon> 添加条件组
        </el-button>
        <el-button v-if="!isRoot" type="danger" link size="small" @click="$emit('remove')">
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="group-content">
      <div v-for="(item, index) in modelValue.conditions" :key="index" class="condition-item-wrapper">
        <!-- 递归渲染子组 -->
        <RuleGroup
          v-if="item.logic"
          v-model="modelValue.conditions[index]"
          :is-root="false"
          @remove="removeCondition(index)"
        />
        
        <!-- 渲染普通条件 -->
        <div v-else class="condition-row">
          <el-input v-model="item.field" placeholder="字段名 (例如: os, port)" size="small" style="width: 180px;" />
          <el-select v-model="item.operator" size="small" style="width: 120px;">
            <el-option label="等于" value="equals" />
            <el-option label="包含" value="contains" />
            <el-option label="正则" value="regex" />
            <el-option label="大于" value="greater_than" />
            <el-option label="小于" value="less_than" />
          </el-select>
          <el-input v-model="item.value" placeholder="匹配值" size="small" style="width: 200px;" />
          <el-button type="danger" link size="small" @click="removeCondition(index)" class="remove-btn">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
      </div>
      
      <div v-if="modelValue.conditions.length === 0" class="empty-conditions">
        暂无条件，请点击上方按钮添加
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, FolderAdd, Delete, Close } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: any
  isRoot?: boolean
}>()

const emit = defineEmits(['update:modelValue', 'remove'])

const addCondition = () => {
  props.modelValue.conditions.push({
    field: '',
    operator: 'contains',
    value: ''
  })
}

const addGroup = () => {
  props.modelValue.conditions.push({
    logic: 'and',
    conditions: []
  })
}

const removeCondition = (index: number) => {
  props.modelValue.conditions.splice(index, 1)
}
</script>

<style scoped lang="scss">
.rule-group {
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius-sm);
  background-color: #fafafa;
  margin-bottom: 8px;
  padding: 12px;

  .group-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;

    .logic-select {
      width: 140px;
    }

    .group-actions {
      display: flex;
      gap: 12px;
    }
  }

  .group-content {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-left: 24px;
    border-left: 2px solid #e4e7ed;

    .condition-row {
      display: flex;
      align-items: center;
      gap: 8px;
      background-color: #ffffff;
      padding: 8px 12px;
      border: 1px solid var(--tm-border-color);
      border-radius: 4px;

      .remove-btn {
        margin-left: auto;
      }
    }

    .empty-conditions {
      font-size: 12px;
      color: #909399;
      font-style: italic;
    }
  }
}
</style>