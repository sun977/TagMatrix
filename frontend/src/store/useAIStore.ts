import { defineStore } from 'pinia'
import { GetAppConfig } from '../../wailsjs/go/main/App'

export interface ChatMessage {
  role: 'user' | 'ai'
  content: string
}

export interface ChatSession {
  id: string
  title: string
  updatedAt: number
  messages: ChatMessage[]
}

export const useAIStore = defineStore('aiStore', {
  state: () => ({
    isOpen: false,
    isContextAwareness: true,
    currentModel: 'gpt-4o',
    hasAPIKey: true,
    sessions: [] as ChatSession[],
    currentSessionId: '' as string,
    pageContext: '', // Current page context string
    pendingSQL: '', // SQL pending to be executed
  }),
  getters: {
    currentChatHistory(state): ChatMessage[] {
      if (!state.currentSessionId) return []
      const session = state.sessions.find(s => s.id === state.currentSessionId)
      return session ? session.messages : []
    }
  },
  actions: {
    initSessions() {
      const saved = localStorage.getItem('ai_sessions')
      if (saved) {
        try {
          this.sessions = JSON.parse(saved)
        } catch (e) {}
      }
      if (this.sessions.length > 0) {
        this.currentSessionId = this.sessions[0].id
      }
    },
    saveToLocal() {
      localStorage.setItem('ai_sessions', JSON.stringify(this.sessions))
    },
    async checkAPIKey() {
      try {
        const config = await GetAppConfig()
        this.hasAPIKey = !!(config?.ai?.api_key && config.ai.api_key.trim().length > 0)
        if (config?.ai?.model) {
          this.currentModel = config.ai.model
        }
      } catch (e) {
        console.error('Failed to check API Key', e)
      }
    },
    async togglePanel() {
      this.isOpen = !this.isOpen
      if (this.isOpen) {
        await this.checkAPIKey()
      }
    },
    async openPanel() {
      this.isOpen = true
      await this.checkAPIKey()
    },
    closePanel() {
      this.isOpen = false
    },
    addMessage(msg: ChatMessage) {
      if (!this.currentSessionId) {
        const newId = Date.now().toString()
        this.currentSessionId = newId
        this.sessions.unshift({
          id: newId,
          title: msg.content.substring(0, 20) + (msg.content.length > 20 ? '...' : ''),
          updatedAt: Date.now(),
          messages: []
        })
      }
      const session = this.sessions.find(s => s.id === this.currentSessionId)
      if (session) {
        session.messages.push(msg)
        session.updatedAt = Date.now()
        this.sessions.sort((a, b) => b.updatedAt - a.updatedAt)
        this.saveToLocal()
      }
    },
    updateLastMessage(content: string) {
      const session = this.sessions.find(s => s.id === this.currentSessionId)
      if (session && session.messages.length > 0) {
        session.messages[session.messages.length - 1].content = content
        this.saveToLocal()
      }
    },
    clearHistory() {
      // 这里的 clearHistory 语义变为：开启一个空的新对话，直到用户发第一条消息才真正创建 session
      this.currentSessionId = ''
    },
    switchSession(id: string) {
      this.currentSessionId = id
    },
    deleteSession(id: string) {
      this.sessions = this.sessions.filter(s => s.id !== id)
      if (this.currentSessionId === id) {
        this.currentSessionId = this.sessions.length > 0 ? this.sessions[0].id : ''
      }
      this.saveToLocal()
    }
  }
})
