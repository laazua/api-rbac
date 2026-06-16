import { createApp, ref } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import App from './App.vue'
import router from './router'
import { bootstrapToken } from './utils/permission'
import './styles/global.css'

// ================================================================
// 模块嵌入模式: 从 api-rbac 门户通过 iframe 进入, token 由父页面传递
//
// Token 传递方式 (两种):
//   1. URL 参数: ?rbac_token=xxx (iframe src 携带)
//   2. postMessage: { type: 'RBAC_TOKEN', token, username } (iframe load 后发送)
// ================================================================

async function init() {
  // 1. 尝试从 URL 获取 token
  const urlParams = new URLSearchParams(window.location.search)
  const urlToken = urlParams.get('rbac_token')

  if (urlToken) {
    // 立即存储, 后续 bootstrapToken 会验证并获取权限
    localStorage.setItem('ops_token', urlToken)
    // 清除 URL 中的 token (安全考虑, 不影响 hash 路由)
  }

  // 2. 监听来自父页面的 postMessage
  window.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'RBAC_TOKEN') {
      if (event.data.token) {
        localStorage.setItem('ops_token', event.data.token)
        if (event.data.username) {
          localStorage.setItem('ops_username', event.data.username)
        }
        // token 更新后重新引导
        bootstrapToken().catch(() => {})
      }
    }
  })

  // 3. 验证 token 并获取权限
  await bootstrapToken()

  // 4. 创建 Vue 应用
  const app = createApp(App)
  app.use(ElementPlus, { locale: zhCn })
  app.use(router)
  app.mount('#app')
}

init()
