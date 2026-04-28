<template>
  <el-dialog
    v-model="dialogVisible"
    title="全局设置"
    width="600px"
    class="settings-dialog"
    :show-close="true"
  >
    <div class="settings-content">
      <!-- AI 配置 -->
      <div class="settings-section">
        <h3>AI 配置</h3>
        
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
          <label>选择模型</label>
          <el-select v-model="form.model" class="w-100">
            <el-option label="gpt-4o" value="gpt-4o" />
            <el-option label="gpt-4-turbo" value="gpt-4-turbo" />
            <el-option label="gpt-3.5-turbo" value="gpt-3.5-turbo" />
          </el-select>
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

        <div class="setting-item">
          <label>System Prompt</label>
          <el-input 
            v-model="form.systemPrompt" 
            type="textarea" 
            :rows="4"
            placeholder="自定义 AI 的系统提示词..."
          />
          <div class="help-text">自定义 AI 的系统提示词，用于指导 AI 如何进行数据分析和打标决策。</div>
        </div>
      </div>

      <!-- 系统设置 -->
      <div class="settings-section">
        <h3>系统设置</h3>

        <div class="setting-item">
          <label>默认打标模式</label>
          <el-radio-group v-model="form.defaultMode" class="mode-radio-group">
            <el-radio label="overwrite" border>覆盖模式</el-radio>
            <el-radio label="append" border>追加模式</el-radio>
          </el-radio-group>
        </div>

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

      <!-- 高级设置 -->
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
            <label>开发者模式</label>
            <div class="help-text">开启后允许进入系统数据库管理的高级操作界面</div>
          </div>
          <el-switch v-model="form.developerMode" />
        </div>
      </div>
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
import { GetAppConfig, SaveAppConfig } from '../../wailsjs/go/main/App'
import { config } from '../../wailsjs/go/models'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const dialogVisible = ref(props.modelValue)

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
  apiKey: 'sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
  baseUrl: 'https://api.openai.com/v1',
  model: 'gpt-4o',
  temperature: 0.7,
  systemPrompt: '你是一个专业的数据分析师和标签专家，擅长根据用户提供的数据和规则，为用户提供准确的打标建议。请严格按照给定的规则进行分析，不要添加额外的解释，只返回打标结果。',
  defaultMode: 'overwrite',
  autoBackup: true,
  taskNotification: true,
  previewRows: 20,
  concurrency: 5,
  retries: 3,
  debugMode: false,
  developerMode: false
}

const form = reactive({ ...defaultForm })

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
      form.defaultMode = cfg.system.default_mode || 'overwrite'
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
  } catch (e) {
    console.error('Failed to load settings:', e)
    ElMessage.error('加载配置失败')
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
    newCfg.system.default_mode = form.defaultMode
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
  max-height: 60vh;
  overflow-y: auto;
  padding: 20px 24px;
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

.slider-marks {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--tm-text-secondary);
  margin-top: 4px;
}

.mode-radio-group {
  display: flex;
  gap: 16px;
  
  :deep(.el-radio.is-bordered) {
    margin-right: 0;
    flex: 1;
    border-radius: var(--tm-border-radius-sm);
  }
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>