<template>
  <div>
    <h2 style="margin-bottom:20px">仪表盘</h2>

    <!-- 统计卡片 -->
    <div class="dashboard-cards">
      <el-card class="dashboard-card" shadow="hover">
        <div class="count">{{ stats.users }}</div>
        <div class="label">用户总数</div>
      </el-card>
      <el-card class="dashboard-card" shadow="hover">
        <div class="count">{{ stats.roles }}</div>
        <div class="label">角色总数</div>
      </el-card>
      <el-card class="dashboard-card" shadow="hover">
        <div class="count">{{ stats.permissions }}</div>
        <div class="label">权限总数</div>
      </el-card>
    </div>

    <!-- 快速入口 — 仅显示有权限的模块 -->
    <el-card style="border-radius:8px">
      <div slot="header"><b>快速入口</b></div>
      <el-row :gutter="16">
        <el-col :span="8">
          <el-button
            type="primary" plain style="width:100%;height:80px;font-size:16px"
            icon="el-icon-user" :disabled="!canAccessUser"
            @click="$router.push('/users')"
          >
            用户管理
          </el-button>
        </el-col>
        <el-col :span="8">
          <el-button
            type="success" plain style="width:100%;height:80px;font-size:16px"
            icon="el-icon-s-custom" :disabled="!canAccessRole"
            @click="$router.push('/roles')"
          >
            角色管理
          </el-button>
        </el-col>
        <el-col :span="8">
          <el-button
            type="warning" plain style="width:100%;height:80px;font-size:16px"
            icon="el-icon-lock" :disabled="!canAccessPerm"
            @click="$router.push('/permissions')"
          >
            权限管理
          </el-button>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script>
import { getUsers, getRoles, getPermissions, hasAnyPermission } from '../api'

export default {
  name: 'Dashboard',
  data() {
    return {
      stats: { users: '-', roles: '-', permissions: '-' },
      canAccessUser: false,
      canAccessRole: false,
      canAccessPerm: false
    }
  },
  async created() {
    this.canAccessUser = hasAnyPermission('user')
    this.canAccessRole = hasAnyPermission('role')
    this.canAccessPerm = hasAnyPermission('permission')

    const tasks = []
    if (this.canAccessUser) {
      tasks.push(getUsers({ page: 1, page_size: 1 }).then(r => { this.stats.users = r.data.total }).catch(() => {}))
    }
    if (this.canAccessRole) {
      tasks.push(getRoles({ page: 1, page_size: 1 }).then(r => { this.stats.roles = r.data.total }).catch(() => {}))
    }
    if (this.canAccessPerm) {
      tasks.push(getPermissions({ page: 1, page_size: 1 }).then(r => { this.stats.permissions = r.data.total }).catch(() => {}))
    }
    await Promise.all(tasks)
  }
}
</script>
