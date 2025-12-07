<template>
  <el-container class="layout-container" style="height: 100vh">
    <el-aside width="200px" style="background-color: #545c64">
      <div class="logo">Mochi MQTT</div>
      <el-menu
        active-text-color="#ffd04b"
        background-color="#545c64"
        class="el-menu-vertical-demo"
        default-active="/"
        text-color="#fff"
        router
      >
        <el-menu-item index="/">
          <span>控制台</span>
        </el-menu-item>
        <el-menu-item index="/listeners">
          <span>端口管理</span>
        </el-menu-item>
        <el-menu-item index="/users">
          <span>授权管理</span>
        </el-menu-item>
        <el-menu-item index="/storage">
          <span>持久化</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <span>系统设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header style="text-align: right; font-size: 12px; background-color: #fff; border-bottom: 1px solid #eee; display: flex; justify-content: flex-end; align-items: center;">
        <span v-if="username" style="margin-right: 15px; font-size: 14px;">
            <i class="el-icon-user-solid"></i> {{ username }}
        </span>
        <el-dropdown @command="handleCommand">
            <span class="el-dropdown-link" style="cursor: pointer; display: flex; align-items: center;">
             设置
             <el-icon class="el-icon--right"><arrow-down /></el-icon>
            </span>
            <template #dropdown>
            <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
            </template>
        </el-dropdown>
      </el-header>
      <el-main>
        <router-view></router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'
import api from './api'

const username = computed(() => localStorage.getItem('username') || '')

const handleCommand = (command: string) => {
    if (command === 'logout') {
        api.logout()
    }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: white;
  font-weight: bold;
  font-size: 1.2rem;
  background-color: #434a50;
}
</style>
