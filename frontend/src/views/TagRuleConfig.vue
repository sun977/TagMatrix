<template>
  <div class="tag-rule-container">
    <!-- 左侧标签树区域 -->
    <div class="left-pane">
      <div class="pane-header">
        <h3>标签体系</h3>
        <el-button type="primary" size="small" circle @click="showAddTagDialog(0)">
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
        
        <div class="tree-wrapper" v-loading="loadingTags">
          <el-tree
            ref="treeRef"
            class="tag-tree"
            :data="tagTreeData"
            :props="defaultProps"
            node-key="id"
            default-expand-all
            :filter-node-method="filterNode"
            @node-click="handleNodeClick"
          >
            <template #default="{ node, data }">
              <span class="custom-tree-node">
                <span class="tag-color-dot" :style="{ backgroundColor: data.color || 'var(--tm-accent-primary)' }"></span>
                <span>{{ node.label }}</span>
                <span class="node-actions" @click.stop>
                  <el-icon @click.stop="showAddTagDialog(data.id)"><Plus /></el-icon>
                </span>
              </span>
            </template>
          </el-tree>
        </div>
      </div>
    </div>

    <!-- 分割线 -->
    <div class="divider"></div>

    <!-- 右侧规则配置区域 -->
    <div class="right-pane">
      <template v-if="selectedTag">
        <div class="pane-header">
          <h3>配置规则: <el-tag effect="plain" :color="selectedTag.color" round>{{ selectedTag.label }}</el-tag></h3>
          <el-button type="success" size="small" @click="handleSaveRule" :loading="savingRule">保存规则</el-button>
        </div>

        <div class="pane-content rule-editor">
          <!-- 规则源码编辑器 (作为可视化前的替代) -->
          <el-alert title="规则 JSON 编辑器" type="info" description="请按照 Matcher 的规范编写匹配规则 JSON" show-icon style="margin-bottom: 16px;" />
          <el-input
            v-model="ruleJsonText"
            type="textarea"
            :rows="8"
            placeholder='{"type": "rule", "field": "age", "operator": "gt", "value": 18}'
            style="font-family: monospace;"
          />

          <!-- 试运行面板 -->
          <div class="dry-run-section">
            <el-button type="warning" plain icon="VideoPlay" @click="handleDryRun" :loading="runningDry">试运行 (Dry Run)</el-button>
            <p class="help-text">在当前数据库中抽取 10 条真实数据进行匹配测试，不影响实际结果。</p>
            
            <el-table :data="dryRunResults" style="width: 100%; margin-top: 16px;" empty-text="点击上方按钮进行试运行" border v-loading="runningDry">
              <el-table-column prop="RecordID" label="记录ID" width="100" />
              <el-table-column prop="Data" label="数据预览" show-overflow-tooltip />
              <el-table-column prop="Matched" label="是否命中" width="120">
                <template #default="scope">
                  <el-tag :type="scope.row.Matched ? 'success' : 'danger'">
                    {{ scope.row.Matched ? '命中' : '未命中' }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </template>

      <!-- 未选择标签时的空状态 -->
      <div class="empty-state" v-else>
        <el-empty description="请在左侧选择一个标签以配置打标规则" />
      </div>
    </div>

    <!-- 添加标签对话框 -->
    <el-dialog v-model="dialogVisible" title="新增标签" width="400px">
      <el-form :model="tagForm" label-width="80px">
        <el-form-item label="标签名称" required>
          <el-input v-model="tagForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="tagForm.description" type="textarea" />
        </el-form-item>
        <el-form-item label="颜色">
          <el-color-picker v-model="tagForm.color" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitAddTag" :loading="savingTag">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Plus, Search, VideoPlay } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { CreateTag, GetAllTags, SaveRule, DryRunRule } from '../../wailsjs/go/main/App'
import { model } from '../../wailsjs/go/models'

const filterText = ref('')
const treeRef = ref<any>()

// --- 左侧标签树逻辑 ---
const loadingTags = ref(false)
const tagTreeData = ref<any[]>([])

const defaultProps = {
  children: 'children',
  label: 'label',
}

// 扁平数据转树形
const buildTree = (tags: model.SysTag[], parentId: number): any[] => {
  const result: any[] = []
  for (const tag of tags) {
    if (tag.parent_id === parentId) {
      const node: any = {
        id: tag.id,
        label: tag.name,
        color: tag.color,
        raw: tag,
        children: buildTree(tags, tag.id)
      }
      if (node.children.length === 0) {
        delete node.children
      }
      result.push(node)
    }
  }
  return result
}

const fetchTags = async () => {
  loadingTags.value = true
  try {
    const rawTags = await GetAllTags()
    tagTreeData.value = buildTree(rawTags || [], 0)
  } catch (e: any) {
    ElMessage.error('获取标签失败: ' + String(e))
  } finally {
    loadingTags.value = false
  }
}

watch(filterText, (val) => {
  treeRef.value!.filter(val)
})

const filterNode = (value: string, data: any) => {
  if (!value) return true
  return data.label.includes(value)
}

// --- 右侧规则逻辑 ---
const selectedTag = ref<any>(null)
const ruleJsonText = ref('')
const savingRule = ref(false)
const runningDry = ref(false)
const dryRunResults = ref<any[]>([])

const handleNodeClick = (data: any) => {
  // 这里暂时只允许给没有子节点的叶子标签挂载规则
  if (!data.children || data.children.length === 0) {
    selectedTag.value = data
    // 在真实业务中这里还需要去查一遍是否已有保存的规则，目前先留空
    ruleJsonText.value = ''
    dryRunResults.value = []
  }
}

const handleSaveRule = async () => {
  if (!ruleJsonText.value) {
    ElMessage.warning('规则不能为空')
    return
  }
  savingRule.value = true
  try {
    const rule = new model.SysMatchRule()
    rule.tag_id = selectedTag.value.id
    rule.name = selectedTag.value.label + "的规则"
    rule.rule_json = ruleJsonText.value
    rule.is_enabled = true
    rule.priority = 0

    await SaveRule(rule)
    ElMessage.success('规则保存成功')
  } catch (e: any) {
    ElMessage.error('保存规则失败，可能是 JSON 格式错误: ' + String(e))
  } finally {
    savingRule.value = false
  }
}

const handleDryRun = async () => {
  if (!ruleJsonText.value) {
    ElMessage.warning('请先输入规则')
    return
  }
  runningDry.value = true
  try {
    const res = await DryRunRule(ruleJsonText.value, 10)
    dryRunResults.value = res || []
    if (dryRunResults.value.length === 0) {
      ElMessage.info('当前数据库中没有可供试运行的数据')
    }
  } catch (e: any) {
    ElMessage.error('试运行失败，请检查规则 JSON: ' + String(e))
  } finally {
    runningDry.value = false
  }
}

// --- 添加标签弹窗逻辑 ---
const dialogVisible = ref(false)
const savingTag = ref(false)
const tagForm = ref({
  name: '',
  description: '',
  color: '#52c48f',
  parentId: 0
})

const showAddTagDialog = (parentId: number) => {
  tagForm.value = { name: '', description: '', color: '#52c48f', parentId }
  dialogVisible.value = true
}

const submitAddTag = async () => {
  if (!tagForm.value.name) {
    ElMessage.warning('请输入标签名称')
    return
  }
  savingTag.value = true
  try {
    const tag = new model.SysTag()
    tag.name = tagForm.value.name
    tag.description = tagForm.value.description
    tag.color = tagForm.value.color
    tag.parent_id = tagForm.value.parentId

    await CreateTag(tag)
    ElMessage.success('标签创建成功')
    dialogVisible.value = false
    fetchTags() // 刷新树
  } catch (e: any) {
    ElMessage.error('创建标签失败: ' + String(e))
  } finally {
    savingTag.value = false
  }
}

onMounted(() => {
  fetchTags()
})
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

    .tree-wrapper {
      margin-top: 16px;
      flex: 1;
      overflow-y: auto;
    }

    .tag-tree {
      background: transparent;

      .custom-tree-node {
        display: flex;
        align-items: center;
        width: 100%;
        gap: 8px;
        font-size: 14px;
        position: relative;

        .tag-color-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }

        .node-actions {
          margin-left: auto;
          opacity: 0;
          color: var(--tm-accent-primary);
          transition: opacity 0.2s;
        }

        &:hover .node-actions {
          opacity: 1;
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
      gap: 16px;
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
