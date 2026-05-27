<template>
  <transition name="slide-fade">
    <div 
      v-show="aiStore.isOpen" 
      class="ai-sidebar"
      :class="{ 'no-transition': aiStore.isDragging }"
      :style="{ width: aiStore.sidebarWidth + 'px' }"
    >
      <!-- 拖拽调节宽度的把手 -->
      <div class="sidebar-resizer" @mousedown="startDrag"></div>
      
      <AICopilotHeader />
      <AICopilotChat @preset-click="handleSend" />
      <AICopilotInput @send="handleSend" :disabled="isGenerating" />
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useAIStore } from '../../store/useAIStore'
import AICopilotHeader from './AICopilotHeader.vue'
import AICopilotChat from './AICopilotChat.vue'
import AICopilotInput from './AICopilotInput.vue'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

const aiStore = useAIStore()
const isGenerating = ref(false)

// --- 侧边栏拖拽调节宽度逻辑 ---

const minWidth = 300
// 计算最大宽度为屏幕宽度的一半
const getMaxWidth = () => window.innerWidth / 2

const startDrag = (e: MouseEvent) => {
  aiStore.isDragging = true
  document.body.style.cursor = 'col-resize'
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  e.preventDefault()
}

const onDrag = (e: MouseEvent) => {
  if (!aiStore.isDragging) return
  // 由于侧边栏在右侧，向左拖拽 x 坐标变小，宽度增加
  const newWidth = window.innerWidth - e.clientX
  const maxWidth = getMaxWidth()
  
  if (newWidth < minWidth) {
    aiStore.sidebarWidth = minWidth
  } else if (newWidth > maxWidth) {
    aiStore.sidebarWidth = maxWidth
  } else {
    aiStore.sidebarWidth = newWidth
  }
}

const stopDrag = () => {
  if (aiStore.isDragging) {
    aiStore.isDragging = false
    document.body.style.cursor = ''
    document.removeEventListener('mousemove', onDrag)
    document.removeEventListener('mouseup', stopDrag)
  }
}

let aiMessageIndex = -1
let currentGeneratingContent = ''
let currentReqId = ''

const setupEventHandlers = (reqId: string) => {
  // 移除旧的监听
  removeEventHandlers()
  currentReqId = reqId

  EventsOn(`ai_chat_chunk_${reqId}`, (chunk: string) => {
    currentGeneratingContent += chunk
    aiStore.updateLastMessage(currentGeneratingContent)
  })

  EventsOn(`ai_chat_end_${reqId}`, () => {
    isGenerating.value = false
    aiMessageIndex = -1
    aiStore.saveToLocal()
    removeEventHandlers()
  })

  EventsOn(`ai_chat_error_${reqId}`, (err: string) => {
    currentGeneratingContent += `\n\n[请求出错: ${err}]`
    aiStore.updateLastMessage(currentGeneratingContent)
    isGenerating.value = false
    aiMessageIndex = -1
    aiStore.saveToLocal()
    removeEventHandlers()
  })
}

const removeEventHandlers = () => {
  if (currentReqId) {
    EventsOff(`ai_chat_chunk_${currentReqId}`)
    EventsOff(`ai_chat_end_${currentReqId}`)
    EventsOff(`ai_chat_error_${currentReqId}`)
    currentReqId = ''
  }
}

const globalSendHandler = (e: Event) => {
  const ce = e as CustomEvent
  if (ce.detail) {
    if (!aiStore.isOpen) {
      aiStore.openPanel()
    }
    handleSend(ce.detail)
  }
}

onMounted(() => {
  aiStore.initSessions()
  window.addEventListener('ai-trigger-send', globalSendHandler)
})

onUnmounted(() => {
  removeEventHandlers()
  stopDrag()
  window.removeEventListener('ai-trigger-send', globalSendHandler)
})

const handleSend = async (text: string) => {
  if (!text.trim() || isGenerating.value) return
  
  // 1. 添加用户消息
  aiStore.addMessage({
    role: 'user',
    content: text
  })
  
  // 2. 添加空的 AI 回复占位
  aiStore.addMessage({
    role: 'ai',
    content: ''
  })
  aiMessageIndex = aiStore.currentChatHistory.length - 1
  currentGeneratingContent = ''
  isGenerating.value = true

  // 生成请求ID，用于隔离事件
  const reqId = Date.now().toString() + '_' + Math.random().toString(36).substring(2, 9)
  setupEventHandlers(reqId)

  // 3. 构建发给后端的真实 Payload (带上 Context 及全部历史)
  // 克隆所有历史，防止直接修改 store
  const payloadMsgs = aiStore.currentChatHistory.slice(0, aiMessageIndex).map(m => ({
    role: m.role,
    content: m.content
  }))

  // 加上页面 Context (如果开启了且有)
  if (aiStore.isContextAwareness && aiStore.pageContext && payloadMsgs.length > 0) {
    const reversedMsgs = [...payloadMsgs].reverse()
    const lastUserMsgIdxReversed = reversedMsgs.findIndex((m: any) => m.role === 'user')
    const lastUserMsgIdx = lastUserMsgIdxReversed >= 0 ? payloadMsgs.length - 1 - lastUserMsgIdxReversed : -1
    if (lastUserMsgIdx !== -1) {
      // 防止上下文过长导致 Token 浪费，截断处理
      let ctxString = aiStore.pageContext
      if (ctxString.length > 1000) {
        ctxString = ctxString.substring(0, 1000) + '...[上下文过长已截断]'
      }
      payloadMsgs[lastUserMsgIdx].content = `[当前页面：${ctxString}。若提问与此无关请忽略]\n\n${payloadMsgs[lastUserMsgIdx].content}`
    }
  }

  const payloadObj = {
    messages: payloadMsgs,
    is_agent: aiStore.isAgentMode
  }

  // 4. 调用后端流式接口
  try {
    // 强制转换为 any 以兼容 TS，因为我们在 app.go 修改了参数但前端的绑定位未重新生成
    // 原生 wails 绑定会限制参数数量 (因为 JS 代理层中只有一个 arg)，可以直接使用 window.go 对象绕过这个限制
    await (window as any).go.main.App.ChatWithAIStream(reqId, JSON.stringify(payloadObj))
  } catch (e: any) {
    currentGeneratingContent = `[系统错误: ${e.message || e}]`
    aiStore.updateLastMessage(currentGeneratingContent)
    isGenerating.value = false
    aiMessageIndex = -1
    aiStore.saveToLocal()
    removeEventHandlers()
  }
}
</script>

<style scoped lang="scss">
.ai-sidebar {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--tm-bg-main);
  border-left: 1px solid var(--tm-border-color);
  box-shadow: -4px 0 16px rgba(0, 0, 0, 0.05);
  z-index: 50;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  will-change: transform, width;
  overflow-x: hidden;
  
  &.no-transition {
    transition: none !important;
  }
  
  /* --- 拖拽调整宽度的把手 --- */
  .sidebar-resizer {
    position: absolute;
    top: 0;
    left: -3px; // 悬浮在左侧边框线上
    width: 6px;
    height: 100%;
    cursor: col-resize;
    z-index: 10;
    transition: background-color 0.2s;

    &:hover, &:active {
      background-color: var(--tm-accent-primary);
      opacity: 0.5;
    }
  }
}

html.dark {
  .ai-sidebar {
    box-shadow: -4px 0 16px rgba(0, 0, 0, 0.3);
  }
}

/* 侧边栏展开收起动画 */
.slide-fade-enter-active,
.slide-fade-leave-active {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.3s;
}
.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
