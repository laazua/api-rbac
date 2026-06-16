<template>
  <div class="server-manage">
    <div class="page-header">
      <h2 class="page-title">🖥️ 服务器管理</h2>
      <el-button
        v-if="hasPermission('server', 'create')"
        type="primary"
        @click="showCreateDialog"
      >
        新增服务器
      </el-button>
    </div>

    <!-- 服务器表格 -->
    <el-card shadow="never">
      <el-table :data="servers" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="名称" width="140" />
        <el-table-column prop="ip" label="IP 地址" width="140" />
        <el-table-column prop="cpu" label="CPU" width="160" />
        <el-table-column prop="memory" label="内存" width="100" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="statusType(row.status)"
              effect="dark"
              size="small"
            >
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="280">
          <template #default="{ row }">
            <el-button
              v-if="hasPermission('server', 'restart') && row.status !== 'running'"
              type="success"
              size="small"
              @click="handleRestart(row)"
            >
              🔄 重启
            </el-button>
            <el-button
              v-if="hasPermission('server', 'stop') && row.status === 'running'"
              type="warning"
              size="small"
              @click="handleStop(row)"
            >
              ⏹️ 停止
            </el-button>
            <el-button
              v-if="hasPermission('server', 'delete')"
              type="danger"
              size="small"
              @click="handleDelete(row)"
            >
              🗑️ 删除
            </el-button>
            <span v-if="!hasAnyAction(row)" class="no-action">无可用操作</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增服务器对话框 -->
    <el-dialog v-model="dialogVisible" title="新增服务器" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="如 web-03" />
        </el-form-item>
        <el-form-item label="IP 地址">
          <el-input v-model="form.ip" placeholder="如 10.0.1.12" />
        </el-form-item>
        <el-form-item label="CPU">
          <el-input v-model="form.cpu" placeholder="如 8核 Intel Xeon" />
        </el-form-item>
        <el-form-item label="内存">
          <el-input v-model="form.memory" placeholder="如 32GB" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreate">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hasPermission } from '../utils/permission'
import { getServers, createServer, deleteServer, restartServer, stopServer } from '../api'

const servers = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)

const form = reactive({
  name: '',
  ip: '',
  cpu: '',
  memory: '',
})

onMounted(() => fetchServers())

async function fetchServers() {
  loading.value = true
  try {
    const res = await getServers()
    servers.value = Array.isArray(res.data) ? res.data : []
  } catch (err) {
    ElMessage.error('加载服务器列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  form.name = ''
  form.ip = ''
  form.cpu = ''
  form.memory = ''
  dialogVisible.value = true
}

async function handleCreate() {
  if (!form.name || !form.ip) {
    ElMessage.warning('名称和 IP 不能为空')
    return
  }
  submitting.value = true
  try {
    await createServer({ ...form })
    ElMessage.success('服务器创建成功')
    dialogVisible.value = false
    await fetchServers()
  } catch (err) {
    ElMessage.error('创建失败: ' + err.message)
  } finally {
    submitting.value = false
  }
}

async function handleRestart(row) {
  try {
    await restartServer(row.id)
    ElMessage.success(`服务器 ${row.name} 重启成功`)
    await fetchServers()
  } catch (err) {
    ElMessage.error('重启失败: ' + err.message)
  }
}

async function handleStop(row) {
  try {
    await stopServer(row.id)
    ElMessage.success(`服务器 ${row.name} 已停止`)
    await fetchServers()
  } catch (err) {
    ElMessage.error('停止失败: ' + err.message)
  }
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确定要删除服务器 ${row.name} 吗?`, '确认删除', {
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await deleteServer(row.id)
    ElMessage.success('服务器已删除')
    await fetchServers()
  } catch (err) {
    ElMessage.error('删除失败: ' + err.message)
  }
}

// 检查是否有任何操作权限 (用于显示 "无可用操作")
function hasAnyAction(row) {
  return (
    (hasPermission('server', 'restart') && row.status !== 'running') ||
    (hasPermission('server', 'stop') && row.status === 'running') ||
    hasPermission('server', 'delete')
  )
}

function statusType(status) {
  const map = { running: 'success', stopped: 'info', error: 'danger' }
  return map[status] || 'info'
}

function statusText(status) {
  const map = { running: '运行中', stopped: '已停止', error: '异常' }
  return map[status] || status
}
</script>
