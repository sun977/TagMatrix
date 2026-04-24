<template>
  <div class="layout-container">
    <!-- 左侧极简导航栏 -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <el-icon :size="24" color="var(--tm-accent-primary)"><Grid /></el-icon>
        <span class="logo-text">TagMatrix</span>
      </div>

      <nav class="sidebar-menu">
        <router-link 
          v-for="route in menuRoutes" 
          :key="route.path"
          :to="route.path" 
          class="menu-item"
          active-class="is-active"
        >
          <el-icon><component :is="route.meta?.icon" /></el-icon>
          <span>{{ route.meta?.title }}</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="menu-item setting-btn" @click="openSettings">
          <el-icon><Setting /></el-icon>
          <span>全局设置</span>
        </div>
      </div>
    </aside>

    <!-- 中间与右侧容器 -->
    <div class="content-wrapper">
      <!-- 顶部顶栏 -->
      <header class="header">
        <div class="header-left">
          <h2>{{ currentRouteTitle }}</h2>
        </div>
        <div class="header-right">
          <div class="task-marquee">
            <!-- 预留全局任务状态跑马灯 -->
            <el-icon class="is-loading"><Loading /></el-icon>
            <span>系统空闲中...</span>
          </div>
          <el-button round @click="toggleRightPanel">
            <el-icon><Files /></el-icon>
            AI 对话 / 文件
          </el-button>
        </div>
      </header>

      <!-- 视图与右侧面板容器 -->
      <div class="main-body">
        <!-- 中央工作区 -->
        <main class="main-content" :class="{ 'panel-open': isRightPanelOpen }">
          <router-view v-slot="{ Component }">
            <transition name="fade" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </main>

        <!-- 右侧面板 (折叠隐藏) -->
        <aside class="right-panel" :class="{ 'is-open': isRightPanelOpen }">
          <div class="panel-header">
            <h3>AI 智能助手</h3>
            <el-icon class="close-icon" @click="toggleRightPanel"><Close /></el-icon>
          </div>
          <div class="panel-content">
            <!-- 这里之后放 AI 对话组件 -->
            <p class="placeholder-text">你好，我是 TagMatrix AI 助手。</p>
          </div>
        </aside>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Grid, Setting, House, Files, PriceTag, List, Loading, Close } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

const isRightPanelOpen = ref(false)

// 过滤出要在菜单中显示的路由
const menuRoutes = computed(() => {
  return router.options.routes.filter(r => r.meta && r.meta.title)
})

const currentRouteTitle = computed(() => {
  return route.meta.title || '概览'
})

const openSettings = () => {
  // TODO: 呼出设置模态框
  console.log('Open Settings')
}

const toggleRightPanel = () => {
  isRightPanelOpen.value = !isRightPanelOpen.value
}
</script>

<style scoped lang="scss">
.layout-container {
  display: flex;
  width: 100vw;
  height: 100vh;
  background-color: var(--tm-bg-main);
  overflow: hidden;
}

/* --- 左侧边栏 --- */
.sidebar {
  width: 240px;
  background-color: var(--tm-bg-sidebar);
  border-right: 1px solid var(--tm-border-color);
  display: flex;
  flex-direction: column;
  padding: 20px 16px;
  box-sizing: border-box;
  flex-shrink: 0;

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 12px 30px;
    
    .logo-text {
      font-weight: 600;
      font-size: 18px;
      color: var(--tm-text-primary);
      letter-spacing: -0.5px;
    }
  }

  .sidebar-menu {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    border-radius: var(--tm-border-radius-sm);
    color: var(--tm-text-regular);
    text-decoration: none;
    font-size: 14px;
    font-weight: 500;
    transition: all 0.2s ease;
    cursor: pointer;

    &:hover {
      background-color: var(--tm-bg-hover);
    }

    &.is-active {
      background-color: var(--tm-bg-active);
      color: var(--tm-text-primary);
      font-weight: 600;
    }
  }

  .sidebar-footer {
    padding-top: 20px;
    border-top: 1px solid var(--tm-border-color);
  }
}

/* --- 中间与右侧容器 --- */
.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0; /* 避免 flex 子项超出 */
}

/* --- 顶部顶栏 --- */
.header {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  border-bottom: 1px solid var(--tm-border-color);
  background-color: var(--tm-bg-main);

  .header-left h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--tm-text-primary);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 20px;

    .task-marquee {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
      color: var(--tm-text-secondary);
    }
  }
}

/* --- 视图与右侧面板容器 --- */
.main-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}

/* --- 中央工作区 --- */
.main-content {
  flex: 1;
  padding: 32px;
  overflow-y: auto;
  background-color: var(--tm-bg-main);
  transition: padding-right 0.3s ease;
}

/* --- 右侧面板 --- */
.right-panel {
  width: 320px;
  background-color: var(--tm-bg-sidebar); /* 与侧边栏一样的浅灰 */
  border-left: 1px solid var(--tm-border-color);
  display: flex;
  flex-direction: column;
  transform: translateX(100%);
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 10;

  &.is-open {
    transform: translateX(0);
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid var(--tm-border-color);

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
    }

    .close-icon {
      cursor: pointer;
      font-size: 18px;
      color: var(--tm-text-secondary);
      transition: color 0.2s;

      &:hover {
        color: var(--tm-text-primary);
      }
    }
  }

  .panel-content {
    flex: 1;
    padding: 20px;
    overflow-y: auto;

    .placeholder-text {
      color: var(--tm-text-secondary);
      font-size: 14px;
      text-align: center;
      margin-top: 40px;
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
