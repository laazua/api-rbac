<template>
  <div>
    <h2 style="margin-bottom:20px">用户管理</h2>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <el-row :gutter="12" type="flex" align="middle">
        <el-col :span="8">
          <el-input v-model="search.keyword" placeholder="搜索用户名/邮箱" clearable
            prefix-icon="el-icon-search" @clear="fetchData" @keyup.enter.native="fetchData" />
        </el-col>
        <el-col :span="4">
          <el-button type="primary" icon="el-icon-search" @click="fetchData">搜索</el-button>
          <el-button v-if="hasPermission('user','create')" type="success" icon="el-icon-plus" @click="openCreate">新增用户</el-button>
        </el-col>
      </el-row>
    </div>

    <!-- 表格 -->
    <div class="table-container">
      <el-table v-loading="loading" :data="tableData" border stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="email" label="邮箱" width="200" />
        <el-table-column label="状态" width="80">
          <template slot-scope="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template slot-scope="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" min-width="280">
          <template slot-scope="{ row }">
            <el-button v-if="hasPermission('user','update')" type="text" size="small" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="hasPermission('user','update')" type="text" size="small" icon="el-icon-key" @click="openChangePwd(row)">改密</el-button>
            <el-button v-if="hasPermission('user','update')" type="text" size="small" icon="el-icon-s-custom" @click="openAssignRoles(row)">分配角色</el-button>
            <el-button v-if="hasPermission('user','delete')" type="text" size="small" style="color:#f56c6c" icon="el-icon-delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        style="margin-top:16px;text-align:right"
        layout="total, sizes, prev, pager, next"
        :total="total"
        :page-size="search.page_size"
        :current-page="search.page"
        :page-sizes="[10, 20, 50]"
        @size-change="v => { search.page_size = v; fetchData() }"
        @current-change="v => { search.page = v; fetchData() }"
      />
    </div>

    <!-- 创建/编辑对话框 -->
    <el-dialog :title="isEdit ? '编辑用户' : '新增用户'" :visible.sync="dialogVisible" width="480px">
      <el-form ref="userForm" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="isEdit" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password" v-if="!isEdit">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="状态" prop="status" v-if="isEdit">
          <el-radio-group v-model="form.status">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </span>
    </el-dialog>

    <!-- 改密对话框 -->
    <el-dialog title="修改密码" :visible.sync="pwdVisible" width="420px">
      <el-form ref="pwdForm" :model="pwdForm" :rules="pwdRules" label-width="80px">
        <el-form-item label="旧密码" prop="old_password">
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="pwdForm.new_password" type="password" show-password />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" @click="handleChangePwd">确定</el-button>
      </span>
    </el-dialog>

    <!-- 分配角色对话框 -->
    <el-dialog title="分配角色" :visible.sync="roleVisible" width="540px">
      <el-transfer
        v-model="selectedRoles"
        :data="roleList"
        :titles="['可选角色', '已选角色']"
        :props="{ key: 'id', label: 'name' }"
      />
      <span slot="footer">
        <el-button @click="roleVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAssignRoles">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import {
  getUsers, createUser, updateUser, deleteUser, changePassword,
  assignUserRoles, getRoles, getUser, hasPermission
} from '../api'

export default {
  name: 'UserManage',
  data() {
    return {
      search: { page: 1, page_size: 10, keyword: '' },
      tableData: [], total: 0, loading: false,
      dialogVisible: false, isEdit: false, submitting: false,
      form: { username: '', password: '', email: '', status: 1 },
      formRules: {
        username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
        password: [{ required: true, min: 6, message: '密码不少于6位', trigger: 'blur' }]
      },
      currentUserId: null,
      pwdVisible: false,
      pwdForm: { old_password: '', new_password: '' },
      pwdRules: {
        old_password: [{ required: true, message: '请输入旧密码', trigger: 'blur' }],
        new_password: [{ required: true, min: 6, message: '新密码不少于6位', trigger: 'blur' }]
      },
      roleVisible: false, selectedRoles: [], roleList: []
    }
  },
  created() { this.fetchData() },
  methods: {
    hasPermission,
    async fetchData() {
      this.loading = true
      try {
        const res = await getUsers(this.search)
        const list = res.data.list || []
        list.sort((a, b) => a.id - b.id)
        this.tableData = list
        this.total = res.data.total || 0
      } finally { this.loading = false }
    },
    formatTime(t) { return t ? new Date(t).toLocaleString() : '-' },
    openCreate() {
      this.isEdit = false
      this.form = { username: '', password: '', email: '' }
      this.dialogVisible = true
    },
    openEdit(row) {
      this.isEdit = true
      this.currentUserId = row.id
      this.form = { username: row.username, email: row.email, status: row.status }
      this.dialogVisible = true
    },
    async handleSubmit() {
      this.$refs.userForm.validate(async valid => {
        if (!valid) return
        this.submitting = true
        try {
          if (this.isEdit) {
            await updateUser(this.currentUserId, { email: this.form.email, status: this.form.status })
            this.$message.success('更新成功')
          } else {
            await createUser(this.form)
            this.$message.success('创建成功')
          }
          this.dialogVisible = false
          this.fetchData()
        } finally { this.submitting = false }
      })
    },
    async handleDelete(row) {
      try {
        await this.$confirm(`确定删除用户「${row.username}」吗？`, '提示', { type: 'warning' })
        await deleteUser(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      } catch { /* 取消 */ }
    },
    openChangePwd(row) {
      this.currentUserId = row.id
      this.pwdForm = { old_password: '', new_password: '' }
      this.pwdVisible = true
    },
    handleChangePwd() {
      this.$refs.pwdForm.validate(async valid => {
        if (!valid) return
        try {
          await changePassword(this.currentUserId, this.pwdForm)
          this.$message.success('密码修改成功')
          this.pwdVisible = false
        } catch { /* handled */ }
      })
    },
    async openAssignRoles(row) {
      this.currentUserId = row.id
      this.selectedRoles = []
      try {
        const [rolesRes, userRes] = await Promise.all([
          getRoles({ page: 1, page_size: 100 }),
          getUser(row.id)
        ])
        this.roleList = (rolesRes.data.list || []).map(r => ({ id: r.id, name: r.name }))
        // 预设已选角色
        if (userRes.data.roles) {
          this.selectedRoles = userRes.data.roles.map(r => r.id)
        }
      } catch { /* handled */ }
      this.roleVisible = true
    },
    async handleAssignRoles() {
      try {
        await assignUserRoles(this.currentUserId, this.selectedRoles)
        this.$message.success('角色分配成功')
        this.roleVisible = false
        this.fetchData()
      } catch { /* handled */ }
    }
  }
}
</script>
