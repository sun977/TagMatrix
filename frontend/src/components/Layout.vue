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
        <div class="version-info" v-if="!isCollapsed">
          © 2026 {{ authorName }} | v{{ appVersion }}
        </div>
      </div>

      <!-- 拖拽调节宽度的把手 -->
      <div class="sidebar-resizer" @mousedown="startDrag" v-show="!isCollapsed"></div>
    </aside>

    <!-- 主容器 -->
    <div class="content-wrapper">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>

    <!-- 右下角 AI 助手悬浮按钮 (暂时隐藏，等开发AI功能时再开启) -->
    <!-- <div class="ai-assistant-btn" @click="toggleAIPanel">
      <el-icon :size="24"><Service /></el-icon>
    </div> -->

    <!-- 全局设置模态框 -->
    <SettingsDialog v-model="isSettingsOpen" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onUnmounted, onMounted } from 'vue'
import { useRouter } from 'vue-router'
// @ts-ignore: Vetur / TS plugin issue with script setup
import SettingsDialog from './SettingsDialog.vue'
import { GetAppConfig } from '../../wailsjs/go/main/App'
import { config } from '../../wailsjs/go/models'

const router = useRouter()

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

const toggleAIPanel = () => {
  // TODO: 呼出 AI 面板
  console.log('Toggle AI Panel')
}

// --- 侧边栏拖拽调节宽度逻辑 ---
const sidebarWidth = ref(240)
const isCollapsed = ref(false)
const isDragging = ref(false)

const minWidth = 200
const maxWidth = 500
const collapsedWidth = 68 // 折叠后的宽度

const actualSidebarWidth = computed(() => {
  return isCollapsed.value ? collapsedWidth : sidebarWidth.value
})

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
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
.layout-container {
  display: flex;
  width: 100vw;
  height: 100vh;
  background-color: var(--tm-bg-main);
  overflow: hidden;

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
    gap: 12px;
    padding: 0 12px 40px;
    white-space: nowrap;
    overflow: hidden;
    
    .app-logo {
      width: 28px;
      height: 28px;
      object-fit: contain;
      border-radius: 4px;
      flex-shrink: 0;

      &.is-collapsed {
        width: 32px;
        height: 32px;
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
  position: relative;
  background-color: var(--tm-bg-main);
}

/* --- 右下角 AI 助手按钮 --- */
.ai-assistant-btn {
  position: fixed;
  right: 32px;
  bottom: 32px;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background-color: var(--tm-accent-primary);
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(82, 196, 143, 0.3);
  cursor: pointer;
  z-index: 100;
  transition: transform 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(82, 196, 143, 0.4);
  }

  &:active {
    transform: translateY(0);
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