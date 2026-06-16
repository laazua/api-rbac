# Go 运维管理系统 — api-rbac 解耦集成完整示例

> 演示 **Go (Gin) + Vue 3 (Element Plus)** 业务系统如何将 **api-rbac** 作为独立权限微服务集成，实现权限与业务**完全解耦**。

## 项目结构

```
go-ops-system/
├── main.go                       # Go Gin 后端入口 (~160 行)
├── go.mod / go.sum               # Go 模块 (replace → ../../)
├── setup_permissions.sh          # 一键初始化运维权限
├── README.md
├── internal/
│   ├── handler/
│   │   ├── auth_handler.go       # 登录/刷新/获取权限 (转发 RBAC)
│   │   ├── server_handler.go     # 服务器管理
│   │   ├── deployment_handler.go # 发布管理
│   │   └── alert_handler.go      # 告警管理
│   ├── middleware/
│   │   └── auth.go               # Token 提取 + 用户验证
│   └── model/
│       └── models.go             # 业务数据模型
└── web/                          # Vue 3 前端
    ├── index.html
    ├── package.json
    ├── vite.config.js
    └── src/
        ├── main.js               # 入口
        ├── App.vue               # 根组件
        ├── api/index.js          # Axios + API 函数
        ├── router/index.js       # 路由 + 权限导航守卫
        ├── utils/permission.js   # hasPermission / hasAnyPermission
        ├── views/
        │   ├── Login.vue         # 登录页
        │   ├── Layout.vue        # 主布局 (侧边栏 + 头部)
        │   ├── Dashboard.vue     # 仪表盘
        │   ├── ServerManage.vue  # 服务器管理
        │   ├── DeploymentManage.vue # 发布管理
        │   ├── AlertManage.vue   # 告警管理
        │   └── MyPermissions.vue # 我的权限
        └── styles/global.css     # 全局样式
```

## 核心架构

```
┌─────────────────────────────────────────────────────────────┐
│  运维管理系统 (:8081)                                         │
│  ┌──────────────────────┐   ┌─────────────────────────────┐ │
│  │  Vue 3 前端 (Vite)    │   │  Go Gin 后端                 │ │
│  │  Element Plus UI      │──→│  /api/auth/*   → 转发 RBAC  │ │
│  │  权限控制菜单/按钮     │   │  /api/servers  → 业务逻辑   │ │
│  │  localStorage 存权限   │   │  中间件: Token提取 + 鉴权   │ │
│  └──────────────────────┘   └──────────┬──────────────────┘ │
│                                        │                     │
└────────────────────────────────────────┼─────────────────────┘
                                         │ HTTP (pkg/client SDK)
                                         ▼
                              ┌─────────────────────┐
                              │  api-rbac (:8087)   │
                              │  用户/角色/权限管理   │
                              │  权限校验 API        │
                              └─────────────────────┘
```

## 设计亮点

- **三层权限控制**: 后端 API 鉴权 (ResilientGuard) → 前端按钮显隐 (hasPermission) → 前端菜单显隐 (hasAnyPermission)
- **韧性降级**: 使用 `ResilientGuard` 中间件 — 熔断器 + 本地缓存, RBAC 宕机时自动降级
- **零权限代码入侵**: 业务 Handler 中没有任何权限判断代码，鉴权完全由中间件完成
- **SDK 复用**: 直接引用项目内置的 `pkg/client` SDK (`ResilientGuard`, `RBACClient`)

## 运行步骤

### 前提条件

- Go 1.26+ (运行后端)
- Node.js 18+ (运行前端)
- api-rbac 服务已启动在 `localhost:8087`
- Python 3 (仅 setup_permissions.sh 解析 JSON 用)

### 1. 启动 api-rbac

```bash
cd /opt/codes/api-rbac
# 配置数据库后启动
go run ./cmd/server
# 首次运行会提示创建 admin 密码
```

### 2. 初始化运维权限

```bash
cd examples/go-ops-system
chmod +x setup_permissions.sh

# 使用默认 admin 密码
./setup_permissions.sh

# 或指定管理员账号密码
ADMIN_ACCOUNT=admin ADMIN_PASSWORD=YourPassword ./setup_permissions.sh
```

脚本会自动创建:

| 类型 | 名称 | 权限 |
|------|------|------|
| 权限 | 查看/创建/重启/停止/删除服务器 | `server:read/create/restart/stop/delete` |
| 权限 | 查看/执行/回滚发布 | `deployment:read/execute/rollback` |
| 权限 | 查看/确认告警 | `alert:read/ack` |
| 角色 | 运维管理员 | 全部运维权限 |
| 角色 | 运维查看者 | 仅 read 权限 |

测试账号:
- **opsadmin / 123456** (运维管理员 — 全部权限)
- **opsviewer / 123456** (运维查看者 — 仅查看)

### 3. 启动运维系统后端

```bash
cd examples/go-ops-system
go run .
# 启动在 http://0.0.0.0:8081
```

### 4. 启动前端

```bash
cd examples/go-ops-system/web
npm install
npm run dev
# 启动在 http://localhost:5173
```

### 5. 测试

访问 `http://localhost:5173`:

**运维管理员登录** (opsadmin):
- 侧边栏显示: 仪表盘、服务器管理、发布管理、告警管理、我的权限
- 服务器页: 可看到 🖥️ 新增、🔄 重启、⏹️ 停止、🗑️ 删除按钮
- 发布页: 可看到 📤 执行发布、↩️ 回滚按钮
- 告警页: 可看到 ✅ 确认按钮

**运维查看者登录** (opsviewer):
- 侧边栏显示: 仪表盘、服务器管理、发布管理、告警管理、我的权限
- 服务器页: **无操作按钮** (仅查看)
- 发布页: **无回滚按钮** (仅查看)
- 告警页: **无确认按钮** (仅查看)

**curl 测试后端鉴权**:
```bash
# 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"opsviewer","password":"123456"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

# 查看服务器 → 成功 (server:read)
curl http://localhost:8081/api/servers -H "Authorization: Bearer $TOKEN"

# 重启服务器 → 403 无权限 (缺少 server:restart)
curl -X POST http://localhost:8081/api/servers/restart \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":1}'
# {"code":1003,"message":"无权限"}

# 即使绕过前端直接调 API，后端也会拦截！
```

## 核心代码

### 路由注册 — 一行中间件完成鉴权

```go
// main.go

// GET /api/servers → server:read
serverGroup.GET("",
    client.ResilientGuard(rbacClient, client.FailModeCache, 300, "server", "read"),
    serverH.List)

// POST /api/servers/restart → server:restart
serverOps.POST("/restart",
    client.ResilientGuard(rbacClient, client.FailModeCache, 300, "server", "restart"),
    serverH.Restart)
```

### 业务 Handler — 零权限代码

```go
// server_handler.go

func (h *ServerHandler) Restart(c *gin.Context) {
    // 权限校验已由 ResilientGuard 中间件完成
    // 这里只有纯粹的业务逻辑
    var req struct { ID int `json:"id" binding:"required"` }
    c.ShouldBindJSON(&req)
    s := h.servers[req.ID]
    s.Status = "running"
    response.Success(c, gin.H{"message": "服务器 " + s.Name + " 重启成功"})
}
```

### 前端权限控制 — 两个函数统治一切

```javascript
// utils/permission.js

// 按钮显隐: v-if="hasPermission('server', 'restart')"
function hasPermission(resource, action) {
  const perms = getPermissionsMap()
  if (perms['*']?.includes('*')) return true          // *:* 超级管理
  if (perms[resource]?.includes('*')) return true     // resource:*
  if (perms['*']?.includes(action)) return true       // *:action
  return perms[resource]?.includes(action)            // 精确匹配
}

// 菜单显隐: v-if="hasAnyPermission('server')"
function hasAnyPermission(resource) {
  const perms = getPermissionsMap()
  return perms['*'] || perms[resource]?.length > 0
}
```

## 新增业务模块

假设新增 **日志管理**:

1. **在 api-rbac 创建权限**:
```bash
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"查看日志","resource":"log","action":"read"}'
```

2. **后端添加 Handler**:
```go
// internal/handler/log_handler.go
func (h *LogHandler) List(c *gin.Context) {
    response.Success(c, someLogs)
}
```

3. **注册路由**:
```go
logGroup.GET("",
    client.ResilientGuard(rbacClient, client.FailModeCache, 300, "log", "read"),
    logH.List)
```

4. **前端添加路由 + 菜单**:
```javascript
// router/index.js
{ path: 'logs', component: () => import('../views/LogManage.vue'),
  meta: { title: '日志管理', icon: 'Document', resource: 'log' } }
```

5. **创建页面组件** — 完成。不需要改任何用户/角色/权限判断代码。

## 安全最佳实践

```
✅ 前端显隐只是 UX 优化, 真正安全的是后端 API 鉴权
✅ 即使用户通过浏览器 DevTools 直接发请求, 后端也会拒绝
✅ api-rbac 的 JWT secret 只在 RBAC 服务中存储
✅ 业务系统不存储用户密码, 仅转发到 RBAC 验证
✅ 韧性中间件: RBAC 宕机时熔断 + 走缓存, 不会导致全系统 502
```
