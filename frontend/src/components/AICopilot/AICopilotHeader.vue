<template>
  <div class="ai-header">
    <div class="header-left">
      <el-dropdown trigger="click" @command="handleModelChange">
        <span class="model-selector">
          {{ currentModelName }}
          <el-icon class="el-icon--right"><ArrowDown /></el-icon>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="gpt-4o">GPT-4o</el-dropdown-item>
            <el-dropdown-item command="claude-3.5-sonnet">Claude 3.5 Sonnet</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    
    <div class="header-right">
      <el-popover placement="bottom" :width="320" trigger="click" popper-class="ai-history-popover">
        <template #reference>
          <div class="icon-btn">
            <el-tooltip content="历史对话" placement="bottom">
              <el-icon><Memo /></el-icon>
            </el-tooltip>
          </div>
        </template>
        
        <div class="history-list-wrapper">
          <div class="history-list-header">
            <span>历史记录</span>
            <el-button link type="primary" size="small" @click="startNewChat">
              <el-icon><Plus /></el-icon> 新建对话
            </el-button>
          </div>
          <div v-if="aiStore.sessions.length === 0" class="no-history">
            暂无历史对话
          </div>
          <div v-else class="history-list">
            <div 
              v-for="session in aiStore.sessions" 
              :key="session.id"
              class="history-item"
              :class="{ 'is-active': session.id === aiStore.currentSessionId }"
              @click="aiStore.switchSession(session.id)"
            >
              <div class="history-title">{{ session.title }}</div>
              <div class="history-actions" @click.stop="aiStore.deleteSession(session.id)">
                <el-icon><Delete /></el-icon>
              </div>
            </div>
          </div>
        </div>
      </el-popover>

      <el-tooltip content="开启新对话" placement="bottom">
        <div class="icon-btn" @click="startNewChat">
          <el-icon><ChatRound /></el-icon>
        </div>
      </el-tooltip>
      <el-tooltip content="上下文感知" placement="bottom">
        <div 
          class="icon-btn" 
          :class="{ active: aiStore.isContextAwareness }" 
          @click="aiStore.isContextAwareness = !aiStore.isContextAwareness"
        >
          <el-icon><View v-if="aiStore.isContextAwareness" /><Hide v-else /></el-icon>
        </div>
      </el-tooltip>
      <el-tooltip content="快捷注入 System Prompt" placement="bottom">
        <div class="icon-btn">
          <el-icon><Operation /></el-icon>
        </div>
      </el-tooltip>
      <div class="icon-btn close-btn" @click="aiStore.closePanel()">
        <el-icon><Close /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAIStore } from '../../store/useAIStore'

const aiStore = useAIStore()

const currentModelName = computed(() => {
  if (aiStore.currentModel === 'gpt-4o') return 'GPT-4o'
  if (aiStore.currentModel === 'claude-3.5-sonnet') return 'Claude 3.5 Sonnet'
  return aiStore.currentModel
})

const handleModelChange = (cmd: string) => {
  aiStore.currentModel = cmd
}

const startNewChat = () => {
  aiStore.clearHistory()
}
</script>

<style lang="scss">
.ai-history-popover {
  padding: 0 !important;
}
</style>

<style scoped lang="scss">
.history-list-wrapper {
  display: flex;
  flex-direction: column;
  max-height: 400px;
  
  .history-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--tm-border-color);
    font-weight: 600;
    font-size: 14px;
    color: var(--tm-text-primary);
  }
  
  .no-history {
    padding: 32px;
    text-align: center;
    color: var(--tm-text-secondary);
    font-size: 13px;
  }
  
  .history-list {
    flex: 1;
    overflow-y: auto;
    
    .history-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 16px;
      cursor: pointer;
      border-bottom: 1px solid var(--tm-border-color);
      transition: background-color 0.2s;
      
      &:hover {
        background-color: var(--tm-bg-hover);
        .history-actions {
          opacity: 1;
        }
      }
      
      &.is-active {
        background-color: var(--tm-accent-light);
        .history-title {
          color: var(--tm-accent-primary);
          font-weight: 600;
        }
      }
      
      .history-title {
        font-size: 13px;
        color: var(--tm-text-regular);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        flex: 1;
        padding-right: 12px;
      }
      
      .history-actions {
        opacity: 0;
        color: #f56c6c;
        padding: 4px;
        border-radius: 4px;
        transition: opacity 0.2s, background-color 0.2s;
        
        &:hover {
          background-color: #fef0f0;
        }
      }
    }
  }
}

.ai-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid var(--tm-border-color);
  background-color: var(--tm-bg-sidebar);

  .header-left {
    .model-selector {
      cursor: pointer;
      display: flex;
      align-items: center;
      font-weight: 600;
      color: var(--tm-text-primary);
      font-size: 14px;
      
      &:hover {
        color: var(--tm-accent-primary);
      }
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;

    .icon-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 28px;
      height: 28px;
      border-radius: var(--tm-border-radius-sm);
      cursor: pointer;
      color: var(--tm-text-secondary);
      transition: all 0.2s;

      &:hover {
        background-color: var(--tm-bg-hover);
        color: var(--tm-text-primary);
      }

      &.active {
        color: var(--tm-accent-primary);
        background-color: var(--tm-accent-light);
      }

      &.close-btn:hover {
        color: #f56c6c;
        background-color: #fef0f0;
      }
    }
  }
}
</style>
