<template>
  <div class="alert-manage">
    <h2 class="page-title">🔔 告警管理</h2>

    <!-- 告警统计 -->
    <el-row :gutter="20" class="alert-stats">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card" body-style="text-align:center;">
          <div style="font-size:28px;color:#f5222d;">{{ criticalCount }}</div>
          <div style="color:#999;">严重告警</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card" body-style="text-align:center;">
          <div style="font-size:28px;color:#fa8c16;">{{ warningCount }}</div>
          <div style="color:#999;">警告告警</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card" body-style="text-align:center;">
          <div style="font-size:28px;color:#1890ff;">{{ infoCount }}</div>
          <div style="color:#999;">信息告警</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card" body-style="text-align:center;">
          <div style="font-size:28px;color:#f5222d;">{{ unackedCount }}</div>
          <div style="color:#999;">未确认</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 告警列表 -->
    <el-card shadow="never">
      <el-table :data="alerts" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag
              :type="levelType(row.level)"
              effect="dark"
              size="small"
            >
              {{ levelText(row.level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="source" label="来源" width="140" />
        <el-table-column prop="message" label="告警内容" min-width="300" show-overflow-tooltip />
        <el-table-column prop="time" label="时间" width="180" />
        <el-table-column prop="acked" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.acked ? 'success' : 'danger'" size="small">
              {{ row.acked ? '已确认' : '未确认' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="acked_by" label="确认人" width="100">
          <template #default="{ row }">
            {{ row.acked_by || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button
              v-if="hasPermission('alert', 'ack') && !row.acked"
              type="primary"
              size="small"
              @click="handleAck(row)"
            >
              ✅ 确认
            </el-button>
            <span v-else-if="row.acked" style="color:#999;">已处理</span>
            <span v-else class="no-action">无操作权限</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { hasPermission } from '../utils/permission'
import { getAlerts, ackAlert } from '../api'

const alerts = ref([])
const loading = ref(false)

const criticalCount = computed(() => alerts.value.filter((a) => a.level === 'critical').length)
const warningCount = computed(() => alerts.value.filter((a) => a.level === 'warning').length)
const infoCount = computed(() => alerts.value.filter((a) => a.level === 'info').length)
const unackedCount = computed(() => alerts.value.filter((a) => !a.acked).length)

onMounted(() => fetchAlerts())

async function fetchAlerts() {
  loading.value = true
  try {
    const res = await getAlerts()
    alerts.value = Array.isArray(res.data) ? res.data : []
  } catch (err) {
    ElMessage.error('加载告警列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

async function handleAck(row) {
  try {
    await ackAlert(row.id)
    ElMessage.success('告警已确认')
    await fetchAlerts()
  } catch (err) {
    ElMessage.error('确认告警失败: ' + err.message)
  }
}

function levelType(level) {
  const map = { critical: 'danger', warning: 'warning', info: 'info' }
  return map[level] || 'info'
}

function levelText(level) {
  const map = { critical: '严重', warning: '警告', info: '信息' }
  return map[level] || level
}
</script>
