<template>
  <div>
    <h2>控制台</h2>
    
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">客户端</div>
          </template>
          <div class="stat-value">{{ stats.clients_connected }} / {{ stats.clients_total }}</div>
          <div class="stat-label">在线 / 总数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">消息</div>
          </template>
          <div class="stat-value">{{ stats.messages_received }} / {{ stats.messages_sent }}</div>
          <div class="stat-label">接收 / 发送</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">订阅</div>
          </template>
          <div class="stat-value">{{ stats.subscriptions }}</div>
          <div class="stat-label">活跃订阅</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
             <div class="card-header">保留消息</div>
          </template>
          <div class="stat-value">{{ stats.retained }}</div>
          <div class="stat-label">数量</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="8">
        <el-card shadow="hover">
           <template #header>
            <div class="card-header">系统信息</div>
           </template>
           <p><strong>版本:</strong> {{ stats.version }}</p>
           <p><strong>GO 协程:</strong> {{ stats.threads }}</p>
           <p><strong>内存占用:</strong> {{ formatBytes(stats.memory_alloc) }}</p>
           <p><strong>运行时长:</strong> {{ formatUptime(stats.uptime) }}</p>
        </el-card>
      </el-col>
      
      <el-col :span="8">
        <el-card shadow="hover">
           <template #header>
            <div class="card-header">流量统计</div>
           </template>
           <p><strong>接收流量:</strong> {{ formatBytes(stats.bytes_received) }}</p>
           <p><strong>发送流量:</strong> {{ formatBytes(stats.bytes_sent) }}</p>
           <p><strong>包接收:</strong> {{ stats.packets_received }}</p>
           <p><strong>包发送:</strong> {{ stats.packets_sent }}</p>
        </el-card>
      </el-col>

      <el-col :span="8">
         <el-card shadow="hover">
           <template #header>
            <div class="card-header">会话状态</div>
           </template>
           <p><strong>最大在线:</strong> {{ stats.clients_maximum }}</p>
           <p><strong>虽离线但保留:</strong> {{ stats.clients_disconnected }}</p>
           <p><strong>飞行中消息:</strong> {{ stats.inflight }}</p>
           <p><strong>丢弃消息:</strong> {{ stats.messages_dropped }}</p>
        </el-card>
      </el-col>
    </el-row>

    <div style="margin-top: 20px; text-align: right;">
        <el-button @click="fetchStats">刷新数据</el-button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const stats = ref<any>({})
let timer: any = null

const fetchStats = async () => {
  try {
    const res = await api.getStats()
    stats.value = res.data
  } catch (e: any) {
    ElMessage.error('无法获取系统状态: ' + e.message)
  }
}

const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatUptime = (seconds: number) => {
    if (!seconds) return '0s'
    const d = Math.floor(seconds / (3600*24));
    const h = Math.floor(seconds % (3600*24) / 3600);
    const m = Math.floor(seconds % 3600 / 60);
    const s = Math.floor(seconds % 60);
    
    const dDisplay = d > 0 ? d + (d == 1 ? " 天, " : " 天, ") : "";
    const hDisplay = h > 0 ? h + (h == 1 ? " 小时, " : " 小时, ") : "";
    const mDisplay = m > 0 ? m + (m == 1 ? " 分, " : " 分, ") : "";
    const sDisplay = s > 0 ? s + (s == 1 ? " 秒" : " 秒") : "";
    return dDisplay + hDisplay + mDisplay + sDisplay;
}

onMounted(() => {
  fetchStats()
  timer = setInterval(fetchStats, 5000)
})

onUnmounted(() => {
    if (timer) clearInterval(timer)
})
</script>

<style scoped>
.stat-value {
    font-size: 24px;
    font-weight: bold;
    color: #409EFF;
}
.stat-label {
    font-size: 14px;
    color: #909399;
}
</style>
