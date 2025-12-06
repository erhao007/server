<template>
  <div>
    <h2>持久化数据管理</h2>
    <el-tabs v-model="activeTab">
      <el-tab-pane label="客户端 (Clients)" name="clients">
        <el-table :data="clients" style="width: 100%">
          <el-table-column prop="id" label="客户端 ID" width="180" />
          <el-table-column prop="remote" label="远程地址" width="180" />
          <el-table-column prop="username" label="用户名" />
          <el-table-column prop="clean" label="清除会话 (Clean Session)" />
          <el-table-column fixed="right" label="操作" width="120">
            <template #default="scope">
              <el-button type="danger" size="small" @click="handleDeleteClient(scope.row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-button style="margin-top: 10px" @click="fetchClients">刷新</el-button>
      </el-tab-pane>

      <el-tab-pane label="订阅 (Subscriptions)" name="subscriptions">
        <el-table :data="subscriptions" style="width: 100%">
          <el-table-column prop="client" label="客户端 ID" width="180" />
          <el-table-column prop="filter" label="订阅主题 (Filter)" />
          <el-table-column prop="qos" label="QoS" width="100" />
          <el-table-column fixed="right" label="操作" width="120">
            <template #default="scope">
              <el-button type="danger" size="small" @click="handleDeleteSubscription(scope.row.client, scope.row.filter)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-button style="margin-top: 10px" @click="fetchSubscriptions">刷新</el-button>
      </el-tab-pane>

      <el-tab-pane label="保留消息 (Retained)" name="retained">
        <el-table :data="retained" style="width: 100%">
          <el-table-column prop="topic_name" label="主题" />
          <el-table-column prop="client" label="发布者 ID" width="180" />
          <el-table-column fixed="right" label="操作" width="120">
            <template #default="scope">
              <el-button type="danger" size="small" @click="handleDeleteRetained(scope.row.topic_name)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-button style="margin-top: 10px" @click="fetchRetained">刷新</el-button>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const activeTab = ref('clients')
const clients = ref([])
const subscriptions = ref([])
const retained = ref([])

const fetchClients = async () => {
  try {
    const res = await api.getStoredClients()
    clients.value = res.data || []
  } catch (e: any) {
    ElMessage.error('获取客户端数据失败: ' + e.message)
  }
}

const fetchSubscriptions = async () => {
  try {
    const res = await api.getStoredSubscriptions()
    subscriptions.value = res.data || []
  } catch (e: any) {
    ElMessage.error('获取订阅数据失败: ' + e.message)
  }
}

const fetchRetained = async () => {
  try {
    const res = await api.getStoredRetained()
    retained.value = res.data || []
  } catch (e: any) {
    ElMessage.error('获取保留消息败: ' + e.message)
  }
}

const loadData = () => {
  if (activeTab.value === 'clients') fetchClients()
  if (activeTab.value === 'subscriptions') fetchSubscriptions()
  if (activeTab.value === 'retained') fetchRetained()
}

watch(activeTab, loadData)

const handleDeleteClient = (id: string) => {
  ElMessageBox.confirm('确定要删除该客户端记录吗?', '警告', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
    .then(async () => {
      await api.deleteStoredClient(id)
      ElMessage.success('已删除')
      fetchClients()
    })
}

const handleDeleteSubscription = (client: string, filter: string) => {
  ElMessageBox.confirm('确定要删除该订阅记录吗?', '警告', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
    .then(async () => {
      await api.deleteStoredSubscription(client, filter)
      ElMessage.success('已删除')
      fetchSubscriptions()
    })
}

const handleDeleteRetained = (topic: string) => {
  ElMessageBox.confirm('确定要删除该保留消息吗?', '警告', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
    .then(async () => {
      await api.deleteStoredRetained(topic)
      ElMessage.success('已删除')
      fetchRetained()
    })
}

onMounted(loadData)
</script>
