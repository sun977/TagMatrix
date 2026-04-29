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
          :schema-keys="schemaKeys"
          @remove="removeCondition(index)"
        />
        
        <!-- 渲染普通条件 -->
        <div v-else class="condition-row">
          <el-select 
            v-model="item.field" 
            filterable 
            allow-create 
            default-first-option 
            placeholder="字段名 (可手写)" 
            size="small" 
            style="width: 180px;"
          >
            <el-option v-for="key in schemaKeys" :key="key" :label="key" :value="key" />
          </el-select>
          <el-select v-model="item.operator" size="small" style="width: 120px;">
            <el-option-group label="基础比较">
              <el-option label="等于 (equals)" value="equals" />
              <el-option label="不等于 (not_equals)" value="not_equals" />
              <el-option label="存在 (exists)" value="exists" />
              <el-option label="为空 (is_null)" value="is_null" />
              <el-option label="不为空 (is_not_null)" value="is_not_null" />
            </el-option-group>
            
            <el-option-group label="文本匹配">
              <el-option label="包含 (contains)" value="contains" />
              <el-option label="不包含 (not_contains)" value="not_contains" />
              <el-option label="开头是 (starts_with)" value="starts_with" />
              <el-option label="结尾是 (ends_with)" value="ends_with" />
              <el-option label="正则匹配 (regex)" value="regex" />
              <el-option label="模糊匹配 (like)" value="like" />
            </el-option-group>

            <el-option-group label="数值/大小比较">
              <el-option label="大于 (>)" value="greater_than" />
              <el-option label="小于 (<)" value="less_than" />
              <el-option label="大于等于 (>=)" value="greater_than_or_equal" />
              <el-option label="小于等于 (<=)" value="less_than_or_equal" />
            </el-option-group>

            <el-option-group label="集合/特殊匹配">
              <el-option label="在列表中 (in)" value="in" />
              <el-option label="不在列表中 (not_in)" value="not_in" />
              <el-option label="列表包含 (list_contains)" value="list_contains" />
              <el-option label="IP网段 (cidr)" value="cidr" />
            </el-option-group>
          </el-select>

          <!-- 目标值输入 -->
          <el-input 
            v-if="!['exists', 'is_null', 'is_not_null'].includes(item.operator)"
            v-model="item.value" 
            placeholder="匹配值 (in操作用逗号分隔)" 
            size="small" 
            style="width: 200px;" 
          />

          <!-- 忽略大小写开关 -->
          <el-tooltip content="忽略大小写" placement="top" :enterable="false">
            <el-switch 
              v-model="item.ignore_case" 
              inline-prompt 
              style="--el-switch-on-color: #52c48f"
              size="small"
            />
          </el-tooltip>
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

<script lang="ts">
import { defineComponent } from 'vue'

// 显式添加默认导出以解决 Vetur/IDE 提示的 "has no default export" 模块导入报错
// 并在递归组件调用时提供明确的 name 属性
export default defineComponent({
  name: 'RuleGroup'
})
</script>

<script setup lang="ts">
import { Plus, FolderAdd, Delete, Close } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: any
  isRoot?: boolean
  schemaKeys?: string[]
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

// 处理在 template 中 v-for 的 index 可能是 string 或 number 的情况，修复潜在的 TS 类型报错
const removeCondition = (index: number | string) => {
  props.modelValue.conditions.splice(Number(index), 1)
}
</script>

<style scoped lang="scss">
.rule-group {
  border: 1px solid var(--tm-border-color);
  border-radius: var(--tm-border-radius-sm);
  background-color: var(--tm-bg-subtle);
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
    border-left: 2px solid var(--tm-border-color);

    .condition-row {
      display: flex;
      align-items: center;
      gap: 8px;
      background-color: var(--tm-bg-main);
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