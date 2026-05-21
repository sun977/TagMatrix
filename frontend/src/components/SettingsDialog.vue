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

            <div class="setting-item">
              <label>主题与外观</label>
              <el-radio-group v-model="form.theme" class="mode-radio-group">
                <el-radio label="light" border class="theme-radio">亮色模式</el-radio>
                <el-radio label="dark" border class="theme-radio">暗色模式</el-radio>
                <el-radio label="auto" border class="theme-radio">跟随系统</el-radio>
              </el-radio-group>
            </div>

            <div class="setting-item flex-between mt-4">
              <div class="item-text">
                <label>任务完成通知</label>
                <div class="help-text">打标任务完成后发送系统横幅通知</div>
              </div>
              <el-switch v-model="form.taskNotification" />
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
              <el-button 
                :type="testSuccess ? 'success' : 'primary'" 
                :plain="!testSuccess" 
                @click="testAIConnection" 
                :loading="isTesting"
              >
                {{ testSuccess ? '连接成功' : '测试连接' }}
              </el-button>
            </div>
          </div>
        </el-tab-pane>

        <!-- Prompt 与策略 -->
        <el-tab-pane label="Prompt 设置" name="prompts">
          <div class="settings-section">
            <h3>系统提示词</h3>
            <div class="setting-item">
              <!-- <label>System Prompt</label> -->
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

        <!-- 打标算法设置 -->
        <el-tab-pane label="MDCT 设置" name="mdct">
          <div class="settings-section">
            <h3>
              多维共识打标算法 (MDCT) 权重
              <el-tooltip effect="dark" placement="bottom-start" max-width="400px">
                <template #content>
                  <div style="line-height: 1.5; font-size: 12px;">
                    <strong>【打分机制说明】</strong><br/>
                    当数据命中多个标签产生冲突时，系统按以下 4 个维度加权求和决出主标签：<br/>
                    <strong>1. 优先级得分(W1)：</strong>系统人工意志。通过 W1 的高权重产生总分断层碾压。只要 W1 足够大，微小的 Priority 差距即可决定胜负。<br/>
                    <strong>2. 逻辑深度(W2)：</strong>代表画像精准度。依据规则 AST 树打分（多一层 AND/OR +10分）。精确匹配(如 equals) 乘1.5倍；正则/范围 乘1.2倍；模糊匹配 乘1.0倍。<br/>
                    <strong>3. 数据置信度(W3)：</strong>依据数据“画像丰满度”打分。考察核心字段的非空完整度，倾向把主标签颁发给字段填写详尽的数据。<br/>
                    <strong>4. AI 语义裁决(W4)：</strong>延迟计算的仲裁者。仅开启 AI 且前 3 个维度总分差 &lt; 5% 时触发，由大模型打破平局。<br/>
                    <em>总分 = (W1 × 优先级) + (W2 × 逻辑深度) + (W3 × 置信度) + (W4 × AI 裁决)</em>
                  </div>
                </template>
                <el-icon style="font-size: 15px; color: var(--tm-text-secondary); cursor: help; vertical-align: -2px; margin-left: 4px; outline: none;">
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </h3>
            <div class="help-text" style="margin-bottom: 20px;">
              用于调整标签冲突时的主标签判定优先级。注意：若希望人工配置的优先级具有一票否决权，W1 的值必须远大于 W2+W3 的总和。
            </div>

            <div class="setting-item">
              <label>人为静态权重(W1)</label>
              <el-input-number v-model="form.w1" :min="0" :step="1" controls-position="right" class="w-100" />
              <div class="help-text">基础决定权，默认 1000 以确保优先级统治权。</div>
            </div>

            <div class="setting-item">
              <label>规则逻辑深度权重(W2)</label>
              <el-input-number v-model="form.w2" :min="0" :step="1" controls-position="right" class="w-100" />
              <div class="help-text">依据匹配规则的复杂程度计算的客观数据复杂度得分(默认10)。</div>
            </div>

            <div class="setting-item">
              <label>数据置信度权重(W3)</label>
              <el-input-number v-model="form.w3" :min="0" :step="1" controls-position="right" class="w-100" />
              <div class="help-text">依据命中该规则的数据完整度质量进行打分的权重(默认10)。</div>
            </div>
            
            <div class="setting-item">
              <label>AI 语义裁决分权重(W4)</label>
              <el-input-number v-model="form.w4" :min="0" :step="1" controls-position="right" class="w-100" />
              <div class="help-text">AI 介入平局时的得分加成(默认100)。</div>
            </div>

            <div class="setting-item flex-between mt-4">
              <div class="item-text">
                <label>允许 AI 介入深度打标裁决</label>
                <div class="help-text">出于数据隐私保护，此功能默认关闭。如果关闭，在平局时将退回使用规则 ID 兜底，绝不上传数据。</div>
              </div>
              <el-tooltip
                :content="!form.apiKey ? '请先在「AI 模型配置」中填写 API Key 后方可开启此功能' : '开启 AI 深度打标裁决'"
                placement="top-end"
                :disabled="!!form.apiKey && !form.allowAiArbiter"
              >
                <div style="display: inline-block;">
                  <el-switch v-model="form.allowAiArbiter" :disabled="!form.apiKey" />
                </div>
              </el-tooltip>
            </div>
          </div>
        </el-tab-pane>

        <!-- 网络与代理 -->
        <el-tab-pane label="网络与代理" name="network">
          <div class="settings-section">
            <h3>代理设置</h3>
            <div class="setting-item">
              <label>代理模式</label>
              <el-radio-group v-model="form.proxyMode" class="mode-radio-group">
                <el-radio label="direct" border class="theme-radio">直连</el-radio>
                <el-radio label="system" border class="theme-radio">系统代理(默认)</el-radio>
                <el-radio label="custom" border class="theme-radio">自定义代理</el-radio>
              </el-radio-group>
            </div>

            <div class="setting-item" v-if="form.proxyMode === 'custom'">
              <label>代理服务器地址</label>
              <el-input 
                v-model="form.proxyUrl" 
                placeholder="例如：http://127.0.0.1:7890 或 socks5://127.0.0.1:1080"
              />
              <div class="help-text">支持 http(s) 或 socks5 代理</div>
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
        <el-button @click="resetDefaults">重置默认配置</el-button>
        <div>
          <el-button @click="dialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="saveSettings">应用并保存</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { QuestionFilled } from '@element-plus/icons-vue'
import { GetAppConfig, SaveAppConfig, TestAIConnection, GetAppPaths, OpenDirectoryInOS, BackupAppConfig } from '../../wailsjs/go/main/App'
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
  theme: 'auto',
  apiKey: '',
  baseUrl: 'https://api.openai.com/v1',
  model: 'gpt-4o-mini',
  temperature: 0.7,
  systemPrompt: `你是TagMatrix系统的全局智能助手，精通数据处理、标签规则配置和SQLite编写。

TagMatrix操作指南：
1.数据管理与SQL控制台:
底层使用SQLite数据库。原始导入数据在raw_data_records表的data字段(JSON格式文本)，查询时务必使用json_extract函数(或->/->>操作符)。根据用户需求生成准确的查询SQL。
2.标签规则引擎语法:
用于特征提取或打标，生成JSON规范规则。支持嵌套，逻辑节点{"and":[...]}、{"or":[...]}或非短路节点{"evaluate_all":[...]}。条件节点须含field(待匹配字段)、operator(操作符)、value(目标值)，可选"ignore_case":true。
支持的操作符(必须严格遵守):equals,not_equals,contains,not_contains,starts_with,ends_with,greater_than,less_than,greater_than_or_equal,less_than_or_equal,in(value为数组),not_in,is_null,is_not_null,regex,like,exists,cidr,list_contains。
新增频次与副作用算子:count_contains(统计子串出现次数),count_regex(统计正则命中次数),row_inc(当前行计数+N),global_inc(全局计数+N)。
示例:用户需求设备为honeypot且os含linux，规则为:{"and":[{"field":"device_type","operator":"equals","value":"honeypot"},{"field":"os","operator":"contains","value":"linux"}]}
3.页面上下文感知:
若问题带有指代词(如"这个页面")，请结合系统注入的当前页面环境信息解答；若提问显然与当前页面无关，请直接忽略上下文提示。

回答原则：
1.直入主题：先给代码/规则结果，再解析，不长篇大论。
2.格式规范：SQL/正则/JSON/代码等必用Markdown代码块包裹。涉及界面操作用有序列表。`,
  taskNotification: true,
  concurrency: 5,
  retries: 3,
  debugMode: false,
  developerMode: false,
  proxyMode: 'system',
  proxyUrl: '',
  w1: 1000,
  w2: 10,
  w3: 10,
  w4: 100,
  allowAiArbiter: false
}

const form = reactive({ ...defaultForm })
const isTesting = ref(false)
const testSuccess = ref(false)

watch(() => [form.apiKey, form.baseUrl, form.model], () => {
  testSuccess.value = false
})

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
      form.theme = cfg.system.theme || 'auto'
      form.taskNotification = cfg.system.task_notification
    }
    if (cfg && cfg.adv) {
      form.concurrency = cfg.adv.concurrency || 5
      form.retries = cfg.adv.retries || 3
      form.debugMode = cfg.adv.debug_mode
      form.developerMode = cfg.adv.developer_mode
    }
    if (cfg && cfg.network) {
      form.proxyMode = cfg.network.proxy_mode || 'system'
      form.proxyUrl = cfg.network.proxy_url || ''
    }
    if (cfg && (cfg as any).mdct) {
      const mdct = (cfg as any).mdct
      if (mdct.w1 === 0 && mdct.w2 === 0 && mdct.w3 === 0 && mdct.w4 === 0) {
        form.w1 = 1000
        form.w2 = 10
        form.w3 = 10
        form.w4 = 100
      } else {
        form.w1 = mdct.w1
        form.w2 = mdct.w2
        form.w3 = mdct.w3
        form.w4 = mdct.w4
      }
      form.allowAiArbiter = !!mdct.allow_ai_arbiter && !!form.apiKey
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
    testSuccess.value = true
  } catch (err: any) {
    ElMessage.error('连接失败: ' + (err.message || err))
    testSuccess.value = false
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
  ElMessageBox.confirm(
    '此操作将恢复所有设置为初始状态，并清除您当前的 API 密钥。为防止数据丢失，系统会自动将您当前的配置备份在本地目录下。\n\n是否确认重置？',
    '重置默认配置',
    {
      confirmButtonText: '确认重置',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      // 1. 调用后端备份当前配置
      const backupName = await BackupAppConfig()
      
      // 2. 覆盖前端表单为默认值并保存
      Object.assign(form, defaultForm)
      await saveSettings()
      
      // 3. 提示成功并告知备份位置
      ElMessage({
        message: `已恢复系统默认配置，原配置已备份为: ${backupName}`,
        type: 'success',
        duration: 5000,
        showClose: true
      })
    } catch (e: any) {
      ElMessage.error('重置失败: ' + (e.message || e))
    }
  }).catch(() => {})
}

const saveSettings = async () => {
  try {
    // 安全校验：如果保存时没有 API Key，强制关闭 AI 裁决开关
    if (!form.apiKey && form.allowAiArbiter) {
      form.allowAiArbiter = false
    }

    const newCfg = new config.AppConfig()
    newCfg.ai = new config.AIConfig()
    newCfg.ai.api_key = form.apiKey
    newCfg.ai.base_url = form.baseUrl
    newCfg.ai.model = form.model
    newCfg.ai.temperature = form.temperature
    newCfg.ai.system_prompt = form.systemPrompt

    newCfg.system = new config.SystemConfig()
    newCfg.system.theme = form.theme
    newCfg.system.task_notification = form.taskNotification

    newCfg.adv = new config.AdvConfig()
    newCfg.adv.concurrency = form.concurrency
    newCfg.adv.retries = form.retries
    newCfg.adv.debug_mode = form.debugMode
    newCfg.adv.developer_mode = form.developerMode

    newCfg.network = new config.NetworkConfig()
    newCfg.network.proxy_mode = form.proxyMode
    newCfg.network.proxy_url = form.proxyUrl

    ;(newCfg as any).mdct = {
      w1: form.w1,
      w2: form.w2,
      w3: form.w3,
      w4: form.w4,
      allow_ai_arbiter: form.allowAiArbiter
    }

    await SaveAppConfig(newCfg)
    ElMessage.success('设置已保存')
    
    // 派发事件，通知 App.vue 更新主题
    window.dispatchEvent(new CustomEvent('theme-changed', { detail: form.theme }))

    emit('saved')
    // 移除 dialogVisible.value = false
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
  height: 100%;
  flex: 1;
  :deep(.el-tabs__header.is-left) {
    margin-right: 0;
    width: 130px;
    min-width: 130px; /* 锁定宽度 */
    flex-shrink: 0; /* 绝对禁止缩小，防止出现滚动条时左侧导航被挤压导致“往左靠” */
    background-color: var(--tm-bg-subtle);
    padding-top: 10px;
  }
  :deep(.el-tabs__item) {
    text-align: left;
    padding: 0 16px;
    justify-content: flex-start;
  }
  :deep(.el-tabs__content) {
    padding: 20px 24px 40px;
    height: 100%;
    box-sizing: border-box;
    overflow-y: auto; /* 只有内容超出时才显示滚动条，避免空轨道 */
    overflow-x: hidden;

    /* 全局统一美化设置区域的滚动条，使用细长轻量的样式替代系统默认的厚重滚动条 */
    &::-webkit-scrollbar {
      width: 6px;
    }
    &::-webkit-scrollbar-thumb {
      background-color: #c0c4cc;
      border-radius: 4px;
    }
    &::-webkit-scrollbar-thumb:hover {
      background-color: #909399;
    }
    &::-webkit-scrollbar-track {
      background-color: transparent;
    }
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

.mode-radio-group {
  display: flex;
  width: 100%;
  gap: 0;
  
  .theme-radio {
    flex: 1;
    margin-right: 0;
    text-align: center;
    display: flex;
    justify-content: center;
    align-items: center;

    &:not(:last-child) {
      border-right: none;
      border-top-right-radius: 0;
      border-bottom-right-radius: 0;
    }
    &:not(:first-child) {
      border-top-left-radius: 0;
      border-bottom-left-radius: 0;
    }

    :deep(.el-radio__input) {
      margin-top: 1px;
    }
    :deep(.el-radio__label) {
      padding-left: 6px;
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