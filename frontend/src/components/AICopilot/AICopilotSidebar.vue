<template>
  <transition name="slide-fade">
    <div v-show="aiStore.isOpen" class="ai-sidebar">
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
import { ChatWithAIStream } from '../../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

const aiStore = useAIStore()
const isGenerating = ref(false)

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

onMounted(() => {
  aiStore.initSessions()
})

onUnmounted(() => {
  removeEventHandlers()
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
    const lastUserMsgIdx = payloadMsgs.findLastIndex(m => m.role === 'user')
    if (lastUserMsgIdx !== -1) {
      // 防止上下文过长导致 Token 浪费，截断处理
      let ctxString = aiStore.pageContext
      if (ctxString.length > 1000) {
        ctxString = ctxString.substring(0, 1000) + '...[上下文过长已截断]'
      }
      payloadMsgs[lastUserMsgIdx].content = `[系统注入：用户当前停留在【${ctxString}】页面。注意：这仅作背景参考，如果用户的最新提问与该页面功能无关，请务必忽略此提示，不要生搬硬套。]\n\n${payloadMsgs[lastUserMsgIdx].content}`
    }
  }

  // 4. 调用后端流式接口
  try {
    // 强制转换为 any 以兼容 TS，因为我们在 app.go 修改了参数但前端的绑定位未重新生成
    // 原生 wails 绑定会限制参数数量 (因为 JS 代理层中只有一个 arg)，可以直接使用 window.go 对象绕过这个限制
    await (window as any).go.main.App.ChatWithAIStream(reqId, JSON.stringify(payloadMsgs))
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
  width: 400px;
  flex-shrink: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--tm-bg-main);
  border-left: 1px solid var(--tm-border-color);
  box-shadow: -4px 0 16px rgba(0, 0, 0, 0.05);
  z-index: 50;
  position: relative;
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
