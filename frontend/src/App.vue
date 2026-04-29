<template>
  <router-view />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { GetAppConfig } from '../wailsjs/go/main/App'

// 初始化主题
const applyTheme = async () => {
  try {
    const cfg = await GetAppConfig()
    if (cfg && cfg.system && cfg.system.theme) {
      const theme = cfg.system.theme
      const htmlEl = document.documentElement

      if (theme === 'dark') {
        htmlEl.classList.add('dark')
      } else if (theme === 'light') {
        htmlEl.classList.remove('dark')
      } else {
        // Auto (跟随系统)
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
          htmlEl.classList.add('dark')
        } else {
          htmlEl.classList.remove('dark')
        }
      }
    }
  } catch (e) {
    console.error('Failed to init theme:', e)
  }
}

onMounted(() => {
  applyTheme()
  
  // 监听系统主题变化（针对 auto 模式）
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', async (e) => {
    const cfg = await GetAppConfig()
    if (cfg && cfg.system && cfg.system.theme === 'auto') {
      if (e.matches) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    }
  })
})

// 监听自定义事件以应对从设置面板触发的主题更改
window.addEventListener('theme-changed', (e: any) => {
  const theme = e.detail
  const htmlEl = document.documentElement
  if (theme === 'dark') {
    htmlEl.classList.add('dark')
  } else if (theme === 'light') {
    htmlEl.classList.remove('dark')
  } else {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      htmlEl.classList.add('dark')
    } else {
      htmlEl.classList.remove('dark')
    }
  }
})
</script>

<style>
/* 全局样式已在 assets/styles/main.scss 中定义 */
</style>
