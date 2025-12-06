<template>
  <div>
    <h2>授权管理</h2>
    
    <div style="margin-bottom: 20px;">
      <el-button type="primary" @click="openAddDialog">添加用户</el-button>
    </div>

    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>已注册用户</span>
        </div>
      </template>
      <el-table :data="users" style="width: 100%">
        <el-table-column prop="username" label="用户名" width="180" />
        <el-table-column prop="remarks" label="备注" />
        <el-table-column label="角色" width="120">
          <template #default="scope">
            <el-tag :type="scope.row.is_admin ? 'warning' : 'info'">
                {{ scope.row.is_admin ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="180">
            <template #default="scope">
                {{ scope.row.client ? 'Client Rule' : 'User' }}
            </template>
        </el-table-column>
        <el-table-column label="状态">
          <template #default="scope">
            <el-tag :type="scope.row.disallow ? 'danger' : 'success'">
                {{ scope.row.disallow ? '禁止' : '允许' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column fixed="right" label="操作" width="160">
          <template #default="scope">
            <el-button type="primary" size="small" @click="handleEdit(scope.row)">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(scope.row.username)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '添加用户'" width="30%">
      <el-form :model="form" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="form.username" :disabled="isEdit" placeholder="用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" placeholder="密码 (不修改请留空)" type="password" show-password />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remarks" type="textarea" placeholder="用户备注信息" />
        </el-form-item>
        <el-form-item label="管理员">
            <el-switch v-model="form.is_admin" active-text="是" inactive-text="否" />
        </el-form-item>
        <el-form-item label="允许访问">
            <el-switch v-model="form.allow" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="onSubmit">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api, { type User } from '../api'

const users = ref<User[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)

const form = reactive({
  username: '',
  password: '',
  remarks: '',
  allow: true,
  is_admin: false
})

const fetchUsers = async () => {
  try {
    const res = await api.getUsers()
    users.value = res.data
  } catch (e: any) {
    ElMessage.error('获取用户列表失败: ' + e.message)
  }
}

const openAddDialog = () => {
    isEdit.value = false
    form.username = ''
    form.password = ''
    form.remarks = ''
    form.allow = true
    form.is_admin = false
    dialogVisible.value = true
}

const handleEdit = (user: User) => {
    isEdit.value = true
    form.username = user.username
    form.password = '' // Don't show password
    form.remarks = user.remarks || ''
    form.allow = !user.disallow
    form.is_admin = !!user.is_admin
    dialogVisible.value = true
}

const onSubmit = async () => {
  if (!form.username) {
    ElMessage.warning('用户名必填')
    return
  }
  if (!isEdit.value && !form.password) {
      ElMessage.warning('密码必填')
      return
  }

  try {
    if (isEdit.value) {
        await api.updateUser(form.username, form.password, form.allow, form.remarks, form.is_admin)
        ElMessage.success('用户已更新')
    } else {
        await api.addUser(form.username, form.password, form.allow, form.remarks, form.is_admin)
        ElMessage.success('用户已添加')
    }
    dialogVisible.value = false
    fetchUsers()
  } catch (e: any) {
    ElMessage.error('操作失败: ' + e.message)
  }
}

const handleDelete = (username: string) => {
  ElMessageBox.confirm(
    '确定要删除该用户吗?',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
    .then(async () => {
      try {
        await api.deleteUser(username)
        ElMessage.success('用户已删除')
        fetchUsers()
      } catch (e: any) {
        ElMessage.error('删除失败: ' + e.message)
      }
    })
    .catch(() => {})
}

onMounted(() => {
  fetchUsers()
})
</script>
