import { createRouter, createWebHashHistory } from 'vue-router'
import { isLoggedIn, hasAnyPermission } from '../utils/permission'

// ================================================================
// 路由表
//   模块嵌入模式 — 无登录页, token 由 api-rbac 门户通过 iframe 传入
// ================================================================

const routes = [
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'Monitor', resource: null },
      },
      {
        path: 'servers',
        name: 'ServerManage',
        component: () => import('../views/ServerManage.vue'),
        meta: { title: '服务器管理', icon: 'Monitor', resource: 'server' },
      },
      {
        path: 'deployments',
        name: 'DeploymentManage',
        component: () => import('../views/DeploymentManage.vue'),
        meta: { title: '发布管理', icon: 'Upload', resource: 'deployment' },
      },
      {
        path: 'alerts',
        name: 'AlertManage',
        component: () => import('../views/AlertManage.vue'),
        meta: { title: '告警管理', icon: 'Bell', resource: 'alert' },
      },
      {
        path: 'my-permissions',
        name: 'MyPermissions',
        component: () => import('../views/MyPermissions.vue'),
        meta: { title: '我的权限', icon: 'Key', resource: null },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// ================================================================
// 导出菜单路由 (供 Layout 动态菜单使用)
// ================================================================
export function getMenuRoutes() {
  const parentRoute = routes.find((r) => r.path === '/')
  if (!parentRoute || !parentRoute.children) return []

  return parentRoute.children
    .filter((r) => {
      if (r.meta.resource === null) return true
      return hasAnyPermission(r.meta.resource)
    })
    .map((r) => ({
      path: `/${r.path}`,
      title: r.meta.title,
      icon: r.meta.icon,
    }))
}

export default router
