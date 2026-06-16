<template>
  <div class="deployment-manage">
    <div class="page-header">
      <h2 class="page-title">📦 发布管理</h2>
      <el-button
        v-if="hasPermission('deployment', 'execute')"
        type="primary"
        @click="showExecuteDialog"
      >
        执行发布
      </el-button>
    </div>

    <!-- 发布列表 -->
    <el-card shadow="never">
      <el-table :data="deployments" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="project" label="项目" width="160" />
        <el-table-column prop="version" label="版本" width="120" />
        <el-table-column prop="env" label="环境" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.env === 'production' ? 'danger' : 'warning'"
              size="small"
            >
              {{ row.env }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'success' ? 'success' : 'danger'"
              size="small"
            >
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="120">
          <template #default="{ row }">
            <el-button
              v-if="hasPermission('deployment', 'rollback')"
              type="warning"
              size="small"
              @click="handleRollback(row)"
            >
              ↩️ 回滚
            </el-button>
            <span v-if="!hasPermission('deployment', 'rollback')" class="no-action">
              无可用操作
            </span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 执行发布对话框 -->
    <el-dialog v-model="dialogVisible" title="执行发布" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="项目">
          <el-input v-model="form.project" placeholder="如 web-app" />
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="form.version" placeholder="如 v2.4.0" />
        </el-form-item>
        <el-form-item label="环境">
          <el-select v-model="form.env" placeholder="请选择环境" style="width: 100%">
            <el-option label="生产环境 (production)" value="production" />
            <el-option label="预发布 (staging)" value="staging" />
            <el-option label="测试环境 (testing)" value="testing" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleExecute">
          确定发布
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { hasPermission } from '../utils/permission'
import { getDeployments, executeDeployment, rollbackDeployment } from '../api'

const deployments = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)

const form = reactive({
  project: '',
  version: '',
  env: 'staging',
})

onMounted(() => fetchDeployments())

async function fetchDeployments() {
  loading.value = true
  try {
    const res = await getDeployments()
    deployments.value = Array.isArray(res.data) ? res.data : []
  } catch (err) {
    ElMessage.error('加载发布列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

function showExecuteDialog() {
  form.project = ''
  form.version = ''
  form.env = 'staging'
  dialogVisible.value = true
}

async function handleExecute() {
  if (!form.project || !form.version || !form.env) {
    ElMessage.warning('请填写完整的发布信息')
    return
  }
  submitting.value = true
  try {
    await executeDeployment({ ...form })
    ElMessage.success('发布执行成功')
    dialogVisible.value = false
    await fetchDeployments()
  } catch (err) {
    ElMessage.error('发布失败: ' + err.message)
  } finally {
    submitting.value = false
  }
}

async function handleRollback(row) {
  try {
    await ElMessageBox.confirm(
      `确定要回滚 ${row.project} 的版本 ${row.version} 吗?`,
      '确认回滚',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await rollbackDeployment(row.id)
    ElMessage.success('回滚成功')
    await fetchDeployments()
  } catch (err) {
    ElMessage.error('回滚失败: ' + err.message)
  }
}
</script>
