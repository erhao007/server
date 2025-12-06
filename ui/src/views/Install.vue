<template>
  <div class="install-container">
    <el-card class="install-card">
      <template #header>
        <div class="card-header">
          <h2>Mochi MQTT 初始化</h2>
          <p class="subtitle">创建超级管理员账户</p>
        </div>
      </template>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px" label-position="top">
        <el-form-item label="管理员用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" prefix-icon="User" />
        </el-form-item>
        <el-form-item label="管理员密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="form.confirmPassword" type="password" placeholder="请再次输入密码" prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleInstall" style="width: 100%">
            完成初始化
          </el-button>
        </el-form-item>
      </el-form>
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
    </el-card>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import api from '../api'

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)
const errorMsg = ref('')

const form = reactive({
  username: 'admin',
  password: '',
  confirmPassword: ''
})

const validatePass2 = (_rule: any, value: any, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== form.password) {
    callback(new Error('两次输入密码不一致!'))
  } else {
    callback()
  }
}

const rules = reactive<FormRules>({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, message: '长度在 3 个字符以上', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 5, message: '长度在 5 个字符以上', trigger: 'blur' }
  ],
  confirmPassword: [
    { validator: validatePass2, trigger: 'blur' }
  ]
})

const handleInstall = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      errorMsg.value = ''
      try {
        await api.install(form.username, form.password)
        loading.value = false
        // Redirect to login or auto-login
        // For simplicity, redirect to login which will show clean state
        router.push('/login')
      } catch (err: any) {
        loading.value = false
        errorMsg.value = err.response?.data?.error || '初始化失败，请重试'
      }
    }
  })
}
</script>

<style scoped>
.install-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f0f2f5;
  background-image: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.install-card {
  width: 400px;
  border-radius: 8px;
}

.card-header {
  text-align: center;
}

.card-header h2 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.subtitle {
  margin: 5px 0 0;
  color: #909399;
  font-size: 14px;
}

.error-msg {
  color: #f56c6c;
  text-align: center;
  margin-top: 10px;
  font-size: 14px;
}
</style>
