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

        <!-- 全局设置 -->
        <div class="menu-item setting-btn" @click="openSettings">
          <el-icon><Setting /></el-icon>
          <span>全局设置</span>
        </div>
      </nav>
    </aside>

    <!-- 主容器 -->
    <div class="content-wrapper">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>

    <!-- 右下角 AI 助手悬浮按钮 -->
    <div class="ai-assistant-btn" @click="toggleAIPanel">
      <el-icon :size="24"><Service /></el-icon>
    </div>

    <!-- 全局设置模态框 -->
    <SettingsDialog v-model="isSettingsOpen" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Grid, Setting, House, Files, PriceTag, List, Service } from '@element-plus/icons-vue'
import SettingsDialog from './SettingsDialog.vue'

const router = useRouter()

const isSettingsOpen = ref(false)

// 过滤出要在菜单中显示的路由
const menuRoutes = computed(() => {
  return router.options.routes.filter(r => r.meta && r.meta.title)
})

const openSettings = () => {
  isSettingsOpen.value = true
}

const toggleAIPanel = () => {
  // TODO: 呼出 AI 面板
  console.log('Toggle AI Panel')
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
  padding: 24px 16px;
  box-sizing: border-box;
  flex-shrink: 0;

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 12px 40px;
    
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

    .el-icon {
      font-size: 18px;
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
    margin-top: 24px;
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