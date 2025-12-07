<template>
  <div class="settings-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ t('settings.title') }}</span>
        </div>
      </template>

      <el-form :model="form" label-width="120px" v-loading="loading">
        <el-divider content-position="left">{{ t('settings.network') }}</el-divider>
        <el-form-item :label="t('settings.mdns_enable')">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item :label="t('settings.mdns_name')">
          <el-input v-model="form.name" :disabled="!form.enabled" placeholder="Mochi MQTT" />
        </el-form-item>

        <el-divider content-position="left">{{ t('settings.tls') }}</el-divider>
        <el-form-item :label="t('settings.tls_enable')">
          <el-switch v-model="tlsForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('settings.tls_port')">
           <el-input v-model="tlsForm.port" :disabled="!tlsForm.enabled" placeholder=":8883" />
        </el-form-item>
        <el-form-item label="Cert (PEM)">
            <el-input v-model="tlsForm.cert" type="textarea" :rows="3" :disabled="!tlsForm.enabled" placeholder="-----BEGIN CERTIFICATE-----" />
        </el-form-item>
        <el-form-item label="Key (PEM)">
            <el-input v-model="tlsForm.key" type="textarea" :rows="3" :disabled="!tlsForm.enabled" placeholder="-----BEGIN PRIVATE KEY-----" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="saveSetings">{{ t('common.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script lang="ts" setup>
import { reactive, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

// Simple translation map for now, can be moved to i18n
const zh = {
  'settings.title': '系统设置',
  'settings.network': '局域网发现 (mDNS)',
  'settings.mdns_enable': '开启广播',
  'settings.mdns_name': '服务名称',
  'settings.tls': 'TLS 安全连接',
  'settings.tls_enable': '开启 TLS',
  'settings.tls_port': '端口',
  'common.save': '保存配置'
}

const t = (key: string) => {
  return (zh as any)[key] || key
}

const loading = ref(false)
const form = reactive({
  enabled: false,
  name: '',
  port: 1883
})
const tlsForm = reactive({
  enabled: false,
  port: ':8883',
  cert: '',
  key: ''
})

const fetchSettings = async () => {
  loading.value = true
  try {
    const res = await api.getMdnsConfig()
    form.enabled = res.data.enabled
    form.name = res.data.name
    form.port = res.data.port
    
    // TLS
    const tlsRes = await api.getTlsConfig()
    tlsForm.enabled = tlsRes.data.enabled
    tlsForm.port = tlsRes.data.port || ':8883'
    tlsForm.cert = tlsRes.data.cert
    tlsForm.key = tlsRes.data.key
  } catch (err) {
    console.error(err)
    ElMessage.error('获取设置失败')
  } finally {
    loading.value = false
  }
}

const saveSetings = async () => {
  loading.value = true
  try {
    // Save MDNS
    await api.updateMdnsConfig(form.enabled, form.name)
    // Save TLS
    await api.updateTlsConfig(tlsForm.enabled, tlsForm.port, tlsForm.cert, tlsForm.key)
    
    ElMessage.success('配置已更新')
    // Refresh to get potentially updated/masked values
    await fetchSettings()
  } catch (err: any) {
    console.error(err)
    ElMessage.error('更新失败: ' + (err.response?.data?.error || err.message))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSettings()
})
</script>

<style scoped>
.settings-container {
  max-width: 800px;
}
</style>
