<template>
  <div>
    <h2 style="margin-bottom:20px">角色管理</h2>

    <div class="search-bar">
      <el-row :gutter="12" type="flex" align="middle">
        <el-col :span="8">
          <el-input v-model="search.keyword" placeholder="搜索角色名称/描述" clearable
            prefix-icon="el-icon-search" @clear="fetchData" @keyup.enter.native="fetchData" />
        </el-col>
        <el-col :span="4">
          <el-button type="primary" icon="el-icon-search" @click="fetchData">搜索</el-button>
          <el-button v-if="hasPermission('role','create')" type="success" icon="el-icon-plus" @click="openCreate">新增角色</el-button>
        </el-col>
      </el-row>
    </div>

    <div class="table-container">
      <el-table v-loading="loading" :data="tableData" border stripe>
        <el-table-column prop="id" label="ID" width="60" align="center" sortable />
        <el-table-column prop="name" label="角色名称" width="160" align="left" />
        <el-table-column prop="description" label="描述" min-width="200" />
        <el-table-column label="关联权限" width="140">
          <template slot-scope="{ row }">
            <el-tag v-for="p in (row.permissions || [])" :key="p.id" size="mini" style="margin:2px">
              {{ p.name }}
            </el-tag>
            <span v-if="!row.permissions || row.permissions.length === 0" style="color:#c0c4cc">无</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template slot-scope="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" align="center">
          <template slot-scope="{ row }">
            <el-button v-if="hasPermission('role','update')" type="text" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="hasPermission('role','update')" type="text" icon="el-icon-lock" @click="openAssignPerms(row)">分配权限</el-button>
            <el-button v-if="hasPermission('role','delete')" type="text" style="color:#f56c6c" icon="el-icon-delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        style="margin-top:16px;text-align:right"
        layout="total, sizes, prev, pager, next"
        :total="total" :page-size="search.page_size" :current-page="search.page"
        :page-sizes="[10, 20, 50]"
        @size-change="v => { search.page_size = v; fetchData() }"
        @current-change="v => { search.page = v; fetchData() }"
      />
    </div>

    <!-- 创建/编辑 -->
    <el-dialog :title="isEdit ? '编辑角色' : '新增角色'" :visible.sync="dialogVisible" width="480px">
      <el-form ref="roleForm" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入角色名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </span>
    </el-dialog>

    <!-- 分配权限 -->
    <el-dialog title="分配权限" :visible.sync="permVisible" width="580px">
      <el-transfer
        v-model="selectedPerms"
        :data="permList"
        :titles="['可选权限', '已选权限']"
        :props="{ key: 'id', label: 'name' }"
      />
      <span slot="footer">
        <el-button @click="permVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAssignPerms">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import {
  getRoles, createRole, updateRole, deleteRole,
  assignRolePermissions, getPermissions, getRole, hasPermission
} from '../api'

export default {
  name: 'RoleManage',
  data() {
    return {
      search: { page: 1, page_size: 10, keyword: '' },
      tableData: [], total: 0, loading: false,
      dialogVisible: false, isEdit: false, submitting: false,
      form: { name: '', description: '' },
      formRules: { name: [{ required: true, message: '请输入角色名', trigger: 'blur' }] },
      currentRoleId: null,
      permVisible: false, selectedPerms: [], permList: []
    }
  },
  created() { this.fetchData() },
  methods: {
    hasPermission,
    async fetchData() {
      this.loading = true
      try {
        const res = await getRoles(this.search)
        const list = res.data.list || []
        list.sort((a, b) => a.id - b.id)
        this.tableData = list
        this.total = res.data.total || 0
      } finally { this.loading = false }
    },
    formatTime(t) { return t ? new Date(t).toLocaleString() : '-' },
    openCreate() {
      this.isEdit = false
      this.form = { name: '', description: '' }
      this.dialogVisible = true
    },
    openEdit(row) {
      this.isEdit = true
      this.currentRoleId = row.id
      this.form = { name: row.name, description: row.description }
      this.dialogVisible = true
    },
    async handleSubmit() {
      this.$refs.roleForm.validate(async valid => {
        if (!valid) return
        this.submitting = true
        try {
          if (this.isEdit) {
            await updateRole(this.currentRoleId, this.form)
            this.$message.success('更新成功')
          } else {
            await createRole(this.form)
            this.$message.success('创建成功')
          }
          this.dialogVisible = false
          this.fetchData()
        } finally { this.submitting = false }
      })
    },
    async handleDelete(row) {
      try {
        await this.$confirm(`确定删除角色「${row.name}」吗？关联的权限也会被清除。`, '提示', { type: 'warning' })
        await deleteRole(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      } catch { /* 取消 */ }
    },
    async openAssignPerms(row) {
      this.currentRoleId = row.id
      this.selectedPerms = []
      try {
        const [permsRes, roleRes] = await Promise.all([
          getPermissions({ page: 1, page_size: 100 }),
          getRole(row.id)
        ])
        this.permList = (permsRes.data.list || []).map(p => ({
          id: p.id,
          name: `${p.name} (${p.resource}:${p.action})`
        }))
        if (roleRes.data.permissions) {
          this.selectedPerms = roleRes.data.permissions.map(p => p.id)
        }
      } catch { /* handled */ }
      this.permVisible = true
    },
    async handleAssignPerms() {
      try {
        await assignRolePermissions(this.currentRoleId, this.selectedPerms)
        this.$message.success('权限分配成功')
        this.permVisible = false
        this.fetchData()
      } catch { /* handled */ }
    }
  }
}
</script>