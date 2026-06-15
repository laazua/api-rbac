import Vue from 'vue'
import VueRouter from 'vue-router'
import { hasAnyPermission } from '../api'

Vue.use(VueRouter)

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/portal',
    name: 'Portal',
    component: () => import('../views/Portal.vue'),
    meta: { title: '模块门户' }
  },
  {
    path: '/module/:code',
    name: 'ModuleFrame',
    component: () => import('../views/ModuleFrame.vue'),
    meta: { title: '外部模块' }
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '系统概览', icon: 'el-icon-s-home', resource: null }
      },
      {
        path: 'users',
        name: 'UserManage',
        component: () => import('../views/UserManage.vue'),
        meta: { title: '用户管理', icon: 'el-icon-user', resource: 'user' }
      },
      {
        path: 'roles',
        name: 'RoleManage',
        component: () => import('../views/RoleManage.vue'),
        meta: { title: '角色管理', icon: 'el-icon-s-custom', resource: 'role' }
      },
      {
        path: 'permissions',
        name: 'PermissionManage',
        component: () => import('../views/PermissionManage.vue'),
        meta: { title: '权限管理', icon: 'el-icon-lock', resource: 'permission' }
      },
      {
        path: 'modules',
        name: 'ModuleManage',
        component: () => import('../views/ModuleManage.vue'),
        meta: { title: '模块管理', icon: 'el-icon-s-grid', resource: 'module' }
      }
    ]
  }
]

const router = new VueRouter({
  mode: 'hash',
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')

  // 未登录 → 跳转登录页
  if (to.path !== '/login' && !token) {
    next('/login')
    return
  }

  // 已登录 + 访问登录页 → 跳转门户
  if (to.path === '/login' && token) {
    next('/portal')
    return
  }

  // 已登录 + 访问根路径 → 跳转门户
  if (to.path === '/' && token) {
    next('/portal')
    return
  }

  // 权限守卫：RBAC 子系统内，检查资源权限
  if (to.meta && to.meta.resource) {
    if (!hasAnyPermission(to.meta.resource)) {
      next('/dashboard')
      return
    }
  }

  next()
})

export default router
