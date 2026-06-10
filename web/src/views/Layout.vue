<template>
  <el-container style="height: 100%">
    <!-- 侧边栏 -->
    <el-aside width="220px" class="layout-sidebar">
      <div
        style="height:60px;line-height:60px;text-align:center;color:#fff;font-size:18px;font-weight:bold;background:#1f2d3d;border-bottom:1px solid #2d3f51"
      >
        RBAC 管理系统
      </div>
      <el-menu
        :default-active="activeMenu"
        background-color="#1f2d3d"
        text-color="#bfcbd9"
        active-text-color="#409eff"
        router
      >
        <el-menu-item
          v-for="item in menuItems"
          :key="item.path"
          :index="item.path"
        >
          <i :class="item.icon" />
          <span>{{ item.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 右侧区域 -->
    <el-container>
      <!-- 顶部 Header -->
      <el-header height="60px" class="layout-header">
        <span class="logo">RBAC 权限管理系统</span>
        <el-dropdown trigger="click" @command="handleCommand">
          <span class="user-info">
            <el-avatar size="small" icon="el-icon-user-solid" />
            <span>{{ username }}</span>
            <i class="el-icon-arrow-down" />
          </span>
          <el-dropdown-menu slot="dropdown">
            <el-dropdown-item command="profile">个人中心</el-dropdown-item>
            <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
      </el-header>

      <!-- 主内容区 -->
      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script>
export default {
  name: 'Layout',
  data() {
    return {
      username: localStorage.getItem('username') || 'admin',
      menuItems: []
    }
  },
  computed: {
    activeMenu() {
      return '/' + this.$route.path.split('/')[1]
    }
  },
  created() {
    this.menuItems = this.$router.options.routes
      .find(r => r.path === '/')
      .children
      .filter(c => c.meta && c.meta.title)
      .map(c => ({ path: '/' + c.path, title: c.meta.title, icon: c.meta.icon }))
  },
  methods: {
    handleCommand(cmd) {
      if (cmd === 'logout') {
        localStorage.removeItem('token')
        localStorage.removeItem('username')
        this.$router.push('/login')
        this.$message.success('已退出登录')
      }
    }
  }
}
</script>
