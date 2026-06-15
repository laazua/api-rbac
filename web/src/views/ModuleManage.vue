<template>
  <div>
    <h2 style="margin-bottom:20px">模块管理</h2>

    <div class="search-bar">
      <el-row :gutter="12" type="flex" align="middle">
        <el-col :span="8">
          <el-input v-model="search.keyword" placeholder="搜索模块名称/编码/描述" clearable
            prefix-icon="el-icon-search" @clear="fetchData" @keyup.enter.native="fetchData" />
        </el-col>
        <el-col :span="4">
          <el-button type="primary" icon="el-icon-search" @click="fetchData">搜索</el-button>
          <el-button v-if="hasPermission('module','create')" type="success" icon="el-icon-plus" @click="openCreate">新增模块</el-button>
        </el-col>
      </el-row>
    </div>

    <div class="table-container">
      <el-table v-loading="loading" :data="tableData" border stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="模块名称" width="140">
          <template slot-scope="{ row }">
            <i :class="row.icon" style="margin-right:6px"></i>
            {{ row.name }}
          </template>
        </el-table-column>
        <el-table-column prop="code" label="编码" width="140">
          <template slot-scope="{ row }">
            <el-tag type="info" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="icon" label="图标" width="100">
          <template slot-scope="{ row }">
            <i :class="row.icon" style="font-size:20px"></i>
            <span style="margin-left:4px;font-size:12px;color:#909399">{{ row.icon }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="140" />
        <el-table-column prop="url" label="入口地址" width="180">
          <template slot-scope="{ row }">
            <span v-if="row.url" style="font-size:12px;color:#409eff">{{ row.url }}</span>
            <span v-else style="font-size:12px;color:#c0c4cc">内置路由</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column prop="status" label="状态" width="80">
          <template slot-scope="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template slot-scope="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="160">
          <template slot-scope="{ row }">
            <el-button v-if="hasPermission('module','update')" type="text" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="hasPermission('module','delete')" type="text" style="color:#f56c6c" icon="el-icon-delete" @click="handleDelete(row)">删除</el-button>
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
    <el-dialog :title="isEdit ? '编辑模块' : '新增模块'" :visible.sync="dialogVisible" width="520px">
      <el-form ref="moduleForm" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="模块名称" prop="name">
          <el-input v-model="form.name" placeholder="例：系统管理" />
        </el-form-item>
        <el-form-item label="模块编码" prop="code">
          <el-input v-model="form.code" placeholder="例：system_mgmt" />
          <span style="font-size:12px;color:#909399">唯一标识符，推荐使用小写字母+下划线</span>
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-input v-model="form.icon" placeholder="例：el-icon-setting">
            <template slot="prepend">
              <i :class="form.icon || 'el-icon-question'" />
            </template>
          </el-input>
          <span style="font-size:12px;color:#909399">Element UI 图标类名，如 el-icon-setting</span>
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="模块功能描述" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="form.sort" :min="0" :max="999" />
          <span style="margin-left:8px;font-size:12px;color:#909399">数字越小越靠前</span>
        </el-form-item>
        <el-form-item label="入口地址" prop="url">
          <el-input v-model="form.url" placeholder="例: http://localhost:8090 或 /payment" />
          <span style="font-size:12px;color:#909399">外部模块的前端入口URL，留空则使用内置路由</span>
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态" prop="status">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0"
            active-text="启用" inactive-text="禁用" />
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
  getModules, createModule, updateModule, deleteModule, hasPermission
} from '../api'

export default {
  name: 'ModuleManage',
  data() {
    return {
      search: { page: 1, page_size: 10, keyword: '' },
      tableData: [], total: 0, loading: false,
      dialogVisible: false, isEdit: false, submitting: false,
      form: { name: '', code: '', icon: '', description: '', sort: 0, url: '', status: 1 },
      formRules: {
        name: [{ required: true, message: '请输入模块名称', trigger: 'blur' }],
        code: [
          { required: true, message: '请输入模块编码', trigger: 'blur' },
          { pattern: /^[a-z][a-z0-9_]*$/, message: '编码需以小写字母开头，只含字母数字下划线', trigger: 'blur' }
        ]
      },
      currentModuleId: null
    }
  },
  created() { this.fetchData() },
  methods: {
    hasPermission,
    async fetchData() {
      this.loading = true
      try {
        const res = await getModules(this.search)
        const list = res.data.list || []
        list.sort((a, b) => a.sort - b.sort || a.id - b.id)
        this.tableData = list
        this.total = res.data.total || 0
      } finally { this.loading = false }
    },
    formatTime(t) { return t ? new Date(t).toLocaleString() : '-' },
    openCreate() {
      this.isEdit = false
      this.form = { name: '', code: '', icon: '', description: '', sort: 0, url: '', status: 1 }
      this.dialogVisible = true
    },
    openEdit(row) {
      this.isEdit = true
      this.currentModuleId = row.id
      this.form = {
        name: row.name,
        code: row.code,
        icon: row.icon,
        description: row.description,
        sort: row.sort,
        url: row.url || '',
        status: row.status
      }
      this.dialogVisible = true
    },
    async handleSubmit() {
      this.$refs.moduleForm.validate(async valid => {
        if (!valid) return
        this.submitting = true
        try {
          if (this.isEdit) {
            await updateModule(this.currentModuleId, this.form)
            this.$message.success('更新成功')
          } else {
            await createModule(this.form)
            this.$message.success('创建成功')
          }
          this.dialogVisible = false
          this.fetchData()
        } finally { this.submitting = false }
      })
    },
    async handleDelete(row) {
      try {
        await this.$confirm(`确定删除模块「${row.name}」吗？`, '提示', { type: 'warning' })
        await deleteModule(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      } catch { /* 取消 */ }
    }
  }
}
</script>
