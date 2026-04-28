import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import Layout from '../components/Layout.vue'
import { GetAppConfig } from '../../wailsjs/go/main/App'

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
        meta: { title: '数据集管理', icon: 'Folder' }
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
      },
      {
        path: 'tagged-data',
        name: 'TaggedData',
        component: () => import('../views/TaggedData.vue'),
        meta: { title: '打标数据看板', icon: 'DataBoard' }
      },
      {
        path: 'database-admin',
        name: 'DatabaseAdmin',
        component: () => import('../views/dataAdmin/DatabaseAdmin.vue'),
        meta: { title: '数据库管理', icon: 'Coin', requireDev: true }
      }
    ]
  }
]

const router = createRouter({
  // Wails 推荐使用 Hash 模式
  history: createWebHashHistory(),
  routes
})

router.beforeEach(async (to, from, next) => {
  if (to.meta.requireDev) {
    try {
      const cfg = await GetAppConfig()
      if (cfg && cfg.adv && cfg.adv.developer_mode) {
        next()
      } else {
        next('/dashboard')
      }
    } catch (e) {
      next('/dashboard')
    }
  } else {
    next()
  }
})

export default router
