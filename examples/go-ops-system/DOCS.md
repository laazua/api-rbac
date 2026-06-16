# api-rbac 模块扩展完整指南

> 以「运维管理系统 (go-ops-system)」为例，详细说明如何在 api-rbac 中扩展一个新的业务模块，实现**统一门户入口 + 单点登录 + 细粒度权限控制**。

---

## 目录

1. [架构概览](#1-架构概览)
2. [核心概念](#2-核心概念)
3. [完整接入步骤](#3-完整接入步骤)
   - [3.1 创建权限 (Permission)](#31-创建权限-permission)
   - [3.2 创建角色并绑定权限 (Role)](#32-创建角色并绑定权限-role)
   - [3.3 创建模块 (Module)](#33-创建模块-module)
   - [3.4 将模块绑定到角色 (Role-Module)](#34-将模块绑定到角色-role-module)
   - [3.5 创建用户并分配角色 (User-Role)](#35-创建用户并分配角色-user-role)
4. [授权数据流](#4-授权数据流)
   - [4.1 登录与权限加载](#41-登录与权限加载)
   - [4.2 门户模块可见性](#42-门户模块可见性)
   - [4.3 模块入口与 Token 传递](#43-模块入口与-token-传递)
   - [4.4 业务系统接收 Token](#44-业务系统接收-token)
   - [4.5 业务系统 API 鉴权](#45-业务系统-api-鉴权)
5. [业务系统开发模板](#5-业务系统开发模板)
   - [5.1 后端必备要素](#51-后端必备要素)
   - [5.2 前端必备要素](#52-前端必备要素)
6. [前后端权限控制](#6-前后端权限控制)
   - [6.1 三层权限模型](#61-三层权限模型)
   - [6.2 后端鉴权代码](#62-后端鉴权代码)
   - [6.3 前端权限函数](#63-前端权限函数)
7. [常见问题排查](#7-常见问题排查)

---

## 1. 架构概览

```
┌──────────────────────────────────────────────────────────────────┐
│                    api-rbac (权限管理中心)                         │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ 用户管理  │  │ 角色管理  │  │ 权限管理  │  │ 模块管理  │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ 前端门户 (Portal.vue)                                      │    │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐                   │    │
│  │  │ 模块A    │  │ 模块B    │  │ 运维管理 │  ← 按角色可见     │    │
│  │  └─────────┘  └─────────┘  └────┬────┘                   │    │
│  └─────────────────────────────────┼─────────────────────────┘    │
│                                    │ 点击 → ModuleFrame (iframe)  │
└────────────────────────────────────┼──────────────────────────────┘
                                     │ ?rbac_token=<jwt>
                                     │ postMessage({type:'RBAC_TOKEN'})
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│              运维管理系统 (独立服务 :8083)                         │
│                                                                  │
│  ┌─────────────────────┐   ┌────────────────────────────────┐    │
│  │ Vue 3 前端           │   │ Go Gin 后端                     │    │
│  │ • 接收 rbac_token    │──→│ • /api/auth/permissions → RBAC │    │
│  │ • hasPermission()    │   │ • /api/servers → 权限中间件     │    │
│  │ • 菜单/按钮动态显隐   │   │ • ResilientGuard 韧性鉴权      │    │
│  └─────────────────────┘   └────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

**关键原则**：
- api-rbac 负责**谁有什么权限**（身份认证 + 权限校验）
- 业务系统负责**业务数据和逻辑**（服务器管理、发布、告警等）
- 通过 **JWT Token** 串联两端，业务系统不存储密码、不管理权限

---

## 2. 核心概念

在 api-rbac 中，权限体系由 5 个核心实体组成：

```
用户 (User) ──多对多──→ 角色 (Role) ──多对多──→ 权限 (Permission)
                                          └──多对多──→ 模块 (Module)
```

| 实体 | 说明 | 示例 |
|------|------|------|
| **User** | 登录用户 | `opsadmin` |
| **Role** | 权限的集合 | `运维管理员` |
| **Permission** | 资源+操作 | `server:read`, `server:restart` |
| **Module** | 业务模块入口 | `运维管理系统 (code=ops, url=http://...)` |

**授权链路**：给用户分配角色 → 角色拥有权限和模块 → 用户自动获得对应权限和模块可见性。

---

## 3. 完整接入步骤

> 以下操作通过 api-rbac 的 REST API 完成，可以手动调用或使用 `setup_permissions.sh` 自动化。

### 3.1 创建权限 (Permission)

权限由 `resource:action` 组成，是系统中最小粒度的权限单元。

```bash
RBAC_URL="http://localhost:8087/api/v1"
ADMIN_TOKEN="<管理员登录获取的token>"

# 创建 10 个运维权限
# 服务器管理
curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"查看服务器","resource":"server","action":"read","description":"查看服务器列表和详情"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"创建服务器","resource":"server","action":"create","description":"创建新服务器"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"重启服务器","resource":"server","action":"restart","description":"重启指定服务器"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"停止服务器","resource":"server","action":"stop","description":"停止指定服务器"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"删除服务器","resource":"server","action":"delete","description":"删除指定服务器"}'

# 发布管理
curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"查看发布","resource":"deployment","action":"read","description":"查看发布记录"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"执行发布","resource":"deployment","action":"execute","description":"执行新的发布任务"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"回滚发布","resource":"deployment","action":"rollback","description":"回滚到上一版本"}'

# 告警管理
curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"查看告警","resource":"alert","action":"read","description":"查看告警列表"}'

curl -X POST "$RBAC_URL/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"确认告警","resource":"alert","action":"ack","description":"确认/关闭告警"}'
```

**命名规范**：
- `resource`: 小写英文，下划线分隔（如 `server_log`, `k8s_cluster`）
- `action`: CRUD 动词（`read/create/update/delete`）或自定义动词（`restart/execute/rollback/ack`）
- 通配符：`*:*` = 超级管理员，`server:*` = 服务器所有操作，`*:read` = 所有资源的读操作

### 3.2 创建角色并绑定权限 (Role)

角色是权限的集合，代表一类用户的职能。

```bash
# 3.2a 获取所有运维权限的 ID
ALL_PERMS=$(curl -s "$RBAC_URL/permissions?page=1&page_size=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
OPS_PERM_IDS=$(echo "$ALL_PERMS" | python3 -c "
import sys, json
perms = json.load(sys.stdin)['data']['list']
ids = [str(p['id']) for p in perms if p['resource'] in ('server','deployment','alert')]
print(','.join(ids))
")

# 3.2b 创建「运维管理员」角色 — 拥有全部运维权限
curl -X POST "$RBAC_URL/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"运维管理员","description":"拥有运维系统全部权限"}'

# 记下返回的角色 ID，假设是 3
ADMIN_ROLE_ID=3

# 绑定权限到角色
curl -X POST "$RBAC_URL/roles/$ADMIN_ROLE_ID/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"permission_ids\":[$OPS_PERM_IDS]}"

# 3.2c 创建「运维查看者」角色 — 仅查看权限
curl -X POST "$RBAC_URL/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"运维查看者","description":"仅有运维系统查看权限"}'

VIEWER_ROLE_ID=4
READ_IDS=$(echo "$ALL_PERMS" | python3 -c "
import sys, json
perms = json.load(sys.stdin)['data']['list']
ids = [str(p['id']) for p in perms if p['resource'] in ('server','deployment','alert') and p['action']=='read']
print(','.join(ids))
")
curl -X POST "$RBAC_URL/roles/$VIEWER_ROLE_ID/permissions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"permission_ids\":[$READ_IDS]}"
```

### 3.3 创建模块 (Module)

模块代表 api-rbac 门户上的一个应用入口卡片。**这是用户能看到模块的关键**。

```bash
OPS_URL="http://192.168.165.89:8083"  # 运维系统的访问地址

curl -X POST "$RBAC_URL/modules" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"运维管理系统\",
    \"code\": \"ops\",
    \"url\": \"$OPS_URL\",
    \"icon\": \"🚀\",
    \"description\": \"运维管理系统 (Go+Vue)\",
    \"sort\": 10
  }"
```

| 字段 | 说明 | 注意事项 |
|------|------|----------|
| `name` | 模块显示名称 | 门户卡片标题 |
| `code` | 模块唯一标识 | 英文字母+下划线，路由用 |
| `url` | 模块入口地址 | **必须填写**浏览器可访问的地址（非 localhost） |
| `icon` | 图标 | 支持 emoji / CSS类 / 图片URL |
| `sort` | 排序 | 数值越小越靠前 |

> ⚠️ **`url` 字段至关重要**：这是 iframe 的 `src` 地址。如果你通过另一台电脑访问，`localhost` 是行不通的，需要用实际的 IP 地址。

### 3.4 将模块绑定到角色 (Role-Module)

**这是最容易漏掉的一步！** 权限只决定用户能做什么操作，模块绑定才决定用户在门户能看到哪些模块卡片。

```bash
# 获取模块 ID
MOD_ID=$(curl -s "$RBAC_URL/modules?page=1&page_size=100" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; [print(m['id']) for m in json.load(sys.stdin)['data']['list'] if m['code']=='ops']")

# 绑定到运维管理员角色
curl -X POST "$RBAC_URL/roles/$ADMIN_ROLE_ID/modules" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"module_ids\":[$MOD_ID]}"

# 绑定到运维查看者角色
curl -X POST "$RBAC_URL/roles/$VIEWER_ROLE_ID/modules" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"module_ids\":[$MOD_ID]}"
```

**完整的关系链**：
```
用户 (opsadmin)
  └── 角色 (运维管理员)
        ├── 权限 (server:read, server:restart, ...)  ← 控制能做什么
        └── 模块 (运维管理系统)                       ← 控制门户能看什么
```

### 3.5 创建用户并分配角色 (User-Role)

```bash
# 创建运维管理员用户
curl -X POST "$RBAC_URL/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"opsadmin","password":"123456","email":"opsadmin@example.com"}'

# 获取用户 ID
USER_ID=$(curl -s "$RBAC_URL/users?keyword=opsadmin" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  python3 -c "import sys,json; [print(u['id']) for u in json.load(sys.stdin)['data']['list'] if u['username']=='opsadmin']")

# 分配角色
curl -X POST "$RBAC_URL/users/$USER_ID/roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"role_ids\":[$ADMIN_ROLE_ID]}"
```

---

## 4. 授权数据流

### 4.1 登录与权限加载

```
用户浏览器                    api-rbac 前端                   api-rbac 后端
    │                             │                              │
    │ POST /api/auth/login        │                              │
    │ {account, password} ───────→│ POST /api/v1/auth/login ────→│ 验证密码
    │                             │                              │ 签发 JWT
    │                             │← {token, refresh_token} ────│
    │                             │                              │
    │                             │ GET /api/v1/auth/menu ──────→│ 查询用户权限
    │                             │← {server:[read,restart],     │
    │                             │   deployment:[read],         │
    │                             │   alert:[read,ack]} ────────│
    │                             │                              │
    │                             │ GET /api/v1/auth/modules ───→│ 查询用户模块
    │                             │← [{name:"运维管理系统",       │
    │                             │    code:"ops",url:"..."}] ───│
    │                             │                              │
    │  登录成功，跳转门户          │                              │
    │← 看到"运维管理系统"卡片      │                              │
```

### 4.2 门户模块可见性

`Portal.vue` 调用 `GET /api/v1/auth/modules` 获取当前用户可见的模块列表。api-rbac 后端通过两步查询决定返回哪些模块：

1. **角色-模块绑定**：查询用户所有角色 → 查询角色绑定的模块 (`role_modules` 表)
2. **权限-模块关联**：查询用户权限中 `module_id` 不为空的，找出对应模块

只有满足以上任一条件的模块，才会在门户页面显示。

### 4.3 模块入口与 Token 传递

当用户点击模块卡片时，`ModuleFrame.vue` 通过 **两种方式** 传递 Token：

```javascript
// ModuleFrame.vue 核心逻辑
const token = localStorage.getItem('token')

// 方式1: URL 参数 (iframe src)
this.iframeSrc = `${moduleUrl}?rbac_token=${encodeURIComponent(token)}`

// 方式2: postMessage (iframe onload 后备选)
iframe.contentWindow.postMessage({
  type: 'RBAC_TOKEN',
  token: token,
  username: localStorage.getItem('username')
}, '*')
```

### 4.4 业务系统接收 Token

业务系统前端需要同时处理两种 Token 来源：

```javascript
// main.js — 应用初始化时
async function init() {
  // 1. 从 URL 参数获取 (优先级最高, 在 iframe src 中就带上了)
  const urlParams = new URLSearchParams(window.location.search)
  const urlToken = urlParams.get('rbac_token')
  if (urlToken) {
    localStorage.setItem('ops_token', urlToken)
  }

  // 2. 监听 postMessage (iframe onload 后父页面会发送)
  window.addEventListener('message', (event) => {
    if (event.data?.type === 'RBAC_TOKEN') {
      localStorage.setItem('ops_token', event.data.token)
      localStorage.setItem('ops_username', event.data.username)
      // 重新引导: 验证 token + 获取权限
      bootstrapToken()
    }
  })

  // 3. 验证 token 有效性 + 拉取用户权限
  await bootstrapToken()
}

async function bootstrapToken() {
  const token = localStorage.getItem('ops_token')
  if (!token) return

  // 调用业务系统后端, 后端转发到 api-rbac 验证
  const res = await fetch('/api/auth/permissions', {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  const perms = await res.json()
  // 存储权限: { "server": ["read","restart"], ... }
  localStorage.setItem('ops_permissions', JSON.stringify(perms.data))
}
```

### 4.5 业务系统 API 鉴权

每次业务请求的完整鉴权链：

```
浏览器 (iframe)          业务系统后端 (:8083)            api-rbac (:8087)
    │                         │                              │
    │ POST /api/servers/restart                               │
    │ Authorization: Bearer xxxx                              │
    │ {id:1} ─────────────────→│                              │
    │                         │ ExtractUserInfo 中间件         │
    │                         │ 提取 Token                    │
    │                         │                              │
    │                         │ ResilientGuard 中间件          │
    │                         │ POST /auth/check ────────────→│ 解析 JWT
    │                         │ {resource:"server",          │ 加载用户权限
    │                         │  action:"restart"}            │ 匹配: server:restart
    │                         │← {allowed: true} ────────────│
    │                         │                              │
    │                         │ ✅ 鉴权通过, 执行业务逻辑      │
    │← {code:0, message:"重启成功"}                           │
```

**韧性设计**：使用 `ResilientGuard` 中间件实现：
- **熔断器**：连续 5 次调用 RBAC 失败 → 熔断 30 秒
- **本地缓存**：5 分钟 TTL，熔断期间走缓存
- **降级模式**：`FailModeCache` — RBAC 彻底宕机时用缓存支撑

---

## 5. 业务系统开发模板

### 5.1 后端必备要素

```go
// 1. 初始化 RBAC Client
rbacClient := client.NewRBACClient("http://localhost:8087/api/v1")

// 2. Token 提取中间件 — 验证身份, 注入用户信息
auth := r.Group("/api")
auth.Use(middleware.ExtractUserInfo(rbacClient))
{
    // 获取用户权限 (供前端初始化)
    auth.GET("/auth/permissions", authH.GetPermissions)
}

// 3. 权限校验中间件 — 使用 ResilientGuard (熔断+缓存)
serverGroup.GET("",
    client.ResilientGuard(rbacClient, client.FailModeCache, 300, "server", "read"),
    serverH.List)
serverGroup.POST("/restart",
    client.ResilientGuard(rbacClient, client.FailModeCache, 300, "server", "restart"),
    serverH.Restart)

// 4. 业务 Handler — 零权限代码
func (h *ServerHandler) Restart(c *gin.Context) {
    // 权限已由中间件校验, 这里只有纯业务逻辑
    s := h.servers[req.ID]
    s.Status = "running"
    response.Success(c, gin.H{"message": "服务器 " + s.Name + " 重启成功"})
}
```

### 5.2 前端必备要素

```javascript
// 1. 接收 Token (main.js)
const urlToken = new URLSearchParams(window.location.search).get('rbac_token')
window.addEventListener('message', (e) => {
  if (e.data?.type === 'RBAC_TOKEN') saveToken(e.data.token)
})

// 2. 权限辅助函数 (utils/permission.js)
function hasPermission(resource, action) {
  const p = getPermissionsMap()  // 从 localStorage 读取
  if (p['*']?.includes('*')) return true
  if (p[resource]?.includes('*')) return true
  if (p['*']?.includes(action)) return true
  return p[resource]?.includes(action)
}

function hasAnyPermission(resource) {
  const p = getPermissionsMap()
  return p['*'] || p[resource]?.length > 0
}

// 3. 路由权限守卫 (router/index.js)
{ path: 'servers', component: ServerManage,
  meta: { title: '服务器管理', resource: 'server' } }

// 4. 菜单动态过滤
menuItems = routes.filter(r =>
  r.meta.resource === null || hasAnyPermission(r.meta.resource)
)

// 5. 按钮条件渲染
<el-button v-if="hasPermission('server', 'restart')">重启</el-button>
```

---

## 6. 前后端权限控制

### 6.1 三层权限模型

```
┌─────────────────────────────────────────────────────────────────┐
│  第三层: 后端 API 鉴权 (安全底线, 不可绕过)                        │
│  ───────────────────────────────────                            │
│  ResilientGuard("server", "restart")                            │
│  → api-rbac /auth/check → allowed:false → 返回 403              │
│  即使绕过前端直接发请求, 后端也会拒绝                               │
├─────────────────────────────────────────────────────────────────┤
│  第二层: 前端按钮显隐 (用户体验, 减少无效操作)                      │
│  ───────────────────────────────────                            │
│  v-if="hasPermission('server', 'restart')"                      │
│  → 没有权限就不渲染按钮, 用户看不到也无法点击                       │
├─────────────────────────────────────────────────────────────────┤
│  第一层: 前端菜单显隐 (导航级别控制)                               │
│  ───────────────────────────────                                │
│  hasAnyPermission('server')                                     │
│  → 完全没有服务器权限时, 左侧导航不显示"服务器管理"                 │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 后端鉴权代码

```go
// Go Gin 中使用 ResilientGuard (生产级韧性中间件)
import "github.com/laazua/api-rbac/pkg/client"

// 参数: (rbacClient, failMode, cacheTTLSec, resource, action)
serverGroup.POST("/restart",
    client.ResilientGuard(rbacClient, client.FailModeCache, 300, "server", "restart"),
    serverH.Restart)

// FailModeCache: RBAC 宕机时降级使用本地缓存 (高可用)
// FailModeDeny:  RBAC 宕机时拒绝所有请求 (安全优先)
```

**鉴权结果对应**：

| 场景 | HTTP 状态 | 说明 |
|------|-----------|------|
| 无 Token | 401 | 未认证 |
| 有 Token, 无权限 | 403 | 已认证但无权操作 |
| 有 Token, 有权限 | 放行 | 执行业务逻辑 |
| RBAC 宕机, 缓存命中 | 放行 | 韧性降级 |
| RBAC 宕机, 缓存未命中 | 502 | 服务不可用 |

### 6.3 前端权限函数

整个前端只有 **两个核心函数**，权限数据来自 `localStorage`：

```javascript
// hasPermission(resource, action) — 精确权限检查
hasPermission('server', 'restart')  // → true/false

// hasAnyPermission(resource) — 资源级检查 (菜单用)
hasAnyPermission('server')          // → true/false
```

**通配符支持**：
- `*:*` → 超级管理员，一切通过
- `server:*` → 服务器的所有操作都通过
- `*:read` → 所有资源的读取操作都通过
- `server:restart` → 精确匹配

---

## 7. 常见问题排查

### Q1: 登录后门户看不到模块卡片

**原因**: 模块没有绑定到用户的角色。

**排查**:
1. 确认模块已创建: `GET /api/v1/modules` 检查模块存在
2. 确认模块已绑定到角色: `GET /api/v1/roles/:id` 检查 `modules` 字段
3. 确认用户有该角色: `GET /api/v1/users/:id` 检查 `roles` 字段

**解决**: 重新运行 `setup_permissions.sh`，或手动调用绑定 API：
```bash
curl -X POST "$RBAC_URL/roles/$ROLE_ID/modules" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"module_ids":[$MOD_ID]}'
```

> 这是最容易遗漏的步骤。权限 ≠ 模块可见性，两者独立，**需要分别绑定**。

### Q2: 点击模块卡片后 iframe 显示"无法连接"

**原因**: 模块的 `url` 字段配置不正确。

**排查**:
1. 确认业务系统已启动: `curl <模块URL>/health`
2. 确认 URL 是浏览器可访问的地址（不能用 `localhost`，如果浏览器不在同一台机器）
3. 检查防火墙/网络策略

**解决**: 在模块管理中更新 URL 为实际 IP 地址，如 `http://192.168.165.89:8083`。

### Q3: iframe 加载成功但页面卡在"正在连接 api-rbac"

**原因**: 业务系统前端的 Token 引导流程失败。

**排查**:
1. 打开浏览器 DevTools → Network 面板
2. 检查 iframe 的 URL 是否带有 `?rbac_token=xxx`
3. 检查 `/api/auth/permissions` 请求是否返回 200
4. 检查 Console 是否有 postMessage 事件日志

**解决**:
- 确保业务后端 `/api/auth/permissions` 端点正常工作
- 确保 CORS 中间件允许跨域请求
- 检查 `localStorage` 中 `ops_token` 和 `ops_permissions` 是否正确写入

### Q4: 操作按钮不显示

**原因**: 用户没有对应的操作权限。

**排查**:
1. 在 RBAC 管理后台查看用户的角色
2. 在 RBAC 管理后台查看角色绑定的权限
3. 检查权限的 `resource` 和 `action` 是否与前端代码中的拼写完全一致

**示例**: 前端调用 `hasPermission('server', 'restart')` → RBAC 中的权限名是 `resource=server, action=restart`（注意大小写和拼写）。

### Q5: 绕过前端直接 curl 调 API 返回 403

**这说明后端鉴权正常工作**！前端按钮隐藏只是 UX 优化，真正的安全防线在后端。

```bash
# opsviewer 只有 server:read，没有 server:restart
curl -X POST http://localhost:8083/api/servers/restart \
  -H "Authorization: Bearer $VIEWER_TOKEN" \
  -d '{"id":1}'
# → {"code":1003,"message":"无权限"}  ← 预期行为
```

### Q6: 登录提示"请求过于频繁"

**原因**: api-rbac 对登录接口做了限流（每个 IP 每分钟 5 次）。

**解决**: 等待 1 分钟后重试。限流窗口会自动滑动恢复。

### Q7: 新增了权限但用户重新登录后仍无效果

**原因**: 权限缓存未失效。

**解决**: 当修改了用户的角色或角色的权限时，api-rbac 会自动清除该用户的权限缓存。但如果 Redis 不可用且走内存缓存，重启 api-rbac 服务可清除缓存。

---

## 附录: 完整的数据关系图

```
┌──────────┐     user_roles      ┌──────────────┐   role_permissions   ┌──────────────┐
│  users   │═════════════════════│    roles     │══════════════════════│ permissions  │
├──────────┤                     ├──────────────┤                      ├──────────────┤
│ id       │                     │ id           │                      │ id           │
│ username │                     │ name         │                      │ name         │
│ password │                     │ description  │                      │ resource     │
│ status   │                     └──────┬───────┘                      │ action       │
└──────────┘                            │                              │ module_id ───┐
                                        │ role_modules                 └──────────────┘│
                                        │                                              │
                                        ▼                                              │
                                  ┌──────────────┐                                    │
                                  │   modules    │←───────────────────────────────────┘
                                  ├──────────────┤
                                  │ id           │
                                  │ name         │   模块决定门户可见性
                                  │ code         │   权限决定操作能力
                                  │ url          │   两者独立管理
                                  │ icon         │
                                  │ sort         │
                                  │ status       │
                                  └──────────────┘
```

- **user_roles**: 用户 ↔ 角色 (多对多)
- **role_permissions**: 角色 ↔ 权限 (多对多) — 控制**能做什么**
- **role_modules**: 角色 ↔ 模块 (多对多) — 控制**能看到什么**
- **permission.module_id**: 权限 ↔ 模块 (多对一) — 权限可选的模块归属

**通用规则**: 要新增一个业务模块，需要在 api-rbac 中创建的实体及顺序：

```
1. 权限 (Permission)     — 定义 resource:action
2. 角色 (Role)           — 聚合权限
3. 角色-权限绑定         — role_permissions
4. 模块 (Module)         — 定义入口
5. 角色-模块绑定         — role_modules  ← 容易遗漏!
6. 用户 (User)           — 创建用户
7. 用户-角色绑定         — user_roles
```
