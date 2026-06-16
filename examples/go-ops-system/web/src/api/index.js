import axios from 'axios'

// ================================================================
// Axios 实例
//   模块嵌入模式: token 由 api-rbac 门户通过 iframe URL 参数或 postMessage 传入
//   后端 proxy 到 Go 服务, Go 服务转发鉴权到 api-rbac
// ================================================================

const request = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

// 请求拦截器 — 自动附带 Token (从 localStorage 读取)
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('ops_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器 — 统一错误处理
request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      if ([1002, 1007, 1008].includes(res.code)) {
        localStorage.removeItem('ops_token')
        localStorage.removeItem('ops_permissions')
        return Promise.reject(new Error(res.message || '认证失败, 请刷新 api-rbac 门户页面'))
      }
      if (res.code === 1003) {
        return Promise.reject(new Error(res.message || '无权限'))
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res
  },
  (error) => {
    if (error.response) {
      const status = error.response.status
      if (status === 401 || status === 403) {
        return Promise.reject(new Error('无权限执行此操作'))
      }
    }
    return Promise.reject(error)
  }
)

// ================================================================
// API 函数
// ================================================================

// --- 认证 (获取用户权限, 供前端 bootstrap 使用) ---
export function getPermissions() {
  return request.get('/auth/permissions')
}

// --- 服务器管理 ---
export function getServers() {
  return request.get('/servers')
}

export function createServer(data) {
  return request.post('/servers', data)
}

export function deleteServer(id) {
  return request.delete(`/servers/${id}`)
}

export function restartServer(id) {
  return request.post('/servers/restart', { id })
}

export function stopServer(id) {
  return request.post('/servers/stop', { id })
}

// --- 发布管理 ---
export function getDeployments() {
  return request.get('/deployments')
}

export function getDeployment(id) {
  return request.get(`/deployments/${id}`)
}

export function executeDeployment(data) {
  return request.post('/deployments', data)
}

export function rollbackDeployment(id) {
  return request.post('/deployments/rollback', { id })
}

// --- 告警管理 ---
export function getAlerts() {
  return request.get('/alerts')
}

export function ackAlert(id) {
  return request.post('/alerts/ack', { id })
}
