<template>
  <el-dialog
    v-model="dialogVisible"
    title="全局设置"
    width="700px"
    class="settings-dialog"
    :show-close="true"
  >
    <div class="settings-content">
      <el-tabs v-model="activeTab" tab-position="left" class="settings-tabs">
        <!-- 通用设置 -->
        <el-tab-pane label="通用设置" name="general">
          <div class="settings-section">
            <h3>通用设置</h3>
            <div class="setting-item flex-between">
              <div class="item-text">
                <label>自动备份打标结果</label>
                <div class="help-text">每次打标任务完成后自动备份数据到本地</div>
              </div>
              <el-switch v-model="form.autoBackup" />
            </div>

            <div class="setting-item flex-between">
              <div class="item-text">
                <label>任务完成通知</label>
                <div class="help-text">打标任务完成后发送系统通知</div>
              </div>
              <el-switch v-model="form.taskNotification" />
            </div>

            <div class="setting-item">
              <label>数据预览默认行数</label>
              <el-select v-model="form.previewRows" class="w-100">
                <el-option label="10 行" :value="10" />
                <el-option label="20 行" :value="20" />
                <el-option label="50 行" :value="50" />
              </el-select>
            </div>
          </div>
        </el-tab-pane>

        <!-- AI 模型配置 -->
        <el-tab-pane label="AI 模型配置" name="ai">
          <div class="settings-section">
            <h3>AI 模型配置</h3>
            <div class="setting-item">
              <label>OpenAI API Key</label>
              <el-input 
                v-model="form.apiKey" 
                type="password" 
                show-password
                placeholder="sk-..."
              />
            </div>

            <div class="setting-item">
              <label>Base URL</label>
              <el-input 
                v-model="form.baseUrl" 
                placeholder="https://api.openai.com/v1"
              />
            </div>

            <div class="setting-item">
              <label>选择模型 (Model Name)</label>
              <el-input 
                v-model="form.model" 
                placeholder="gpt-4o / gpt-4o-mini / claude-3.5-sonnet"
              />
            </div>

            <div class="setting-item">
              <div class="label-with-value">
                <label>温度系数 (Temperature)</label>
                <span class="value-text">{{ form.temperature }}</span>
              </div>
              <el-slider 
                v-model="form.temperature" 
                :min="0" :max="1" :step="0.1" 
                :show-tooltip="false"
              />
              <div class="slider-marks">
                <span>0</span>
                <span>0.5</span>
                <span>1</span>
              </div>
            </div>

            <div class="setting-item mt-4">
              <el-button type="primary" plain @click="testAIConnection" :loading="isTesting">
                测试连接
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <!-- Prompt 与策略 -->
        <el-tab-pane label="Prompt 与策略" name="prompts">
          <div class="settings-section">
            <h3>系统提示词管理</h3>
            <div class="setting-item">
              <label>System Prompt</label>
              <el-input 
                v-model="form.systemPrompt" 
                type="textarea" 
                :rows="10"
                placeholder="自定义 AI 的系统提示词..."
              />
              <div class="help-text">自定义 AI 的系统提示词，用于指导 AI 如何进行数据分析和打标决策。</div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 高级与系统 -->
        <el-tab-pane label="高级与系统" name="advanced">
          <div class="settings-section">
            <h3>本地存储信息</h3>
            
            <div class="setting-item">
              <label>数据库路径</label>
              <div class="path-display">
                <el-input v-model="appPaths.dbPath" readonly size="small" />
                <el-button size="small" @click="openDir(appPaths.dbPath)">打开目录</el-button>
              </div>
            </div>

            <div class="setting-item">
              <label>运行日志路径</label>
              <div class="path-display">
                <el-input v-model="appPaths.logPath" readonly size="small" />
                <el-button size="small" @click="openDir(appPaths.logPath)">打开目录</el-button>
              </div>
            </div>
          </div>

          <div class="settings-section">
            <h3>高级设置</h3>
            <div class="setting-item">
              <label>AI 请求并发数</label>
              <el-input-number v-model="form.concurrency" :min="1" :max="20" class="w-100" controls-position="right" />
              <div class="help-text">同时发送的 AI 请求数量，过高可能会触发 API 限流。</div>
            </div>

            <div class="setting-item">
              <label>API 请求失败重试次数</label>
              <el-input-number v-model="form.retries" :min="0" :max="5" class="w-100" controls-position="right" />
            </div>

            <div class="setting-item flex-between">
              <div class="item-text">
                <label>调试模式</label>
                <div class="help-text">开启后会记录详细的请求和响应日志</div>
              </div>
              <el-switch v-model="form.debugMode" />
            </div>

            <div class="setting-item flex-between">
              <div class="item-text">
                <label @click="handleDeveloperClick" class="cursor-pointer select-none">开发者模式</label>
                <div class="help-text">开启后允许进入系统数据库管理的高级操作界面</div>
              </div>
              <el-switch v-model="form.developerMode" :disabled="!developerUnlocked && !form.developerMode" />
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="resetDefaults">重置默认值</el-button>
        <div>
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveSettings">保存设置</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { GetAppConfig, SaveAppConfig, TestAIConnection, GetAppPaths, OpenDirectoryInOS } from '../../wailsjs/go/main/App'
import { config } from '../../wailsjs/go/models'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const dialogVisible = ref(props.modelValue)
const activeTab = ref('general')

watch(() => props.modelValue, (newVal) => {
  dialogVisible.value = newVal
  if (newVal) {
    loadSettings()
  }
})

watch(() => dialogVisible.value, (newVal) => {
  emit('update:modelValue', newVal)
})

const defaultForm = {
  apiKey: '',
  baseUrl: 'https://api.openai.com/v1',
  model: 'gpt-4o-mini',
  temperature: 0.7,
  systemPrompt: `你是一个专业的数据分析和打标辅助助手。
你的主要任务是：
1. 解答用户关于 TagMatrix 数据打标系统如何使用的问题。
2. 根据用户提供的需求，生成针对当前 SQLite 数据库的准确查询 SQL。
3. 帮助用户分析数据特征。

请注意：
- 用户的原始导入数据存储在 raw_data_records 表的 data 字段中（JSON 格式）。在 SQLite 中查询 JSON 数据请使用 json_extract 函数。
- 给出 SQL 时请使用 markdown 代码块包裹，以便前端渲染。`,
  autoBackup: true,
  taskNotification: true,
  previewRows: 20,
  concurrency: 5,
  retries: 3,
  debugMode: false,
  developerMode: false
}

const form = reactive({ ...defaultForm })
const isTesting = ref(false)
const appPaths = reactive({ dbPath: '', logPath: '' })

// 开发者模式解锁逻辑
const developerUnlocked = ref(false)
let clickCount = 0
let clickTimer: any = null

const handleDeveloperClick = () => {
  if (developerUnlocked.value || form.developerMode) return
  
  clickCount++
  if (clickCount >= 5) {
    developerUnlocked.value = true
    ElMessage.success('开发者模式开关已解锁')
    clickCount = 0
  }

  if (clickTimer) clearTimeout(clickTimer)
  clickTimer = setTimeout(() => {
    clickCount = 0
  }, 1500)
}

const loadSettings = async () => {
  try {
    const cfg = await GetAppConfig()
    if (cfg && cfg.ai) {
      form.apiKey = cfg.ai.api_key || ''
      form.baseUrl = cfg.ai.base_url || ''
      form.model = cfg.ai.model || ''
      form.temperature = cfg.ai.temperature || 0.7
      form.systemPrompt = cfg.ai.system_prompt || ''
    }
    if (cfg && cfg.system) {
      form.autoBackup = cfg.system.auto_backup
      form.taskNotification = cfg.system.task_notification
      form.previewRows = cfg.system.preview_rows || 20
    }
    if (cfg && cfg.adv) {
      form.concurrency = cfg.adv.concurrency || 5
      form.retries = cfg.adv.retries || 3
      form.debugMode = cfg.adv.debug_mode
      form.developerMode = cfg.adv.developer_mode
    }

    const paths = await GetAppPaths()
    if (paths) {
      appPaths.dbPath = paths.dbPath
      appPaths.logPath = paths.logPath
    }
  } catch (e) {
    console.error('Failed to load settings:', e)
    ElMessage.error('加载配置失败')
  }
}

const testAIConnection = async () => {
  if (!form.apiKey) {
    ElMessage.warning('请先填写 API Key')
    return
  }
  isTesting.value = true
  try {
    await TestAIConnection(form.apiKey, form.baseUrl, form.model)
    ElMessage.success('连接成功！API 密钥与网络均正常。')
  } catch (err: any) {
    ElMessage.error('连接失败: ' + err.message)
  } finally {
    isTesting.value = false
  }
}

const openDir = async (path: string) => {
  if (path) {
    await OpenDirectoryInOS(path)
  }
}

const resetDefaults = () => {
  Object.assign(form, defaultForm)
}

const saveSettings = async () => {
  try {
    const newCfg = new config.AppConfig()
    newCfg.ai = new config.AIConfig()
    newCfg.ai.api_key = form.apiKey
    newCfg.ai.base_url = form.baseUrl
    newCfg.ai.model = form.model
    newCfg.ai.temperature = form.temperature
    newCfg.ai.system_prompt = form.systemPrompt

    newCfg.system = new config.SystemConfig()
    newCfg.system.auto_backup = form.autoBackup
    newCfg.system.task_notification = form.taskNotification
    newCfg.system.preview_rows = form.previewRows

    newCfg.adv = new config.AdvConfig()
    newCfg.adv.concurrency = form.concurrency
    newCfg.adv.retries = form.retries
    newCfg.adv.debug_mode = form.debugMode
    newCfg.adv.developer_mode = form.developerMode

    await SaveAppConfig(newCfg)
    ElMessage.success('设置已保存')
    
    emit('saved')
    dialogVisible.value = false
  } catch (e) {
    console.error('Failed to save settings:', e)
    ElMessage.error('保存设置失败')
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped lang="scss">
.settings-dialog {
  :deep(.el-dialog__header) {
    border-bottom: 1px solid var(--tm-border-color);
    margin-right: 0;
    padding-bottom: 16px;
    font-weight: 600;
  }
  
  :deep(.el-dialog__body) {
    padding: 0;
  }

  :deep(.el-dialog__footer) {
    border-top: 1px solid var(--tm-border-color);
    padding: 16px 20px;
  }
}

.settings-content {
  height: 60vh;
  display: flex;
}

.settings-tabs {
  width: 100%;
  :deep(.el-tabs__header.is-left) {
    margin-right: 0;
    width: 160px;
    background-color: var(--tm-bg-subtle);
    padding-top: 10px;
  }
  :deep(.el-tabs__item) {
    text-align: left;
    padding: 0 20px;
    justify-content: flex-start;
  }
  :deep(.el-tabs__content) {
    padding: 20px 24px;
    height: 100%;
    overflow-y: auto;
  }
}

.settings-section {
  margin-bottom: 32px;

  &:last-child {
    margin-bottom: 0;
  }

  h3 {
    margin: 0 0 20px 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--tm-text-primary);
  }
}

.setting-item {
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }

  label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: var(--tm-text-primary);
    margin-bottom: 8px;
  }

  .label-with-value {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    label {
      margin-bottom: 0;
    }
    
    .value-text {
      font-size: 14px;
      font-weight: 600;
      color: var(--tm-text-primary);
    }
  }

  .help-text {
    font-size: 12px;
    color: var(--tm-text-secondary);
    margin-top: 6px;
  }

  .w-100 {
    width: 100%;
  }
}

.flex-between {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .item-text {
    label {
      margin-bottom: 4px;
    }
    .help-text {
      margin-top: 0;
    }
  }
}

.path-display {
  display: flex;
  gap: 10px;
  align-items: center;
}

.slider-marks {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--tm-text-secondary);
  margin-top: 4px;
}

.mt-4 {
  margin-top: 16px;
}

.cursor-pointer {
  cursor: pointer;
}
.select-none {
  user-select: none;
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>