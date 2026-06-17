<template>
  <div class="ai-input-wrapper">
    <div class="slash-commands-popup" v-if="showCommands" :style="popupStyle">
      <div 
        v-for="(cmd, index) in commands" 
        :key="cmd.id"
        class="command-item"
        :class="{ 'is-active': selectedCommandIndex === index }"
        @click="selectCommand(cmd)"
        @mouseenter="selectedCommandIndex = index"
      >
        <div class="command-icon"><el-icon><component :is="cmd.icon" /></el-icon></div>
        <div class="command-info">
          <div class="command-name">/{{ cmd.name }}</div>
          <div class="command-desc">{{ cmd.description }}</div>
        </div>
      </div>
    </div>
    
    <div class="input-container" :class="{ 'is-disabled': disabled }">
      <textarea
        ref="textareaRef"
        v-model="inputText"
        class="auto-resize-textarea"
        placeholder="⏎发送 | ⇧⏎换行 | /快捷操作"
        rows="1"
        :readonly="disabled"
        @input="handleInput"
        @blur="handleBlur"
        @keydown="handleKeydown"
      ></textarea>
      
      <div class="action-bar">
        <div class="left-actions">
          <el-tooltip :content="aiStore.isAgentMode ? 'Agent 模式' : 'Ask 模式'" placement="top" :show-after="500">
            <div class="mode-switch-btn" :class="{ 'is-agent': aiStore.isAgentMode }" @click="aiStore.toggleAgentMode">
              <el-icon class="mode-icon">
                <svg v-if="!aiStore.isAgentMode" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2"/><path d="M20 14h2"/><path d="M15 13v2"/><path d="M9 13v2"/></svg>
              </el-icon>
              <span class="mode-text">{{ aiStore.isAgentMode ? 'Agent' : 'Ask' }}</span>
            </div>
          </el-tooltip>
          <el-tooltip content="添加附件 (规划中)" placement="top" :show-after="500">
            <div class="action-btn">
              <el-icon><Plus /></el-icon>
            </div>
          </el-tooltip>
        </div>
        <div class="right-actions">
          <el-tooltip :content="disabled ? '暂停生成' : '发送'" placement="top" :show-after="500">
            <div 
              class="send-btn" 
              :class="{ 'is-active': inputText.trim().length > 0 && !disabled, 'is-stop': disabled }" 
              @click="handleRightBtnClick"
            >
              <el-icon><Promotion v-if="!disabled" /><VideoPause v-else /></el-icon>
            </div>
          </el-tooltip>
        </div>
      </div>
    </div>
    <!-- <div class="footer-tips">
      AI 生成内容仅供参考，执行操作前请确认。
    </div> -->
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, computed } from 'vue'
import { Promotion, Loading, MagicStick, Document, Search, Plus, VideoPause } from '@element-plus/icons-vue'

import { useAIStore } from '../../store/useAIStore'

const aiStore = useAIStore()

const props = defineProps<{
  disabled?: boolean
}>()

const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const emit = defineEmits<{
  (e: 'send', message: string): void
  (e: 'stop'): void
}>()

// 快捷指令相关
const showCommands = ref(false)
const selectedCommandIndex = ref(0)
const commands = [
  { id: 'sql', name: 'sql', description: '帮我写提取 SQL', icon: MagicStick, prompt: '请帮我写一段针对原始数据的提取 SQL，需求是：' },
  { id: 'regex', name: 'regex', description: '生成正则表达式', icon: Document, prompt: '请帮我写一个正则表达式，用于提取：' },
  { id: 'rule', name: 'rule', description: '生成 JSON 规则', icon: Search, prompt: '请帮我生成一段 TagMatrix JSON 提取规则，需求是：' }
]

// 让弹出菜单在输入框上方显示
const popupStyle = computed(() => {
  return {
    bottom: `calc(100% - 20px)`
  }
})

const autoResize = () => {
  if (!textareaRef.value) return
  // 重置高度以获取正确的 scrollHeight
  textareaRef.value.style.height = 'auto'
  // 设置最大高度限制 (例如 150px)
  const newHeight = Math.min(textareaRef.value.scrollHeight, 150)
  textareaRef.value.style.height = `${newHeight}px`
}

const handleInput = () => {
  autoResize()
  
  // 检查是否应该显示快捷指令
  const val = inputText.value
  const el = textareaRef.value
  if (!el) return

  const cursorPosition = el.selectionStart
  const textBeforeCursor = val.substring(0, cursorPosition)
  
  // 匹配行首的 '/' 或者空格后的 '/'
  const match = textBeforeCursor.match(/(?:^|\s)\/([a-zA-Z]*)$/)
  
  if (match) {
    showCommands.value = true
  } else {
    showCommands.value = false
  }
}

const handleBlur = () => {
  // 延迟隐藏，以便能够点击选中
  setTimeout(() => {
    showCommands.value = false
  }, 200)
}

const selectCommand = (cmd: any) => {
  const el = textareaRef.value
  if (!el) return

  const val = inputText.value
  const cursorPosition = el.selectionStart
  const textBeforeCursor = val.substring(0, cursorPosition)
  
  // 找到最后一个 '/' 并替换为其 prompt
  const lastSlashIndex = textBeforeCursor.lastIndexOf('/')
  if (lastSlashIndex !== -1) {
    const textAfterCursor = val.substring(cursorPosition)
    inputText.value = val.substring(0, lastSlashIndex) + cmd.prompt + textAfterCursor
    
    // 重新设置光标位置
    nextTick(() => {
      const newCursorPos = lastSlashIndex + cmd.prompt.length
      el.focus()
      el.setSelectionRange(newCursorPos, newCursorPos)
      autoResize()
    })
  }
  
  showCommands.value = false
  selectedCommandIndex.value = 0
}

const handleKeydown = (e: KeyboardEvent) => {
  if (showCommands.value) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedCommandIndex.value = (selectedCommandIndex.value + 1) % commands.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedCommandIndex.value = (selectedCommandIndex.value - 1 + commands.length) % commands.length
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      selectCommand(commands[selectedCommandIndex.value])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      showCommands.value = false
      return
    }
  }

  if (props.disabled) {
    e.preventDefault()
    return
  }
  
  if (e.key === 'Enter') {
    if (e.shiftKey) {
      // Shift+Enter 换行，默认行为，需重新计算高度
      nextTick(() => {
        autoResize()
      })
    } else {
      // Enter 发送
      e.preventDefault() // 阻止默认的换行行为
      sendMessage()
    }
  }
}

const handleRightBtnClick = () => {
  if (props.disabled) {
    emit('stop')
  } else {
    sendMessage()
  }
}

const sendMessage = () => {
  if (props.disabled) return
  
  const msg = inputText.value.trim()
  if (!msg) return
  
  emit('send', msg)
  inputText.value = ''
  showCommands.value = false
  
  // 恢复输入框高度
  nextTick(() => {
    if (textareaRef.value) {
      textareaRef.value.style.height = 'auto'
    }
  })
}
</script>

<style scoped lang="scss">
.ai-input-wrapper {
  padding: 10px 12px 12px;
  background-color: var(--tm-bg-sidebar);
  border-top: 1px solid var(--tm-border-color);
  position: relative;

  .slash-commands-popup {
    position: absolute;
    left: 12px;
    right: 12px;
    background-color: var(--tm-bg-main);
    border: 1px solid var(--tm-border-color);
    border-radius: var(--tm-border-radius);
    box-shadow: var(--tm-shadow-md);
    overflow: hidden;
    z-index: 100;
    
    .command-item {
      display: flex;
      align-items: center;
      padding: 10px 12px;
      cursor: pointer;
      transition: background-color 0.2s;
      border-bottom: 1px solid var(--tm-border-color);
      
      &:last-child {
        border-bottom: none;
      }
      
      &.is-active, &:hover {
        background-color: var(--tm-bg-hover);
      }
      
      .command-icon {
        font-size: 16px;
        color: var(--tm-text-secondary);
        margin-right: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;
        background-color: var(--tm-bg-subtle);
        border-radius: 4px;
      }
      
      .command-info {
        display: flex;
        flex-direction: column;
        
        .command-name {
          font-weight: 600;
          font-size: 13px;
          color: var(--tm-text-primary);
          line-height: 1.2;
        }
        
        .command-desc {
          font-size: 12px;
          color: var(--tm-text-secondary);
          margin-top: 4px;
        }
      }
    }
  }

  .input-container {
    position: relative;
    display: flex;
    flex-direction: column;
    background-color: var(--tm-bg-main);
    border: 1px solid var(--tm-border-color);
    border-radius: 16px;
    padding: 10px 12px;
    transition: border-color 0.2s;

    &:focus-within {
      border-color: var(--tm-accent-primary);
    }

    &.is-disabled {
      background-color: var(--tm-bg-hover);
      
      .auto-resize-textarea {
        opacity: 0.7;
        pointer-events: none;
      }
      
      .left-actions {
        opacity: 0.5;
        pointer-events: none;
      }
    }

    .auto-resize-textarea {
      width: 100%;
      border: none;
      outline: none;
      resize: none;
      background: transparent;
      color: var(--tm-text-primary);
      font-family: inherit;
      font-size: 14px;
      line-height: 1.5;
      max-height: 150px;
      overflow-y: auto;
      
      &::placeholder {
        color: var(--tm-text-secondary);
      }
      
      /* 隐藏滚动条但保留滚动功能 */
      &::-webkit-scrollbar {
        width: 4px;
      }
      &::-webkit-scrollbar-thumb {
        background: var(--tm-border-color);
        border-radius: 2px;
      }
    }

    .action-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;
      margin-top: 8px;

      .left-actions {
        display: flex;
        gap: 8px;
        align-items: center;

        .mode-switch-btn {
          display: flex;
          align-items: center;
          gap: 4px;
          padding: 4px 10px;
          border-radius: 14px;
          font-size: 13px;
          font-weight: 500;
          color: var(--tm-text-secondary);
          cursor: pointer;
          transition: all 0.2s;

          &:hover {
            background-color: var(--tm-bg-hover);
            color: var(--tm-text-primary);
          }

          &.is-agent {
            color: var(--tm-accent-primary);
            background-color: var(--tm-bg-subtle);
          }

          .mode-icon {
            font-size: 15px;
          }
        }

        .action-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 28px;
          border-radius: 50%;
          color: var(--tm-text-secondary);
          cursor: pointer;
          transition: background-color 0.2s, color 0.2s;

          &:hover {
            background-color: var(--tm-bg-hover);
            color: var(--tm-text-primary);
          }

          &.is-active {
            color: var(--tm-accent-primary);
            background-color: var(--tm-bg-subtle);
          }
        }
      }

      .right-actions {
        display: flex;

        .send-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 32px;
          height: 32px;
          border-radius: 50%;
          color: var(--tm-text-secondary);
          background-color: var(--tm-bg-hover);
          cursor: not-allowed;
          transition: all 0.2s;

          &.is-stop {
            color: var(--tm-text-primary);
            background-color: transparent;
            border: 1px solid var(--tm-border-color);
            cursor: pointer;
            &:hover {
              color: #f56c6c;
              border-color: #f56c6c;
              background-color: rgba(245, 108, 108, 0.1);
            }
          }

          &.is-active {
            color: var(--tm-bg-main);
            background-color: var(--tm-text-primary);
            cursor: pointer;

            &:hover {
              opacity: 0.8;
            }
          }
        }
      }
    }
  }

  .footer-tips {
    text-align: center;
    font-size: 12px;
    color: var(--tm-text-secondary);
    margin-top: 8px;
    opacity: 0.7;
  }
}

html.dark {
  .ai-input-wrapper {
    .input-container {
      background-color: var(--tm-bg-subtle);
    }
    .slash-commands-popup {
      background-color: var(--tm-bg-subtle);
    }
  }
}
</style>
