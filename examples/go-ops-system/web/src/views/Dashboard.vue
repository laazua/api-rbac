<template>
  <div class="dashboard">
    <h2 class="page-title">📊 运维仪表盘</h2>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6" v-if="hasAnyPermission('server')">
        <el-card shadow="hover" class="stat-card" @click="$router.push('/servers')">
          <div class="stat-icon" style="background: #e6f7ff;">
            <el-icon :size="32" color="#1890ff"><Monitor /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-label">服务器总数</div>
            <div class="stat-value">{{ loading ? '-' : serverCount }}</div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6" v-if="hasAnyPermission('deployment')">
        <el-card shadow="hover" class="stat-card" @click="$router.push('/deployments')">
          <div class="stat-icon" style="background: #f6ffed;">
            <el-icon :size="32" color="#52c41a"><Upload /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-label">发布总数</div>
            <div class="stat-value">{{ loading ? '-' : deployCount }}</div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6" v-if="hasAnyPermission('alert')">
        <el-card shadow="hover" class="stat-card" @click="$router.push('/alerts')">
          <div class="stat-icon" style="background: #fff7e6;">
            <el-icon :size="32" color="#fa8c16"><Bell /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-label">未确认告警</div>
            <div class="stat-value" style="color: #f5222d;">{{ loading ? '-' : unackedCount }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快捷入口 -->
    <el-row :gutter="20" class="quick-actions">
      <el-col :span="24">
        <el-card shadow="never">
          <template #header>
            <span>🔗 快捷入口</span>
          </template>
          <el-space wrap :size="12">
            <el-button
              v-if="hasAnyPermission('server')"
              type="primary"
              @click="$router.push('/servers')"
            >
              🖥️ 服务器管理
            </el-button>
            <el-button
              v-if="hasAnyPermission('deployment')"
              type="success"
              @click="$router.push('/deployments')"
            >
              📦 发布管理
            </el-button>
            <el-button
              v-if="hasAnyPermission('alert')"
              type="warning"
              @click="$router.push('/alerts')"
            >
              🔔 告警管理
            </el-button>
            <el-button
              type="info"
              @click="$router.push('/my-permissions')"
            >
              🔑 我的权限
            </el-button>
          </el-space>
        </el-card>
      </el-col>
    </el-row>

    <!-- RBAC 集成说明 -->
    <el-row :gutter="20" class="info-row">
      <el-col :span="24">
        <el-card shadow="never">
          <template #header>
            <span>📖 权限集成说明</span>
          </template>
          <el-collapse>
            <el-collapse-item title="🔐 三层权限控制模型" name="1">
              <el-steps direction="vertical" :space="30">
                <el-step
                  title="第三层: 后端 API 鉴权 (安全底线)"
                  description="Go Gin 后端通过 ResilientGuard 中间件调用 api-rbac 检查权限。无权限返回 403，即使绕过前端也无法访问。"
                  status="success"
                />
                <el-step
                  title="第二层: 前端按钮显隐 (用户体验)"
                  description="使用 v-if=&quot;hasPermission('server','restart')&quot; 控制操作按钮，没有权限的用户看不到按钮。"
                />
                <el-step
                  title="第一层: 前端菜单显隐 (导航级别)"
                  description="使用 v-if=&quot;hasAnyPermission('server')&quot; 控制菜单项，没有相关权限的模块菜单不会出现。"
                />
              </el-steps>
            </el-collapse-item>
            <el-collapse-item title="🔄 韧性降级机制" name="2">
              <p>
                本系统使用 <code>ResilientGuard</code> 中间件实现韧性权限校验:
              </p>
              <ul>
                <li><strong>熔断器:</strong> 连续 5 次访问 RBAC 失败后自动熔断 30 秒</li>
                <li><strong>本地缓存:</strong> 5 分钟 TTL 权限缓存，熔断期间走缓存</li>
                <li><strong>降级模式:</strong> <code>FailModeCache</code> — RBAC 宕机时使用缓存数据</li>
              </ul>
            </el-collapse-item>
          </el-collapse>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Monitor, Upload, Bell } from '@element-plus/icons-vue'
import { hasAnyPermission } from '../utils/permission'
import { getServers, getDeployments, getAlerts } from '../api'

const loading = ref(true)
const serverCount = ref(0)
const deployCount = ref(0)
const unackedCount = ref(0)

onMounted(async () => {
  try {
    if (hasAnyPermission('server')) {
      const res = await getServers()
      serverCount.value = Array.isArray(res.data) ? res.data.length : 0
    }
  } catch { /* 忽略权限不足 */ }

  try {
    if (hasAnyPermission('deployment')) {
      const res = await getDeployments()
      deployCount.value = Array.isArray(res.data) ? res.data.length : 0
    }
  } catch { /* 忽略权限不足 */ }

  try {
    if (hasAnyPermission('alert')) {
      const res = await getAlerts()
      if (Array.isArray(res.data)) {
        unackedCount.value = res.data.filter((a) => !a.acked).length
      }
    }
  } catch { /* 忽略权限不足 */ }

  loading.value = false
})
</script>
