<template>
  <div>
    <h2 style="margin-bottom:20px">系统概览</h2>

    <!-- 统计卡片 -->
    <el-row :gutter="16">
      <el-col :xs="24" :sm="12" :md="6" style="margin-bottom:16px">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background:linear-gradient(135deg,#409eff,#337ecc)">
            <i class="el-icon-user" />
          </div>
          <div class="stat-body">
            <div class="stat-num">{{ stats.users }}</div>
            <div class="stat-label">用户总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6" style="margin-bottom:16px">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background:linear-gradient(135deg,#f093fb,#f5576c)">
            <i class="el-icon-s-custom" />
          </div>
          <div class="stat-body">
            <div class="stat-num">{{ stats.roles }}</div>
            <div class="stat-label">角色总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6" style="margin-bottom:16px">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background:linear-gradient(135deg,#4facfe,#00f2fe)">
            <i class="el-icon-lock" />
          </div>
          <div class="stat-body">
            <div class="stat-num">{{ stats.permissions }}</div>
            <div class="stat-label">权限总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6" style="margin-bottom:16px">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background:linear-gradient(135deg,#43e97b,#38f9d7)">
            <i class="el-icon-s-grid" />
          </div>
          <div class="stat-body">
            <div class="stat-num">{{ stats.modules }}</div>
            <div class="stat-label">模块总数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快捷入口 -->
    <el-card style="margin-top:20px;border-radius:8px">
      <div slot="header"><b>快捷入口</b></div>
      <el-row :gutter="12">
        <el-col :span="6">
          <el-button type="primary" plain style="width:100%;height:72px;font-size:15px"
            icon="el-icon-user" :disabled="!can('user')"
            @click="$router.push('/users')">用户管理</el-button>
        </el-col>
        <el-col :span="6">
          <el-button type="success" plain style="width:100%;height:72px;font-size:15px"
            icon="el-icon-s-custom" :disabled="!can('role')"
            @click="$router.push('/roles')">角色管理</el-button>
        </el-col>
        <el-col :span="6">
          <el-button type="warning" plain style="width:100%;height:72px;font-size:15px"
            icon="el-icon-lock" :disabled="!can('permission')"
            @click="$router.push('/permissions')">权限管理</el-button>
        </el-col>
        <el-col :span="6">
          <el-button style="width:100%;height:72px;font-size:15px"
            icon="el-icon-s-grid" :disabled="!can('module')"
            @click="$router.push('/modules')">模块管理</el-button>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script>
import { getUsers, getRoles, getPermissions, getModules, hasAnyPermission } from '../api'

export default {
  name: 'Dashboard',
  data() {
    return {
      stats: { users: '-', roles: '-', permissions: '-', modules: '-' }
    }
  },
  async created() {
    const tasks = []
    if (hasAnyPermission('user')) {
      tasks.push(getUsers({ page:1, page_size:1 }).then(r => { this.stats.users = r.data.total }).catch(()=>{}))
    }
    if (hasAnyPermission('role')) {
      tasks.push(getRoles({ page:1, page_size:1 }).then(r => { this.stats.roles = r.data.total }).catch(()=>{}))
    }
    if (hasAnyPermission('permission')) {
      tasks.push(getPermissions({ page:1, page_size:1 }).then(r => { this.stats.permissions = r.data.total }).catch(()=>{}))
    }
    if (hasAnyPermission('module')) {
      tasks.push(getModules({ page:1, page_size:1 }).then(r => { this.stats.modules = r.data.total }).catch(()=>{}))
    }
    await Promise.all(tasks)
  },
  methods: {
    can(res) { return hasAnyPermission(res) }
  }
}
</script>

<style scoped>
.stat-card {
  border-radius: 8px;
  cursor: default;
}
.stat-card .el-card__body {
  display: flex;
  align-items: center;
  padding: 20px;
}
.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
}
.stat-icon i {
  font-size: 24px;
  color: #fff;
}
.stat-body { flex: 1; }
.stat-num { font-size: 26px; font-weight: bold; color: #303133; line-height: 1.2; }
.stat-label { font-size: 13px; color: #909399; margin-top: 2px; }
</style>
