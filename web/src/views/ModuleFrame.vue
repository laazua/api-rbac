<template>
  <div class="module-frame">
    <!-- 顶部栏 -->
    <div class="frame-header">
      <div class="header-left">
        <el-button type="text" icon="el-icon-back" @click="goPortal">返回门户</el-button>
        <span class="divider">|</span>
        <span class="module-title">{{ moduleName }}</span>
      </div>
      <div class="header-right">
        <el-tag v-if="loading" type="info" size="small">加载中...</el-tag>
        <el-tag v-else-if="loadError" type="danger" size="small">加载失败</el-tag>
        <el-tag v-else type="success" size="small">已连接</el-tag>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="loadError" class="frame-error">
      <el-empty description="模块加载失败">
        <div slot="description" style="color:#909399;font-size:13px">
          <p>无法加载模块「{{ moduleName }}」</p>
          <p style="font-size:12px">请确认模块地址配置正确且服务已启动</p>
          <p style="font-size:12px;color:#c0c4cc">目标地址: {{ moduleUrl }}</p>
        </div>
        <el-button type="primary" @click="reloadFrame">重新加载</el-button>
        <el-button @click="goPortal">返回门户</el-button>
      </el-empty>
    </div>

    <!-- iframe -->
    <iframe
      v-show="!loadError"
      ref="iframe"
      :src="iframeSrc"
      class="frame-iframe"
      frameborder="0"
      @load="onIframeLoad"
      @error="onIframeError"
    />
  </div>
</template>

<script>
import { getModules } from '../api'

export default {
  name: 'ModuleFrame',
  data() {
    return {
      moduleCode: '',
      moduleName: '',
      moduleUrl: '',
      iframeSrc: '',
      loading: true,
      loadError: false
    }
  },
  async created() {
    this.moduleCode = this.$route.params.code
    await this.loadModuleInfo()
  },
  methods: {
    async loadModuleInfo() {
      this.loading = true
      this.loadError = false
      try {
        // 通过编码查找模块，获取其 URL
        const res = await getModules({ page: 1, page_size: 200 })
        const list = res.data.list || []
        const mod = list.find(m => m.code === this.moduleCode)

        if (!mod) {
          this.loadError = true
          this.moduleName = this.moduleCode
          return
        }

        this.moduleName = mod.name
        this.moduleUrl = mod.url || ''

        if (!this.moduleUrl) {
          this.loadError = true
          return
        }

        // 构建 iframe URL，附带 token 供子模块认证
        const token = localStorage.getItem('token') || ''
        const sep = this.moduleUrl.includes('?') ? '&' : '?'
        this.iframeSrc = `${this.moduleUrl}${sep}rbac_token=${encodeURIComponent(token)}`
      } catch {
        this.loadError = true
      } finally {
        this.loading = false
      }
    },
    onIframeLoad() {
      this.loading = false
      // 通过 postMessage 向 iframe 发送 token（备选方案）
      const iframe = this.$refs.iframe
      if (iframe && iframe.contentWindow) {
        const token = localStorage.getItem('token') || ''
        iframe.contentWindow.postMessage({
          type: 'RBAC_TOKEN',
          token: token,
          username: localStorage.getItem('username') || ''
        }, '*')
      }
    },
    onIframeError() {
      this.loadError = true
      this.loading = false
    },
    reloadFrame() {
      this.loadError = false
      this.loading = true
      this.loadModuleInfo()
    },
    goPortal() {
      this.$router.push('/portal')
    }
  }
}
</script>

<style scoped>
.module-frame {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
}

.frame-header {
  height: 44px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  flex-shrink: 0;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.divider {
  color: #dcdfe6;
}
.module-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.frame-error {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f2f5;
}

.frame-iframe {
  flex: 1;
  width: 100%;
  border: none;
}
</style>
