import Vue from 'vue'
import VueRouter from 'vue-router'

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
        meta: { title: '仪表盘', icon: 'el-icon-s-home' }
      },
      {
        path: 'users',
        name: 'UserManage',
        component: () => import('../views/UserManage.vue'),
        meta: { title: '用户管理', icon: 'el-icon-user' }
      },
      {
        path: 'roles',
        name: 'RoleManage',
        component: () => import('../views/RoleManage.vue'),
        meta: { title: '角色管理', icon: 'el-icon-s-custom' }
      },
      {
        path: 'permissions',
        name: 'PermissionManage',
        component: () => import('../views/PermissionManage.vue'),
        meta: { title: '权限管理', icon: 'el-icon-lock' }
      }
    ]
  }
]

const router = new VueRouter({
  mode: 'hash',
  routes
})

// 路由守卫：未登录跳转登录页
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
