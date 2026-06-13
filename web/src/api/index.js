import axios from 'axios'
import { Message } from 'element-ui'

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000
})

// 请求拦截器：自动携带 Token
request.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => Promise.reject(error)
)

// 响应拦截器：统一错误处理
request.interceptors.response.use(
  response => {
    const res = response.data
    if (res.code !== 0) {
      Message.error(res.message || '请求失败')
      if (res.code === 1002 || res.code === 1007 || res.code === 1008) {
        localStorage.removeItem('token')
        localStorage.removeItem('username')
        window.location.hash = '#/login'
      }
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  error => {
    // 处理 HTTP 错误状态码（401/403/404/500 等）
    if (error.response) {
      const { status, data } = error.response
      const message = (data && data.message) || `请求错误 (${status})`

      if (status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('username')
        localStorage.removeItem('permissions')
        Message.error('登录已过期，请重新登录')
        window.location.hash = '#/login'
      } else if (status === 403) {
        Message.error('无权限执行此操作')
      } else if (status === 404) {
        Message.error('请求的资源不存在')
      } else {
        Message.error(message)
      }
    } else {
      Message.error('网络错误，请检查服务是否启动')
    }
    return Promise.reject(error)
  }
)

// ===== 认证接口 =====

/** 登录 */
export function login(account, password) {
  return request.post('/auth/login', { account, password })
}

/** 验证 Token */
export function verifyToken() {
  return request.post('/auth/verify')
}

/** 检查权限 */
export function checkPermission(resource, action) {
  return request.post('/auth/check', { resource, action })
}

/** 获取当前用户权限菜单 */
export function getMenu() {
  return request.get('/auth/menu')
}

// ===== 前端权限辅助 =====

/** 判断是否有指定权限 */
export function hasPermission(resource, action) {
  try {
    const perms = JSON.parse(localStorage.getItem('permissions') || '{}')
    // 通配符
    if (perms['*'] && perms['*'].includes('*')) return true
    if (perms[resource] && perms[resource].includes('*')) return true
    if (perms['*'] && perms['*'].includes(action)) return true
    return perms[resource] && perms[resource].includes(action)
  } catch {
    return false
  }
}

/** 判断是否有至少一个权限（用于菜单显示） */
export function hasAnyPermission(resource) {
  try {
    const perms = JSON.parse(localStorage.getItem('permissions') || '{}')
    if (perms['*']) return true
    return perms[resource] && perms[resource].length > 0
  } catch {
    return false
  }
}

// ===== 用户管理 =====

export function getUsers(params) {
  return request.get('/users', { params })
}

export function getUser(id) {
  return request.get(`/users/${id}`)
}

export function createUser(data) {
  return request.post('/users', data)
}

export function updateUser(id, data) {
  return request.put(`/users/${id}`, data)
}

export function deleteUser(id) {
  return request.delete(`/users/${id}`)
}

export function changePassword(id, data) {
  return request.put(`/users/${id}/password`, data)
}

export function assignUserRoles(id, roleIds) {
  return request.post(`/users/${id}/roles`, { role_ids: roleIds })
}

export function removeUserRole(id, roleId) {
  return request.delete(`/users/${id}/roles/${roleId}`)
}

// ===== 角色管理 =====

export function getRoles(params) {
  return request.get('/roles', { params })
}

export function getRole(id) {
  return request.get(`/roles/${id}`)
}

export function createRole(data) {
  return request.post('/roles', data)
}

export function updateRole(id, data) {
  return request.put(`/roles/${id}`, data)
}

export function deleteRole(id) {
  return request.delete(`/roles/${id}`)
}

export function assignRolePermissions(id, permIds) {
  return request.post(`/roles/${id}/permissions`, { permission_ids: permIds })
}

export function removeRolePermission(id, permId) {
  return request.delete(`/roles/${id}/permissions/${permId}`)
}

// ===== 权限管理 =====

export function getPermissions(params) {
  return request.get('/permissions', { params })
}

export function getPermission(id) {
  return request.get(`/permissions/${id}`)
}

export function createPermission(data) {
  return request.post('/permissions', data)
}

export function updatePermission(id, data) {
  return request.put(`/permissions/${id}`, data)
}

export function deletePermission(id) {
  return request.delete(`/permissions/${id}`)
}
