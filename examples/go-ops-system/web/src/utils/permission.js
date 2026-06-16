import { getPermissions } from '../api'

// ================================================================
// 前端权限判断
//
// 权限数据格式: { "server": ["read","restart"], "deployment": ["read","execute"], ... }
// ================================================================

/**
 * 引导初始化: 验证 token 有效性, 获取用户权限
 * 在 main.js 中调用, 应用启动前执行
 */
export async function bootstrapToken() {
  const token = localStorage.getItem('ops_token')
  if (!token) return false

  try {
    // 调用后端获取权限 (后端会转发到 api-rbac 验证 token)
    const res = await getPermissions()
    if (res.code === 0 && res.data) {
      localStorage.setItem('ops_permissions', JSON.stringify(res.data))
      return true
    }
  } catch {
    // token 无效或服务不可达
    console.warn('Token 验证失败, 等待来自 api-rbac 的 postMessage...')
    return false
  }

  return false
}

/**
 * 获取用户全部权限
 */
export function getPermissionsMap() {
  try {
    const raw = localStorage.getItem('ops_permissions')
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
  }
}

/**
 * 检查是否有指定权限 (支持通配符)
 */
export function hasPermission(resource, action) {
  const perms = getPermissionsMap()
  if (perms['*'] && perms['*'].includes('*')) return true
  if (perms[resource] && perms[resource].includes('*')) return true
  if (perms['*'] && perms['*'].includes(action)) return true
  return perms[resource] && perms[resource].includes(action)
}

/**
 * 检查是否有某资源的任意权限 (菜单显隐用)
 */
export function hasAnyPermission(resource) {
  const perms = getPermissionsMap()
  if (perms['*']) return true
  return perms[resource] && perms[resource].length > 0
}

/**
 * 是否已登录 (token 存在且权限已加载)
 */
export function isLoggedIn() {
  return !!(localStorage.getItem('ops_token'))
}

/**
 * 获取当前用户名
 */
export function getUsername() {
  return localStorage.getItem('ops_username') || ''
}

/**
 * 清除认证信息
 */
export function clearAuth() {
  localStorage.removeItem('ops_token')
  localStorage.removeItem('ops_username')
  localStorage.removeItem('ops_permissions')
}
