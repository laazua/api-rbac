<template>
  <div class="portal">
    <!-- 顶部导航栏 -->
    <div class="portal-header">
      <div class="header-left">
        <span class="portal-logo">🔐 模块门户</span>
      </div>
      <div class="header-right">
        <el-dropdown trigger="click" @command="handleCommand">
          <span class="user-info">
            <el-avatar size="small" icon="el-icon-user-solid" />
            <span>{{ username }}</span>
            <i class="el-icon-arrow-down" />
          </span>
          <el-dropdown-menu slot="dropdown">
            <el-dropdown-item command="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
      </div>
    </div>

    <!-- 模块卡片区域 -->
    <div class="portal-body">
      <div class="portal-welcome">
        <h2>欢迎回来，{{ username }}</h2>
        <p>请选择您要进入的功能模块</p>
      </div>

      <div v-loading="loading" class="module-grid">
        <el-empty v-if="!loading && modules.length === 0"
          description="暂无可用模块，请联系管理员分配权限">
        </el-empty>

        <el-row :gutter="24" type="flex">
          <el-col
            v-for="m in modules"
            :key="m.id"
            :xs="24" :sm="12" :md="8" :lg="6"
            style="margin-bottom: 24px"
          >
            <el-card
              class="module-card"
              shadow="hover"
              @click.native="enterModule(m)"
            >
              <div class="card-body">
                <div class="card-icon" :style="{ background: iconBg(m.icon) }">
                  <img v-if="isImageIcon(m.icon)" :src="m.icon" class="card-icon-img" />
                  <i v-else :class="m.icon || 'el-icon-menu'" />
                </div>
                <h3 class="card-title">{{ m.name }}</h3>
                <p class="card-desc">{{ m.description || '暂无描述' }}</p>
              </div>
              <div class="card-footer">
                <span>进入模块</span>
                <i class="el-icon-arrow-right" />
              </div>
            </el-card>
          </el-col>
        </el-row>
      </div>
    </div>

    <!-- 底部 -->
    <div class="portal-footer">
      <span>RBAC 统一权限管理平台</span>
    </div>
  </div>
</template>

<script>
import { getUserModules } from '../api'
import { isImageIcon, iconGradient } from '../utils/icon'

export default {
  name: 'Portal',
  data() {
    return {
      username: localStorage.getItem('username') || '用户',
      modules: [],
      loading: true
    }
  },
  async created() {
    try {
      const res = await getUserModules()
      this.modules = (res.data.modules || []).sort((a, b) => a.sort - b.sort || a.id - b.id)
    } catch {
      this.modules = []
    } finally {
      this.loading = false
    }
  },
  methods: {
    isImageIcon,
    iconBg(icon) { return iconGradient(icon) },
    enterModule(m) {
      // 如果模块配置了外部 URL → 用 iframe 容器加载
      if (m.url) {
        this.$router.push(`/module/${m.code}`)
        return
      }
      // 内置路由映射（RBAC 子系统内部页面）
      const routeMap = {
        'system_mgmt': '/dashboard'
      }
      const target = routeMap[m.code]
      if (target) {
        this.$router.push(target)
      } else {
        this.$message.info(`模块「${m.name}」未配置入口地址，请在模块管理中设置 URL`)
      }
    },
    handleCommand(cmd) {
      if (cmd === 'logout') {
        localStorage.removeItem('token')
        localStorage.removeItem('username')
        localStorage.removeItem('permissions')
        this.$router.push('/login')
        this.$message.success('已退出登录')
      }
    }
  }
}
</script>

<style scoped>
.portal {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f0f2f5;
}

/* 顶栏 */
.portal-header {
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  flex-shrink: 0;
}
.portal-logo {
  font-size: 18px;
  font-weight: 700;
  color: #303133;
}
.user-info {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #606266;
  font-size: 14px;
}
.user-info:hover { color: #409eff; }

/* 主体 */
.portal-body {
  flex: 1;
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  padding: 40px 24px 20px;
}
.portal-welcome {
  margin-bottom: 36px;
  text-align: center;
}
.portal-welcome h2 {
  margin: 0 0 8px 0;
  font-size: 26px;
  color: #303133;
  font-weight: 600;
}
.portal-welcome p {
  margin: 0;
  color: #909399;
  font-size: 15px;
}

.module-grid {
  min-height: 200px;
}

/* 模块卡片 */
.module-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  border: 2px solid transparent;
  cursor: pointer;
  overflow: hidden;
}
.module-card:hover {
  transform: translateY(-6px);
  border-color: #409eff;
  box-shadow: 0 12px 32px rgba(64, 158, 255, 0.18);
}

.card-body {
  padding: 24px 20px 16px;
  text-align: center;
}
.card-icon {
  width: 60px;
  height: 60px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 14px;
}
.card-icon i {
  font-size: 30px;
  color: #fff;
}
.card-icon .card-icon-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  border-radius: 16px;
}
.card-title {
  margin: 0 0 8px 0;
  font-size: 17px;
  font-weight: 600;
  color: #303133;
}
.card-desc {
  margin: 0;
  font-size: 13px;
  color: #909399;
  min-height: 36px;
  line-height: 1.5;
}

.card-footer {
  background: #fafafa;
  padding: 10px 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 13px;
  color: #409eff;
  border-top: 1px solid #f0f0f0;
  transition: background 0.3s;
}
.module-card:hover .card-footer {
  background: #ecf5ff;
}

/* 底部 */
.portal-footer {
  text-align: center;
  padding: 20px;
  color: #c0c4cc;
  font-size: 13px;
  flex-shrink: 0;
}
</style>
