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
    path: '/',
    component: () => import('../views/Layout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'el-icon-s-home', resource: null }
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
      }
    ]
  }
]

const router = new VueRouter({
  mode: 'hash',
  routes
})

// 路由守卫：未登录跳转登录页，无权限跳转仪表盘
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')

  if (to.path !== '/login' && !token) {
    next('/login')
    return
  }

  if (to.path === '/login' && token) {
    next('/dashboard')
    return
  }

  // 权限守卫：除仪表盘外，检查菜单可见性
  if (to.meta && to.meta.resource) {
    if (!hasAnyPermission(to.meta.resource)) {
      next('/dashboard')
      return
    }
  }

  next()
})

export default router
