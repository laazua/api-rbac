<template>
  <!-- 等待 token 初始化完成 -->
  <div v-if="initializing" class="bootstrap-loading">
    <div class="bootstrap-box">
      <el-icon class="loading-icon" :size="48"><Loading /></el-icon>
      <h3>正在连接 api-rbac ...</h3>
      <p>请从 <strong>api-rbac 门户页面</strong> 点击模块卡片进入</p>
      <p class="hint-text">Token 通过 iframe URL 参数或 postMessage 自动传递</p>
    </div>
  </div>

  <router-view v-else />
</template>

<script setup>
import { ref } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import { isLoggedIn } from './utils/permission'

const initializing = ref(!isLoggedIn())

// 监听权限就绪后解除 loading
const checkReady = setInterval(() => {
  if (isLoggedIn()) {
    initializing.value = false
    clearInterval(checkReady)
  }
}, 500)

// 最多等待 10 秒
setTimeout(() => {
  clearInterval(checkReady)
  initializing.value = false
}, 10000)
</script>

<style scoped>
.bootstrap-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f0f2f5;
}

.bootstrap-box {
  text-align: center;
  padding: 48px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.loading-icon {
  color: #409eff;
  animation: spin 1.5s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.bootstrap-box h3 {
  color: #303133;
  margin-bottom: 12px;
}

.bootstrap-box p {
  color: #909399;
  font-size: 14px;
  margin-bottom: 4px;
}

.hint-text {
  font-size: 12px;
  color: #c0c4cc;
}
</style>
