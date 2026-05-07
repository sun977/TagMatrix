<template>
  <div class="ai-input-wrapper">
    <div class="input-container" :class="{ 'is-disabled': disabled }">
      <textarea
        ref="textareaRef"
        v-model="inputText"
        class="auto-resize-textarea"
        placeholder="发送消息... (Enter 发送, Shift+Enter 换行, / 唤出快捷指令)"
        rows="1"
        :disabled="disabled"
        @input="autoResize"
        @keydown.enter="handleEnter"
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
import { ref, nextTick } from 'vue'

const props = defineProps<{
  disabled?: boolean
}>()

const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const emit = defineEmits<{
  (e: 'send', message: string): void
}>()

const autoResize = () => {
  if (!textareaRef.value) return
  // 重置高度以获取正确的 scrollHeight
  textareaRef.value.style.height = 'auto'
  // 设置最大高度限制 (例如 150px)
  const newHeight = Math.min(textareaRef.value.scrollHeight, 150)
  textareaRef.value.style.height = `${newHeight}px`
}

const handleEnter = (e: KeyboardEvent) => {
  if (props.disabled) {
    e.preventDefault()
    return
  }
  
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

const sendMessage = () => {
  if (props.disabled) return
  
  const msg = inputText.value.trim()
  if (!msg) return
  
  emit('send', msg)
  inputText.value = ''
  
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
  }
}
</style>
