<template>
  <div>
    <h2>端口管理</h2>
    
    <el-card class="box-card" style="margin-bottom: 20px;">
      <template #header>
        <div class="card-header">
          <span>添加监听端口</span>
        </div>
      </template>
      <el-form :inline="true" :model="form" class="demo-form-inline">
        <el-form-item label="类型">
          <el-select v-model="form.type" placeholder="选择类型">
            <el-option label="TCP" value="tcp" />
            <el-option label="Websocket" value="ws" />
          </el-select>
        </el-form-item>
        <el-form-item label="ID">
          <el-input v-model="form.id" placeholder="ID (例如: t2)" />
        </el-form-item>
        <el-form-item label="地址">
          <el-input v-model="form.address" placeholder="地址 (例如: :1884)" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="onSubmit">添加</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>活跃端口</span>
        </div>
      </template>
      <el-table :data="listeners" style="width: 100%">
        <el-table-column prop="id" label="ID" width="180" />
        <el-table-column prop="type" label="类型" width="180" />
        <el-table-column prop="address" label="监听地址" />
        <el-table-column prop="protocol" label="协议" />
        <el-table-column fixed="right" label="操作" width="120">
          <template #default="scope">
            <el-button type="danger" size="small" @click="handleDelete(scope.row.id)">停止</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api, { type Listener } from '../api'

const listeners = ref<Listener[]>([])
const form = reactive({
  type: 'tcp',
  id: '',
  address: ''
})

const fetchListeners = async () => {
  try {
    const res = await api.getListeners()
    listeners.value = res.data
  } catch (e: any) {
    ElMessage.error('获取端口列表失败: ' + e.message)
  }
}

const onSubmit = async () => {
  if (!form.id || !form.address) {
    ElMessage.warning('ID 和地址是必填项')
    return
  }
  try {
    await api.addListener(form.type, form.id, form.address)
    ElMessage.success('监听端口已添加')
    form.id = ''
    form.address = ''
    fetchListeners()
  } catch (e: any) {
    ElMessage.error('添加失败: ' + e.message)
  }
}

const handleDelete = (id: string) => {
  ElMessageBox.confirm(
    '确定要停止该监听端口吗?',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
    .then(async () => {
      try {
        await api.deleteListener(id)
        ElMessage.success('监听端口已停止')
        fetchListeners()
      } catch (e: any) {
        ElMessage.error('停止失败: ' + e.message)
      }
    })
    .catch(() => {})
}

onMounted(() => {
  fetchListeners()
})
</script>
