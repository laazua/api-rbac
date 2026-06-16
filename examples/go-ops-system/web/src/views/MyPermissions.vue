<template>
  <div class="my-permissions">
    <h2 class="page-title">🔑 我的权限</h2>

    <el-card shadow="never">
      <template #header>
        <span>当前用户: <strong>{{ username }}</strong> 的权限列表</span>
      </template>

      <div v-if="Object.keys(permMap).length === 0" class="empty-state">
        <el-empty description="暂无分配权限" />
      </div>

      <div v-else>
        <!-- 超级管理员标记 -->
        <el-alert
          v-if="isSuperAdmin"
          title="超级管理员 — 拥有所有权限 (*:*)"
          type="success"
          :closable="false"
          show-icon
          style="margin-bottom: 20px;"
        />

        <!-- 按资源分组展示权限 -->
        <el-row :gutter="16">
          <el-col
            v-for="(actions, resource) in permMap"
            :key="resource"
            :span="8"
            style="margin-bottom: 16px;"
          >
            <el-card shadow="hover" class="perm-card">
              <template #header>
                <div class="perm-resource">
                  <el-tag type="primary" effect="dark">{{ resource }}</el-tag>
                </div>
              </template>
              <el-space wrap>
                <el-tag
                  v-for="action in actions"
                  :key="action"
                  :type="action === '*' ? 'danger' : ''"
                  effect="plain"
                  size="small"
                >
                  {{ action }}
                </el-tag>
              </el-space>
            </el-card>
          </el-col>
        </el-row>

        <!-- JSON 原始数据 (调试用) -->
        <el-collapse style="margin-top: 20px;">
          <el-collapse-item title="📋 原始权限数据 (JSON)" name="raw">
            <pre class="perm-json">{{ JSON.stringify(permMap, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getPermissionsMap, getUsername } from '../utils/permission'
import { getPermissions as fetchPermissions } from '../api'

const username = ref(getUsername())
const permMap = ref({})

const isSuperAdmin = computed(() => {
  return permMap.value['*'] && permMap.value['*'].includes('*')
})

onMounted(async () => {
  try {
    const res = await fetchPermissions()
    permMap.value = res.data || {}
  } catch {
    // 如果 API 调用失败, 使用本地缓存
    permMap.value = getPermissionsMap()
  }
})
</script>
