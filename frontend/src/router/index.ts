import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import Layout from '../components/Layout.vue'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '概览控制台', icon: 'Grid' }
      },
      {
        path: 'data-source',
        name: 'DataSource',
        component: () => import('../views/DataSource.vue'),
        meta: { title: '数据源管理', icon: 'Coin' }
      },
      {
        path: 'tag-rule',
        name: 'TagRule',
        component: () => import('../views/TagRuleConfig.vue'),
        meta: { title: '标签与规则配置', icon: 'House' }
      },
      {
        path: 'task-kanban',
        name: 'TaskKanban',
        component: () => import('../views/TaskKanban.vue'),
        meta: { title: '打标任务看板', icon: 'Finished' }
      }
    ]
  }
]

const router = createRouter({
  // Wails 推荐使用 Hash 模式
  history: createWebHashHistory(),
  routes
})

export default router
