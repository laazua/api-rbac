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
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="角色名称" width="140" />
        <el-table-column label="可访问模块" width="200">
          <template slot-scope="{ row }">
            <el-tag v-for="m in (row.modules || [])" :key="m.id" size="mini" type="success" style="margin:2px">
              <i :class="m.icon" style="margin-right:2px" />{{ m.name }}
            </el-tag>
            <span v-if="!row.modules || row.modules.length === 0" style="color:#c0c4cc">未分配</span>
          </template>
        </el-table-column>
        <el-table-column label="关联权限" width="200">
          <template slot-scope="{ row }">
            <el-tag v-for="p in (row.permissions || [])" :key="p.id" size="mini" style="margin:2px">
              {{ p.name }}
            </el-tag>
            <span v-if="!row.permissions || row.permissions.length === 0" style="color:#c0c4cc">无</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="100" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template slot-scope="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template slot-scope="{ row }">
            <el-button v-if="hasPermission('role','update')" type="text" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="hasPermission('role','update')" type="text" icon="el-icon-setting" @click="openAssign(row)">分配模块与权限</el-button>
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

    <!-- 分配模块与权限（合并） -->
    <el-dialog
      title="分配模块与权限"
      :visible.sync="assignVisible"
      width="720px"
      top="4vh"
      @opened="onAssignDialogOpened"
    >
      <!-- 模块选择 -->
      <div class="assign-section">
        <div class="section-title">① 选择可访问的模块（控制门户卡片可见性）</div>
        <el-transfer
          ref="moduleTransfer"
          v-model="selectedModules"
          :data="moduleList"
          :titles="['可选模块', '已选模块']"
          :props="{ key: 'id', label: 'name' }"
          @change="onModuleChange"
        />
      </div>

      <!-- 权限选择（按模块分组） -->
      <div v-if="groupedPerms.length > 0" class="assign-section">
        <div class="section-title">
          ② 选择各模块下的操作权限
          <span style="font-weight:400;font-size:12px;color:#909399;margin-left:12px">
            （仅显示已选模块的权限）
          </span>
        </div>
        <div v-for="group in groupedPerms" :key="group.moduleId" class="perm-group">
          <div class="perm-group-header">
            <i :class="group.icon || 'el-icon-menu'" />
            <b>{{ group.moduleName }}</b>
            <span class="perm-group-code">({{ group.moduleCode }})</span>
            <el-button type="text" size="mini" @click="selectAllInGroup(group.moduleId)">全选</el-button>
            <el-button type="text" size="mini" @click="clearGroup(group.moduleId)">清空</el-button>
          </div>
          <el-checkbox-group v-model="selectedPerms" class="perm-checkboxes">
            <el-checkbox
              v-for="p in group.perms"
              :key="p.id"
              :label="p.id"
            >{{ p.display }}</el-checkbox>
          </el-checkbox-group>
        </div>
      </div>
      <div v-else class="assign-section">
        <div class="section-title">② 选择各模块下的操作权限</div>
        <el-empty v-if="selectedModules.length === 0"
          description="请先在①中选择模块，然后此处将显示模块下的权限" :image-size="60" />
        <el-empty v-else
          description="已选模块下暂无权限定义，请先在权限管理中创建" :image-size="60" />
      </div>

      <span slot="footer">
        <el-button @click="assignVisible = false">取消</el-button>
        <el-button type="primary" :loading="assignSubmitting" @click="handleAssignAll">
          保存（模块 + 权限）
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import {
  getRoles, createRole, updateRole, deleteRole, getRole,
  assignRolePermissions, getPermissions,
  assignRoleModules, getModules, hasPermission
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
      // 合并分配弹窗
      assignVisible: false, assignSubmitting: false,
      selectedModules: [], moduleList: [],
      selectedPerms: [],
      allPerms: [],           // 所有权限（含 module_id）
      groupedPerms: [],       // 已选模块下的权限分组
      _rawModules: []         // 原始模块数据（含 icon）
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

    // ===== 创建/编辑 =====
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
        await this.$confirm(`确定删除角色「${row.name}」吗？`, '提示', { type: 'warning' })
        await deleteRole(row.id)
        this.$message.success('删除成功')
        this.fetchData()
      } catch { /* 取消 */ }
    },

    // ===== 合并分配：模块 + 权限 =====
    async openAssign(row) {
      this.currentRoleId = row.id
      this.selectedModules = []
      this.selectedPerms = []
      this.groupedPerms = []

      try {
        // 并行：获取全量模块、全量权限、当前角色详情
        const [modsRes, permsRes, roleRes] = await Promise.all([
          getModules({ page: 1, page_size: 200 }),
          getPermissions({ page: 1, page_size: 500 }),
          getRole(row.id)
        ])

        // 保存原始模块数据（供 buildGroupedPerms 获取 icon）
        this._rawModules = modsRes.data.list || []

        // 构建模块穿梭框数据
        this.moduleList = this._rawModules.map(m => ({
          id: m.id,
          name: `${m.name} (${m.code})`
        }))

        // 预填已分配模块
        if (roleRes.data.modules) {
          this.selectedModules = roleRes.data.modules.map(m => m.id)
        }

        // 预填已分配权限
        if (roleRes.data.permissions) {
          this.selectedPerms = roleRes.data.permissions.map(p => p.id)
        }

        // 保存全量权限（含 module_id 信息）
        this.allPerms = (permsRes.data.list || []).map(p => ({
          id: p.id,
          name: p.name,
          resource: p.resource,
          action: p.action,
          module_id: p.module_id
        }))
      } catch { /* handled */ }

      this.assignVisible = true
    },
    onAssignDialogOpened() {
      // 弹窗完全打开后，根据已选模块构建权限分组
      this.buildGroupedPerms()
    },
    onModuleChange() {
      // 穿梭框变更时重新构建权限分组
      this.$nextTick(() => this.buildGroupedPerms())
    },
    buildGroupedPerms() {
      if (this.selectedModules.length === 0) {
        this.groupedPerms = []
        return
      }

      // 找出已选模块的信息
      const selectedModIds = new Set(this.selectedModules)
      const modMap = {}
      this.moduleList.forEach(m => {
        if (selectedModIds.has(m.id)) {
          // 从名称中解析出 moduleName (code) 格式
          const match = m.name.match(/^(.+?)\s*\((.+?)\)$/)
          modMap[m.id] = {
            id: m.id,
            name: match ? match[1] : m.name,
            code: match ? match[2] : '',
            icon: ''
          }
        }
      })

      // 需要从原始 module 数据获取 icon（从之前 fetch 的 modules 数据）
      // 这里从 moduleList 中无法直接获取 icon，需要在 openAssign 中保存原始数据
      // 简化处理：直接从保存的原始数据中匹配
      if (this._rawModules) {
        this._rawModules.forEach(m => {
          if (modMap[m.id]) {
            modMap[m.id].icon = m.icon || ''
          }
        })
      }

      // 按模块分组权限
      const groups = {}
      this.allPerms.forEach(p => {
        if (!p.module_id || !selectedModIds.has(p.module_id)) return
        if (!groups[p.module_id]) {
          groups[p.module_id] = []
        }
        groups[p.module_id].push({
          id: p.id,
          display: `${p.name} (${p.resource}:${p.action})`
        })
      })

      // 构建最终的 groupedPerms
      this.groupedPerms = Object.keys(groups).map(modId => {
        const id = parseInt(modId)
        const m = modMap[id] || { name: '未知模块', code: '', icon: '' }
        return {
          moduleId: id,
          moduleName: m.name,
          moduleCode: m.code,
          icon: m.icon,
          perms: groups[id]
        }
      })
    },
    selectAllInGroup(modId) {
      const ids = []
      this.allPerms.forEach(p => {
        if (p.module_id === modId) ids.push(p.id)
      })
      // 合并去重
      const set = new Set([...this.selectedPerms, ...ids])
      this.selectedPerms = Array.from(set)
    },
    clearGroup(modId) {
      const removeIds = new Set()
      this.allPerms.forEach(p => {
        if (p.module_id === modId) removeIds.add(p.id)
      })
      this.selectedPerms = this.selectedPerms.filter(id => !removeIds.has(id))
    },
    async handleAssignAll() {
      this.assignSubmitting = true
      try {
        const tasks = []
        // 仅当选择了模块时才提交模块分配
        if (this.selectedModules.length > 0) {
          tasks.push(assignRoleModules(this.currentRoleId, this.selectedModules))
        }
        // 仅当选择了权限时才提交权限分配
        if (this.selectedPerms.length > 0) {
          tasks.push(assignRolePermissions(this.currentRoleId, this.selectedPerms))
        }
        if (tasks.length === 0) {
          this.$message.warning('请至少选择一个模块或权限')
          return
        }
        await Promise.all(tasks)
        this.$message.success('分配成功')
        this.assignVisible = false
        this.fetchData()
      } catch (e) {
        const msg = (e.response && e.response.data && e.response.data.message) || e.message || '未知错误'
        this.$message.error('分配失败：' + msg)
      } finally {
        this.assignSubmitting = false
      }
    }
  }
}
</script>

<style scoped>
.assign-section {
  margin-bottom: 20px;
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid #ebeef5;
}
.perm-group {
  background: #fafafa;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 12px;
}
.perm-group-header {
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}
.perm-group-code {
  font-size: 12px;
  color: #c0c4cc;
}
.perm-checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.perm-checkboxes .el-checkbox {
  margin-right: 16px;
  margin-bottom: 4px;
}
</style>
