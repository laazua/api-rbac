<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '220px'" class="layout-aside">
      <div class="aside-header">
        <span v-if="!isCollapse" class="aside-title">🚀 运维管理</span>
        <span v-else class="aside-title-short">🚀</span>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <el-menu-item
          v-for="item in menuItems"
          :key="item.path"
          :index="item.path"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.title }}</template>
        </el-menu-item>
      </el-menu>

      <div class="aside-footer" @click="toggleCollapse">
        <el-icon><Fold v-if="!isCollapse" /><Expand v-else /></el-icon>
      </div>
    </el-aside>

    <!-- 主体区域 -->
    <el-container>
      <!-- 顶部栏 -->
      <el-header class="layout-header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentTitle">{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-tag type="success" effect="plain" size="small">
            🔗 已接入 api-rbac
          </el-tag>
          <span class="user-info" style="margin-left:12px;">
            👤 {{ username }}
          </span>
          <el-button
            text
            type="primary"
            size="small"
            style="margin-left:16px;"
            @click="$router.push('/my-permissions')"
          >
            <el-icon><Key /></el-icon> 我的权限
          </el-button>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { Monitor, Upload, Bell, Key, Fold, Expand } from '@element-plus/icons-vue'
import { getUsername } from '../utils/permission'
import { getMenuRoutes } from '../router'

const route = useRoute()

const isCollapse = ref(false)
const username = ref(getUsername())

const menuItems = computed(() => getMenuRoutes())
const activeMenu = computed(() => route.path)

const currentTitle = computed(() => {
  const item = menuItems.value.find((m) => m.path === route.path)
  return item ? item.title : ''
})

function toggleCollapse() {
  isCollapse.value = !isCollapse.value
}
</script>
