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
        placeholder="发送消息... (Enter 发送, Shift+Enter 换行, / 唤出快捷指令)"
        rows="1"
        :disabled="disabled"
        @input="handleInput"
        @blur="handleBlur"
        @keydown="handleKeydown"
      ></textarea>
      
      <div 
        class="send-btn" 
        :class="{ 'is-active': inputText.trim().length > 0 && !disabled }" 
        @click="sendMessage"
      >
        <el-icon><Promotion v-if="!disabled" /><Loading v-else class="is-loading" /></el-icon>
      </div>
    </div>
    <div class="footer-tips">
      AI 生成内容仅供参考，执行操作前请确认。
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, computed } from 'vue'
import { Promotion, Loading, MagicStick, Document, Search } from '@element-plus/icons-vue'

const props = defineProps<{
  disabled?: boolean
}>()

const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const emit = defineEmits<{
  (e: 'send', message: string): void
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
  padding: 16px;
  background-color: var(--tm-bg-sidebar);
  border-top: 1px solid var(--tm-border-color);
  position: relative;

  .slash-commands-popup {
    position: absolute;
    left: 16px;
    right: 16px;
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
    align-items: flex-end;
    background-color: var(--tm-bg-main);
    border: 1px solid var(--tm-border-color);
    border-radius: var(--tm-border-radius);
    padding: 8px 12px;
    transition: border-color 0.2s;

    &:focus-within {
      border-color: var(--tm-accent-primary);
    }

    &.is-disabled {
      opacity: 0.7;
      background-color: var(--tm-bg-hover);
      pointer-events: none;
    }

    .auto-resize-textarea {
      flex: 1;
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
      padding-right: 40px; // 为发送按钮留出空间
      
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

    .send-btn {
      position: absolute;
      right: 8px;
      bottom: 8px;
      display: flex;
      align-items: center;
      justify-content: center;
      width: 32px;
      height: 32px;
      border-radius: var(--tm-border-radius-sm);
      color: var(--tm-text-secondary);
      background-color: var(--tm-bg-hover);
      cursor: not-allowed;
      transition: all 0.2s;

      &.is-active {
        color: #ffffff;
        background-color: var(--tm-accent-primary);
        cursor: pointer;

        &:hover {
          background-color: var(--tm-accent-hover);
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
