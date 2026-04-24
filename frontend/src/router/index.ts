import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { title: '概览', icon: 'House' }
  },
  {
    path: '/data-source',
    name: 'DataSource',
    component: () => import('../views/DataSource.vue'),
    meta: { title: '数据源管理', icon: 'Files' }
  },
  {
    path: '/tag-rule',
    name: 'TagRule',
    component: () => import('../views/TagRuleConfig.vue'),
    meta: { title: '标签与规则配置', icon: 'PriceTag' }
  },
  {
    path: '/task-kanban',
    name: 'TaskKanban',
    component: () => import('../views/TaskKanban.vue'),
    meta: { title: '任务看板', icon: 'List' }
  }
]

const router = createRouter({
  // Wails 推荐使用 Hash 模式
  history: createWebHashHistory(),
  routes
})

export default router
