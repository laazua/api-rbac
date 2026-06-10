<template>
  <div class="login-container">
    <el-card class="login-card" shadow="always">
      <div slot="header">RBAC 权限管理系统</div>
      <el-form ref="form" :model="form" :rules="rules" label-position="top">
        <el-form-item label="用户名 / 邮箱" prop="account">
          <el-input
            v-model="form.account"
            placeholder="请输入用户名或邮箱"
            prefix-icon="el-icon-user"
          />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            prefix-icon="el-icon-lock"
            show-password
            @keyup.enter.native="handleLogin"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" style="width: 100%" @click="handleLogin">
            {{ loading ? '登录中...' : '登  录' }}
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { login, getMenu } from '../api'

export default {
  name: 'Login',
  data() {
    return {
      form: { account: '', password: '' },
      rules: {
        account: [{ required: true, message: '请输入用户名或邮箱', trigger: 'blur' }],
        password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
      },
      loading: false
    }
  },
  methods: {
    handleLogin() {
      this.$refs.form.validate(async valid => {
        if (!valid) return
        this.loading = true
        try {
          const res = await login(this.form.account, this.form.password)
          localStorage.setItem('token', res.data.token)
          localStorage.setItem('username', res.data.username)
          // 获取用户权限
          try {
            const menuRes = await getMenu()
            localStorage.setItem('permissions', JSON.stringify(menuRes.data.permissions || {}))
          } catch { /* 忽略 */ }
          this.$message.success('登录成功')
          this.$router.push('/dashboard')
        } catch {
          // 错误已在拦截器中处理
        } finally {
          this.loading = false
        }
      })
    }
  }
}
</script>
