<template>
  <div class="layout-container" :class="{ 'is-dragging': isDragging }">
    <!-- 左侧极简导航栏 -->
    <aside 
      class="sidebar" 
      :class="{ 'is-collapsed': isCollapsed, 'is-dragging': isDragging }"
      :style="{ width: actualSidebarWidth + 'px' }"
    >
      <div class="sidebar-header">
        <img src="../assets/images/appicon.png" alt="TagMatrix Logo" class="app-logo" :class="{ 'is-collapsed': isCollapsed }" />
        <span class="logo-text" v-if="!isCollapsed">TagMatrix</span>
      </div>

      <nav class="sidebar-menu">
        <router-link 
          v-for="route in menuRoutes" 
          :key="route.path"
          :to="'/' + route.path" 
          class="menu-item"
          active-class="is-active"
        >
          <el-icon><component :is="route.meta?.icon" /></el-icon>
          <span v-if="!isCollapsed">{{ route.meta?.title }}</span>
        </router-link>
      </nav>

      <!-- 底部收起按钮及设置按钮 -->
      <div class="sidebar-footer">
        <!-- 全局设置 -->
        <div class="menu-item setting-btn" @click="openSettings">
          <el-icon><Setting /></el-icon>
          <span v-if="!isCollapsed">设置</span>
        </div>
        
        <!-- 底部收起按钮 -->
        <div class="menu-item collapse-btn" @click="toggleCollapse">
          <el-icon><Fold v-if="!isCollapsed" /><Expand v-else /></el-icon>
          <span v-if="!isCollapsed">收起</span>
        </div>

        <!-- 软件版本信息 -->
        <div class="version-info" :class="{ 'is-collapsed': isCollapsed }" @click="handleVersionClick">
          <span v-if="!isCollapsed">© 2026 {{ authorName }} | v{{ appVersion }}</span>
          <el-tooltip v-else :content="`© 2026 ${authorName} | v${appVersion}`" placement="right" :show-after="200">
            <span class="collapsed-text-icon">S</span>
          </el-tooltip>
        </div>
      </div>

      <!-- 拖拽调节宽度的把手 -->
      <div class="sidebar-resizer" @mousedown="startDrag" v-show="!isCollapsed"></div>
    </aside>

    <!-- 主容器 -->
    <div 
      class="content-wrapper"
      :class="{ 'no-transition': aiStore.isDragging }"
      :style="aiStore.isOpen ? { marginRight: aiStore.sidebarWidth + 'px' } : { marginRight: '0px' }"
    >
      <div v-if="firstRunningTask" class="global-running-task-banner">
        <el-icon class="is-loading" style="margin-right: 8px;"><Loading /></el-icon>
        <span>当前正在执行 <strong>{{ firstRunningTask.name }}</strong> 任务，进度 {{ firstRunningTask.progress }}%</span>
      </div>
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>

    <!-- 右侧智能助手面板 -->
    <AICopilotSidebar />

    <!-- 右侧全局布局控制栏 (Layout Controls) -->
    <div 
      class="layout-controls" 
      :class="{ 'is-shifted': aiStore.isOpen, 'no-transition': aiStore.isDragging }" 
      @click="toggleAIPanel"
      :style="aiStore.isOpen ? { right: aiStore.sidebarWidth + 'px' } : {}"
    >
      <el-tooltip content="AI助手" placement="left" :show-after="300">
        <div class="control-btn" :class="{ 'is-active': aiStore.isOpen }">
          <el-icon><CaretLeft v-if="!aiStore.isOpen" /><CaretRight v-else /></el-icon>
        </div>
      </el-tooltip>
    </div>

    <!-- 全局设置模态框 -->
    <SettingsDialog v-model="isSettingsOpen" @saved="handleSettingsSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onUnmounted, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
// @ts-ignore: Vetur / TS plugin issue with script setup
import SettingsDialog from './SettingsDialog.vue'
import AICopilotSidebar from './AICopilot/AICopilotSidebar.vue'
import { useAIStore } from '../store/useAIStore'
import { GetAppConfig, GetTaskBatches } from '../../wailsjs/go/main/App'
import { config, model } from '../../wailsjs/go/models'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { Loading } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'

const router = useRouter()
const route = useRoute()
const aiStore = useAIStore()

const runningTasks = ref<Array<{id: number, name: string, progress: number}>>([])
const firstRunningTask = computed(() => runningTasks.value.length > 0 ? runningTasks.value[0] : null)

const fetchRunningTasks = async () => {
  try {
    const batches = await GetTaskBatches()
    if (batches) {
      runningTasks.value = batches
        .filter((b: model.TagTaskBatch) => b.status === 'running')
        .map((b: model.TagTaskBatch) => ({
          id: b.id,
          name: b.name,
          progress: 0
        }))
    }
  } catch (e) {
    console.error('Failed to fetch task batches', e)
  }
}

// 监听路由变化，自动更新 AI 的上下文
watch(() => route.path, () => {
  if (route.meta && route.meta.title) {
    aiStore.pageContext = route.meta.title as string
  } else {
    aiStore.pageContext = ''
  }
}, { immediate: true })

const appVersion = __APP_VERSION__
const authorName = __APP_AUTHOR__

const isSettingsOpen = ref(false)
const appConfig = ref<config.AppConfig | null>(null)

onMounted(async () => {
  try {
    appConfig.value = await GetAppConfig()
  } catch (e) {
    console.error('Failed to load app config in Layout', e)
  }
  
  window.addEventListener('open-settings', openSettings)

  fetchRunningTasks()

  EventsOn('task_list_updated', fetchRunningTasks)

  EventsOn('taskProgress', (data: any) => {
    if (data.status === 'running') {
      const existing = runningTasks.value.find(t => t.id === data.batchID)
      if (existing) {
        existing.progress = data.progress
      } else {
        // Maybe a newly started task
        fetchRunningTasks()
      }
    } else {
      // Completed, failed, etc.
      runningTasks.value = runningTasks.value.filter(t => t.id !== data.batchID)
    }
  })
})

onUnmounted(() => {
  stopDrag()
  window.removeEventListener('open-settings', openSettings)
  EventsOff('taskProgress')
  EventsOff('task_list_updated')
})

// 过滤出要在菜单中显示的路由
const menuRoutes = computed(() => {
  const mainRoute = router.options.routes.find(r => r.path === '/')
  const allRoutes = mainRoute?.children?.filter(r => r.meta && r.meta.title) || []
  if (appConfig.value && appConfig.value.adv && appConfig.value.adv.developer_mode) {
    return allRoutes
  }
  return allRoutes.filter(r => !r.meta?.requireDev)
})

const openSettings = () => {
  isSettingsOpen.value = true
}

const handleSettingsSaved = async () => {
  try {
    appConfig.value = await GetAppConfig()
    // 更新 AI Store
    aiStore.checkAPIKey()
    // 权限回收判断：如果关了开发者模式且当前在数据库管理页，丝滑退回首页
    if (!appConfig.value?.adv?.developer_mode && router.currentRoute.value.meta.requireDev) {
      router.replace('/dashboard')
    }
  } catch (e) {
    console.error('Failed to reload app config after save', e)
  }
}

const toggleAIPanel = () => {
  aiStore.togglePanel()
}

// --- 侧边栏拖拽调节宽度逻辑 ---
const sidebarWidth = ref(180)
const isCollapsed = ref(false)
const isDragging = ref(false)

const minWidth = 165
const maxWidth = 500
const collapsedWidth = 68 // 折叠后的宽度

const actualSidebarWidth = computed(() => {
  return isCollapsed.value ? collapsedWidth : sidebarWidth.value
})

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
}

// --- 彩蛋逻辑 ---
const versionClickCount = ref(0)
let versionClickTimer: any = null

const handleVersionClick = () => {
  if (isCollapsed.value) return // 收起时不触发彩蛋

  versionClickCount.value++
  
  if (versionClickTimer) {
    clearTimeout(versionClickTimer)
  }
  
  // 连续点击 3 次触发彩蛋
  if (versionClickCount.value >= 3) {
    versionClickCount.value = 0
    ElMessageBox.alert(
      '<div style="text-align: center; padding: 10px 0;">' +
        '<h3 style="margin-top:0; color: var(--tm-text-primary);">发现彩蛋！</h3>' +
        '<p style="color: var(--tm-text-regular); line-height: 1.6; margin: 12px 0;">你好呀，我是本系统的作者 <strong>Sun977</strong>！</p>' +
          '<p style="color: var(--tm-text-regular); margin-bottom:16px; line-height: 1.6;">感谢使用 TagMatrix，愿你的数据标注之路不再痛苦~ ✨</p>' +
          '<div style="display: flex; flex-direction: column; gap: 8px; align-items: center; font-size: 13px; background: rgba(245, 247, 250, 0.5); border: 1px solid var(--tm-border-color); padding: 16px 12px; border-radius: 8px;">' +
            '<img src="/sponsor-code.png" alt="交个朋友" style="width: 180px; height: 180px; border-radius: 6px; border: 1px solid var(--tm-border-color);" onerror="this.style.display=\'none\'" />' +
            '<div style="color: var(--tm-accent-primary); font-weight: 600; font-size: 14px; margin: 4px 0;">交个朋友 / 赞赏支持</div>' +
            '<div style="margin-top: 4px; display: grid; grid-template-columns: 60px 1fr; row-gap: 6px; color: var(--tm-text-regular); text-align: left; line-height: 1.4;">' +
            '<div style="color: var(--tm-text-secondary);">GitHub:</div>' +
            '<div><a href="https://github.com/Sun977" target="_blank" style="color: var(--tm-accent-primary); text-decoration: none; font-weight: 500;">Sun977</a></div>' +
            '<div style="color: var(--tm-text-secondary);">Email:</div>' +
            '<div><span style="user-select: all;">jiuwei977@foxmail.com</span></div>' +
            '<div style="color: var(--tm-text-secondary);">微信:</div>' +
            '<div><span style="user-select: all; font-weight: 500;">EternityCanYang</span></div>' +
          '</div>' +
        '</div>' +
      '</div>',
      '🎉 Surprise!',
      {
        dangerouslyUseHTMLString: true,
        confirmButtonText: '收到啦',
        center: true,
      }
    ).catch(() => {}) // 捕获并忽略关闭弹窗时的错误
  } else {
    // 1秒内没连续点击则重置计数
    versionClickTimer = setTimeout(() => {
      versionClickCount.value = 0
    }, 1000)
  }
}

const startDrag = (e: MouseEvent) => {
  if (isCollapsed.value) return // 折叠状态下不允许拖拽
  isDragging.value = true
  document.body.style.cursor = 'col-resize'
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  // 防止拖拽时选中文本
  e.preventDefault()
}

const onDrag = (e: MouseEvent) => {
  if (!isDragging.value) return
  let newWidth = e.clientX
  if (newWidth < minWidth) newWidth = minWidth
  if (newWidth > maxWidth) newWidth = maxWidth
  sidebarWidth.value = newWidth
}

const stopDrag = () => {
  if (isDragging.value) {
    isDragging.value = false
    document.body.style.cursor = ''
    document.removeEventListener('mousemove', onDrag)
    document.removeEventListener('mouseup', stopDrag)
  }
}

// 组件卸载时确保移除事件监听
onUnmounted(() => {
  stopDrag()
})
</script>

<style scoped lang="scss">
.global-running-task-banner {
  background-color: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  border: 1px solid var(--el-color-warning-light-5);
  padding: 8px 16px;
  text-align: center;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  margin: 16px 24px 0 24px;
}

.layout-container {
  display: flex;
  width: 100vw;
  height: 100vh;
  background-color: var(--tm-bg-main);
  overflow: hidden;
  position: relative;

  // 拖拽时防止全局文本选中
  &.is-dragging {
    user-select: none;
    -webkit-user-select: none;
  }
}

/* --- 左侧边栏 --- */
.sidebar {
  position: relative;
  background-color: var(--tm-bg-sidebar);
  border-right: 1px solid var(--tm-border-color);
  display: flex;
  flex-direction: column;
  padding: 24px 16px;
  box-sizing: border-box;
  flex-shrink: 0;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  will-change: width;

  // 拖拽过程中取消动画过渡，保持鼠标跟随的顺滑
  &.is-dragging {
    transition: none;
  }

  // 折叠状态下的样式调整
  &.is-collapsed {
    padding: 24px 10px;

    .sidebar-header {
      padding: 0 0 40px;
      justify-content: center;
    }

    .menu-item {
      justify-content: center;
      padding: 12px 0;
      
      .el-icon {
        margin-right: 0;
      }
    }
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 4px 40px;
    white-space: nowrap;
    overflow: hidden;
    
    .app-logo {
      width: 28px;
      height: 28px;
      object-fit: contain;
      border-radius: 4px;
      flex-shrink: 0;

      &.is-collapsed {
        width: 28px;
        height: 28px;
      }
    }
    
    .logo-text {
      font-weight: 700;
      font-size: 20px;
      color: var(--tm-text-primary);
      letter-spacing: -0.5px;
    }
  }

  .sidebar-menu {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow-x: hidden;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    border-radius: var(--tm-border-radius-sm);
    color: var(--tm-text-regular);
    text-decoration: none;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.2s ease;
    cursor: pointer;
    white-space: nowrap;

    .el-icon {
      font-size: 18px;
      flex-shrink: 0;
    }

    &:hover {
      background-color: var(--tm-bg-hover);
    }

    &.is-active {
      background-color: var(--tm-bg-active);
      color: var(--tm-text-primary);
      font-weight: 600;
    }
  }

  .setting-btn {
    margin-bottom: 8px;
  }

  .sidebar-footer {
    margin-top: auto;
    padding-top: 16px;
    border-top: 1px solid var(--tm-border-color);
    overflow: hidden;
    display: flex;
    flex-direction: column;

    .version-info {
      font-size: 12px;
      color: var(--tm-text-secondary);
      text-align: center;
      margin-top: 16px;
      padding-bottom: 4px;
      opacity: 0.6;
      white-space: nowrap;
      user-select: none;
      cursor: pointer;
      transition: opacity 0.2s ease;

      &:hover {
        opacity: 0.9;
      }

      &.is-collapsed {
        font-size: 16px;
        display: flex;
        justify-content: center;
        align-items: center;
        
        .collapsed-text-icon {
          font-weight: 700;
          font-size: 16px;
          font-family: monospace;
          color: var(--tm-text-secondary);
          transition: color 0.2s ease;
        }

        &:hover {
          .collapsed-text-icon {
            color: var(--tm-text-primary);
          }
        }
      }
    }
  }

  /* --- 拖拽调整宽度的把手 --- */
  .sidebar-resizer {
    position: absolute;
    top: 0;
    right: -3px; // 悬浮在边框线上
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

/* --- 中间主容器 --- */
.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0; /* 避免 flex 子项超出 */
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  background-color: var(--tm-bg-main);
  transition: margin-right 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  &.no-transition {
    transition: none !important;
  }
}

/* --- 右侧全局布局控制栏 (Layout Controls) --- */
.layout-controls {
  position: absolute;
  top: 15%;
  right: 0;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  background-color: var(--tm-bg-main);
  border: 1px solid var(--tm-border-color);
  border-right: none;
  border-radius: 6px 0 0 6px;
  box-shadow: -2px 0 8px rgba(0, 0, 0, 0.05);
  z-index: 100;
  transition: right 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;

  &.no-transition {
    transition: none !important;
  }

  .control-btn {
    width: 14px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--tm-text-secondary);
    transition: all 0.2s;

    &:hover {
      background-color: var(--tm-bg-hover);
      color: var(--tm-text-primary);
    }

    &.is-active {
      color: var(--tm-text-primary);
    }
  }
}

// 简单的路由切换动画
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>