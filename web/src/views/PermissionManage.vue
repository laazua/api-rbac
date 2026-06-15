<template>
  <div>
    <h2 style="margin-bottom:20px">权限管理</h2>

    <div class="search-bar">
      <el-row :gutter="12" type="flex" align="middle">
        <el-col :span="8">
          <el-input v-model="search.keyword" placeholder="搜索权限名称/资源/操作" clearable
            prefix-icon="el-icon-search" @clear="fetchData" @keyup.enter.native="fetchData" />
        </el-col>
        <el-col :span="4">
          <el-button type="primary" icon="el-icon-search" @click="fetchData">搜索</el-button>
          <el-button v-if="hasPermission('permission','create')" type="success" icon="el-icon-plus" @click="openCreate">新增权限</el-button>
        </el-col>
      </el-row>
    </div>

    <div class="table-container">
      <el-table v-loading="loading" :data="tableData" border stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="权限名称" width="160" />
        <el-table-column prop="resource" label="资源" width="120">
          <template slot-scope="{ row }">
            <el-tag type="primary" size="small">{{ row.resource }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="操作" width="120">
          <template slot-scope="{ row }">
            <el-tag type="success" size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="180" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template slot-scope="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="160">
          <template slot-scope="{ row }">
            <el-button v-if="hasPermission('permission','update')" type="text" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="hasPermission('permission','delete')" type="text" style="color:#f56c6c" icon="el-icon-delete" @click="handleDelete(row)">删除</el-button>
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
    <el-dialog :title="isEdit ? '编辑权限' : '新增权限'" :visible.sync="dialogVisible" width="500px">
      <el-form ref="permForm" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="权限名称" prop="name">
          <el-input v-model="form.name" placeholder="例：删除用户" />
        </el-form-item>
        <el-form-item label="资源" prop="resource">
          <el-input v-model="form.resource" placeholder="例：user" />
        <!--  <span style="font-size:12px;color:#909399">"{resource}:{action}" 组合确定一个权限；"*" 为通配符</span> -->
        </el-form-item>
        <el-form-item label="操作" prop="action">
          <el-select v-model="form.action" placeholder="选择操作" style="width:100%">
            <el-option label="create (创建)" value="create" />
            <el-option label="read (读取)" value="read" />
            <el-option label="update (更新)" value="update" />
            <el-option label="delete (删除)" value="delete" />
            <el-option label="manage (管理)" value="manage" />
            <el-option label="publish (发布)" value="publish" />
            <el-option label="* (全部)" value="*" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="权限用途说明" />
          <span style="font-size:12px;color:#909399">"{resource}:{action}" 组合确定一个权限；"*" 为通配符</span>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import {
  getPermissions, createPermission, updatePermission, deletePermission, hasPermission
} from '../api'

export default {
  name: 'PermissionManage',
  data() {
    return {
      search: { page: 1, page_size: 10, keyword: '' },
      tableData: [], total: 0, loading: false,
      dialogVisible: false, isEdit: false, submitting: false,
      form: { name: '', resource: '', action: '', description: '' },
      formRules: {
        name: [{ required: true, message: '请输入权限名称', trigger: 'blur' }],
        resource: [{ required: true, message: '请输入资源标识', trigger: 'blur' }],
        action: [{ required: true, message: '请选择操作', trigger: 'change' }]
      },
      currentPermId: null
    }
  },
  created() { this.fetchData() },
  methods: {
    hasPermission,
    async fetchData() {
      this.loading = true
      try {
        const res = await getPermissions(this.search)
        const list = res.data.list || []
        list.sort((a, b) => a.id - b.id)
        this.tableData = list
        this.total = res.data.total || 0
      } finally { this.loading = false }
    },
    formatTime(t) { return t ? new Date(t).toLocaleString() : '-' },
    openCreate() {
      this.isEdit = false
      this.form = { name: '', resource: '', action: '', description: '' }
      this.dialogVisible = true
    },
    openEdit(row) {
      this.isEdit = true
      this.currentPermId = row.id
      this.form = { name: row.name, resource: row.resource, action: row.action, description: row.description }
      this.dialogVisible = true
    },
    async handleSubmit() {
      this.$refs.permForm.validate(async valid => {
        if (!valid) return
        this.submitting = true
        try {
          if (this.isEdit) {
            await updatePermission(this.currentPermId, this.form)
            this.$message.success('更新成功')
          } else {
            await createPermission(this.form)
            this.$message.success('创建成功')
          }
          this.dialogVisible = false
          this.fetchData()
        } finally { this.submitting = false }
      })
    },
    async handleDelete(row) {
      try {
        await this.$confirm(`确定删除权限「${row.name}」吗？`, '提示', { type: 'warning' })
        await deletePermission(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      } catch { /* 取消 */ }
    }
  }
}
</script>
