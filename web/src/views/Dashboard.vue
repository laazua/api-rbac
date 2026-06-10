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

    <!-- 快速入口 -->
    <el-card style="border-radius:8px">
      <div slot="header"><b>快速入口</b></div>
      <el-row :gutter="16">
        <el-col :span="8">
          <el-button type="primary" plain style="width:100%;height:80px;font-size:16px"
            icon="el-icon-user" @click="$router.push('/users')">
            用户管理
          </el-button>
        </el-col>
        <el-col :span="8">
          <el-button type="success" plain style="width:100%;height:80px;font-size:16px"
            icon="el-icon-s-custom" @click="$router.push('/roles')">
            角色管理
          </el-button>
        </el-col>
        <el-col :span="8">
          <el-button type="warning" plain style="width:100%;height:80px;font-size:16px"
            icon="el-icon-lock" @click="$router.push('/permissions')">
            权限管理
          </el-button>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script>
import { getUsers, getRoles, getPermissions } from '../api'

export default {
  name: 'Dashboard',
  data() {
    return {
      stats: { users: 0, roles: 0, permissions: 0 }
    }
  },
  async created() {
    try {
      const [u, r, p] = await Promise.all([
        getUsers({ page: 1, page_size: 1 }),
        getRoles({ page: 1, page_size: 1 }),
        getPermissions({ page: 1, page_size: 1 })
      ])
      this.stats.users = u.data.total
      this.stats.roles = r.data.total
      this.stats.permissions = p.data.total
    } catch { /* 忽略 */ }
  }
}
</script>
