<template>
  <div class="ai-chat-container" ref="chatContainerRef">
    <div v-if="!aiStore.hasAPIKey" class="empty-state api-key-missing">
      <div class="empty-icon">
        <el-icon :size="48"><Warning /></el-icon>
      </div>
      <h3 class="empty-title">未配置 AI 引擎</h3>
      <p class="empty-desc">
        请先在全局设置中配置 OpenAI 或兼容大模型的 API Key，即可开启您的AI智能助手。
      </p>
      <el-button type="primary" @click="openSettings" color="var(--tm-accent-primary)">立即配置 AI 引擎</el-button>
    </div>

    <div v-else-if="aiStore.currentChatHistory.length === 0" class="empty-state">
      <div class="empty-icon">
        <el-icon :size="48"><Service /></el-icon>
      </div>
      <h3 class="empty-title">AI Copilot</h3>
      <p class="empty-desc">
        我是您的AI智能助手。您可以随时向我提问，或者让我帮您提取标签规则、编写 SQL 及正则表达式。
      </p>
      
      <!-- 预设问题 (快捷气泡) -->
      <div class="preset-questions">
        <div class="preset-item" @click="emitPreset('帮我写一个提取 JSON 中 name 字段的 SQL')">
          <el-icon><MagicStick /></el-icon>
          <span>帮我写一个提取 JSON 中 name 字段的 SQL</span>
        </div>
        <div class="preset-item" @click="emitPreset('如何配置正则规则来提取手机号？')">
          <el-icon><Document /></el-icon>
          <span>如何配置正则规则来提取手机号？</span>
        </div>
      </div>
    </div>

    <div v-else class="chat-list">
      <div 
        v-for="(msg, index) in aiStore.currentChatHistory" 
        :key="index"
        class="chat-message"
        :class="msg.role === 'user' ? 'is-user' : 'is-ai'"
      >
        <div class="message-avatar">
          <el-icon v-if="msg.role === 'user'"><User /></el-icon>
          <el-icon v-else><Service /></el-icon>
        </div>
        <div class="message-content">
          <div class="message-bubble markdown-body" v-if="msg.role === 'ai'" v-html="renderMarkdown(msg.content)" @click="handleBubbleClick">
          </div>
          <div class="message-bubble" v-else>
            {{ msg.content }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAIStore } from '../../store/useAIStore'
import { marked } from 'marked'
import hljs from 'highlight.js'
import { ElMessage, ElMessageBox } from 'element-plus'
import 'highlight.js/styles/atom-one-dark.css' // Import highlight.js theme

const router = useRouter()
const aiStore = useAIStore()
const chatContainerRef = ref<HTMLElement | null>(null)

const renderer = new marked.Renderer()
const originalCode = renderer.code.bind(renderer)

renderer.code = (token: any) => {
  const code = token.text
  const language = token.lang || 'plaintext'
  
  let highlightedCode = code
  if (language && hljs.getLanguage(language)) {
    try {
      highlightedCode = hljs.highlight(code, { language }).value
    } catch (e) {}
  } else {
    try {
      highlightedCode = hljs.highlightAuto(code).value
    } catch (e) {}
  }

  const encodedCode = encodeURIComponent(code)
  
  return `<div class="code-block-wrapper">
      <div class="code-block-header">
        <span class="code-lang">${language}</span>
        <button class="copy-btn" data-code="${encodedCode}">复制</button>
      </div>
      <pre><code class="hljs language-${language}">${highlightedCode}</code></pre>
    </div>`
}

marked.setOptions({
  breaks: true,
})

const renderMarkdown = (text: string) => {
  if (!text) return ''
  
  // 拦截 <action> 标签并转换为前端按钮组件 (使用原生 HTML 因为在 v-html 内部)
  let processedText = text.replace(
    /<action\s+type=(['"])(.*?)\1(?:\s+query=(['"])([\s\S]*?)\3)?(?:\s+label=(['"])(.*?)\5)?\s*\/>/g,
    (match, _q1, type, _q2, query, _q3, label) => {
      const btnLabel = label || '执行操作'
      
      // 处理转义字符，特别是单双引号的 HTML 实体
      let decodedQuery = query || ''
      decodedQuery = decodedQuery
        .replace(/&quot;/g, '"')
        .replace(/&#39;/g, "'")
        .replace(/&lt;/g, '<')
        .replace(/&gt;/g, '>')
        .replace(/&amp;/g, '&')
        .replace(/\\n/g, '\n')
        .replace(/\\r/g, '\r')
        .replace(/\\t/g, '\t')
        
      // 兼容大模型有时会用两个单引号转义一个单引号的行为
      if (_q3 === "'") {
        decodedQuery = decodedQuery.replace(/''/g, "'")
      }
        
      const safeQuery = encodeURIComponent(decodedQuery)
      
      return `<button class="action-btn" data-type="${type}" data-query="${safeQuery}"><span class="action-icon">🚀</span><span class="action-text">${btnLabel}</span></button>`
    }
  )

  return marked.parse(processedText, { renderer })
}

const handleBubbleClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  
  // 复制按钮逻辑
  if (target.classList.contains('copy-btn')) {
    const codeEncoded = target.getAttribute('data-code')
    if (codeEncoded) {
      const code = decodeURIComponent(codeEncoded)
      navigator.clipboard.writeText(code).then(() => {
        ElMessage.success('已复制到剪贴板')
        target.innerText = '已复制!'
        setTimeout(() => {
          target.innerText = '复制'
        }, 2000)
      }).catch(() => {
        ElMessage.error('复制失败')
      })
    }
    return
  }

  // Action 动作按钮逻辑
  const actionBtn = target.closest('.action-btn') as HTMLElement
  if (actionBtn) {
    const type = actionBtn.getAttribute('data-type')
    const query = decodeURIComponent(actionBtn.getAttribute('data-query') || '')

    if (type === 'execute_sql') {
      ElMessageBox.confirm(
        '该操作将直接携带该 SQL 语句跳转至控制台。在真正执行前您仍有确认修改的机会。',
        '操作确认',
        {
          confirmButtonText: '前往控制台',
          cancelButtonText: '取消',
          type: 'info',
        }
      ).then(() => {
        aiStore.pendingSQL = query
        router.push('/database-admin')
        ElMessage.success('已加载 SQL，请确认后手动执行')
      }).catch(() => {
        // 用户取消
      })
    } else if (type === 'delete_tag') {
      ElMessageBox.confirm(
        '此操作将永久删除该标签及其所有子标签，且会级联删除关联的配置规则。确认删除？',
        '高危操作确认',
        {
          confirmButtonText: '确认删除',
          cancelButtonText: '取消',
          type: 'warning',
        }
      ).then(async () => {
        try {
          const tagPath = query // 这里我们把 tag_path 放在 query 属性里传递过来
          if (!tagPath) {
             ElMessage.error('找不到标签路径')
             return
          }
          // 因为 DeleteTag 接口需要 ID，我们需要先通过 App 获取 Tag ID
          const tag = await (window as any).go.main.App.GetTagByPath(tagPath)
          if (!tag) {
             ElMessage.error('找不到该标签')
             return
          }
          await (window as any).go.main.App.DeleteTag(tag.id)
          ElMessage.success('标签删除成功')
          window.dispatchEvent(new CustomEvent('tag_tree_updated'))
          // 因为目前在 sidebar 里面可能监听不到 events，可以通过全局方式
          // 更好的方式是直接抛出一个刷新事件
          import('../../../wailsjs/runtime/runtime').then((runtime) => {
             runtime.EventsEmit('tag_tree_updated')
          })
        } catch (error: any) {
          ElMessage.error('删除失败: ' + error)
        }
      }).catch(() => {
        // 用户取消
      })
    }
  }
}

const emit = defineEmits<{
  (e: 'preset-click', text: string): void
}>()

const emitPreset = (text: string) => {
  emit('preset-click', text)
}

const openSettings = () => {
  window.dispatchEvent(new CustomEvent('open-settings'))
}

watch(() => aiStore.currentChatHistory, () => {
  nextTick(() => {
    if (chatContainerRef.value) {
      chatContainerRef.value.scrollTop = chatContainerRef.value.scrollHeight
    }
  })
}, { deep: true })
</script>

<style lang="scss">
/* Global styles for markdown rendering within chat */
.markdown-body {
  word-break: break-word;
  line-height: 1.6;
  
  p { margin: 0 0 12px; }
  p:last-child { margin-bottom: 0; }
  
  code {
    background-color: rgba(175, 184, 193, 0.2);
    padding: 0.2em 0.4em;
    border-radius: 4px;
    font-size: 85%;
    font-family: ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, Liberation Mono, monospace;
  }
  
  pre {
    margin: 16px 0;
    padding: 0;
    background-color: transparent;
    border-radius: 6px;
    overflow: hidden;
    
    code {
      display: block;
      padding: 16px;
      overflow-x: auto;
      background-color: #282c34; // match atom-one-dark
      color: #abb2bf;
      border-radius: 0 0 6px 6px;
    }
  }
  
  .code-block-wrapper {
    margin: 16px 0;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid var(--tm-border-color);
    
    pre {
      margin: 0;
      border-radius: 0;
    }
    
    .code-block-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 16px;
      background-color: #1e1e1e;
      color: #9cdcfe;
      font-size: 12px;
      font-family: ui-monospace, SFMono-Regular, monospace;
      border-bottom: 1px solid #333;
      
      .copy-btn {
        background: transparent;
        border: 1px solid #555;
        color: #ccc;
        border-radius: 4px;
        padding: 2px 8px;
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s;
        
        &:hover {
          background-color: #333;
          color: white;
          border-color: #777;
        }
      }
    }
  }

  .action-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin: 12px 0;
    padding: 8px 16px;
    background-color: var(--tm-accent-light);
    color: var(--tm-accent-primary);
    border: 1px solid var(--tm-accent-primary);
    border-radius: var(--tm-border-radius-sm);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      background-color: var(--tm-accent-primary);
      color: white;
    }

    .action-icon {
      font-size: 14px;
    }
  }
}

html.dark {
  .markdown-body {
    code {
      background-color: rgba(255, 255, 255, 0.1);
    }
    
    .code-block-wrapper {
      border-color: #333;
    }
  }
}
</style>

<style scoped lang="scss">
.ai-chat-container {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  background-color: var(--tm-bg-main);

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--tm-text-secondary);
    text-align: center;

    .empty-icon {
      color: var(--tm-accent-primary);
      margin-bottom: 16px;
      opacity: 0.8;
    }

    .empty-title {
      font-size: 18px;
      font-weight: 600;
      color: var(--tm-text-primary);
      margin: 0 0 8px;
    }

    .empty-desc {
      font-size: 14px;
      max-width: 280px;
      line-height: 1.5;
      margin-bottom: 32px;
    }

    &.api-key-missing {
      .el-button {
        margin-top: 8px;
        padding: 12px 24px;
        border-radius: var(--tm-border-radius-sm);
      }
    }

    .preset-questions {
      display: flex;
      flex-direction: column;
      gap: 12px;
      width: 100%;
      max-width: 300px;

      .preset-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px 16px;
        background-color: var(--tm-bg-subtle);
        border: 1px solid var(--tm-border-color);
        border-radius: var(--tm-border-radius-sm);
        font-size: 13px;
        color: var(--tm-text-regular);
        cursor: pointer;
        transition: all 0.2s;

        &:hover {
          background-color: var(--tm-accent-light);
          border-color: var(--tm-accent-primary);
          color: var(--tm-accent-primary);
        }

        .el-icon {
          font-size: 16px;
        }
      }
    }
  }

  .chat-list {
    display: flex;
    flex-direction: column;
    gap: 24px;

    .chat-message {
      display: flex;
      gap: 12px;
      
      &.is-user {
        flex-direction: row-reverse;

        .message-avatar {
          background-color: var(--tm-accent-primary);
          color: white;
        }

        .message-bubble {
          background-color: var(--tm-accent-primary);
          color: white;
          border-top-right-radius: 4px;
        }
      }

      &.is-ai {
        .message-avatar {
          background-color: var(--tm-bg-subtle);
          border: 1px solid var(--tm-border-color);
          color: var(--tm-accent-primary);
        }

        .message-bubble {
          background-color: var(--tm-bg-subtle);
          color: var(--tm-text-primary);
          border: 1px solid var(--tm-border-color);
          border-top-left-radius: 4px;
        }
      }

      .message-avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
      }

      .message-content {
        max-width: 90%;
        display: flex;
        flex-direction: column;
        
        .message-bubble {
          padding: 12px 16px;
          border-radius: 12px;
          font-size: 14px;
          line-height: 1.6;
          word-break: break-word;
          white-space: pre-wrap;
          box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
          
          &.markdown-body {
            white-space: normal;
          }
        }
      }
    }
  }
}
</style>
