# RBAC 统一权限管理系统 — 完整操作手册

> 最后更新: 2026-06-15

---

## 目录

- [系统架构](#系统架构)
- [核心概念](#核心概念)
- [快速入门](#快速入门)
- [页面操作指南](#页面操作指南)
- [外部模块集成](#外部模块集成)
- [完整 API 参考](#完整-api-参考)
- [SDK 与中间件](#sdk-与中间件)
- [配置说明](#配置说明)
- [常见场景示例](#常见场景示例)

---

## 系统架构

```
                          ┌─────────────────┐
                          │   登录页 /login   │
                          └────────┬────────┘
                                   │ 登录成功后
                                   ▼
                          ┌─────────────────┐
                          │  模块门户 /portal │  ← 卡片网格，仅显示有权限的模块
                          └──┬──────┬──────┬┘
                             │      │      │
                   点击「系统管理」  │      点击其他模块
                             ▼      │      ▼
                   ┌──────────────┐ │ ┌──────────────┐
                   │ RBAC 子系统   │ │ │ ModuleFrame  │
                   │ (Layout+侧栏) │ │ │ (iframe容器) │
                   │ /dashboard    │ │ │ /module/:code│
                   │ /users        │ │ └──────────────┘
                   │ /roles        │ │
                   │ /permissions  │ │
                   │ /modules      │ │
                   └──────────────┘ │
```

### 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go 1.26 + Gin + GORM + MySQL + Redis(可选) + JWT(HS256) |
| 前端 | Vue 2.7 + Element UI 2.15 + Vue Router 3.6 + Axios + Vite 5 |
| 认证 | JWT (Access Token 2h + Refresh Token 7d) 或 X-API-Key (SHA256) |
| SDK | Go / Python / Node.js / Java 8+ / Java 17+ |

---

## 核心概念

### 五层数据模型

```
用户 (User) ──多对多──▶ 角色 (Role) ──多对多──▶ 权限 (Permission) ──多对一──▶ 模块 (Module)
                             │
                             └── 多对多 ──▶ 模块 (Module)  ← 直接绑定，简化配置
```

| 实体 | 作用 | 关键字段 |
|------|------|---------|
| **Module (模块)** | 功能子系统入口，控制门户卡片可见性 | `name`, `code`, `icon`, `url`, `sort`, `status` |
| **Permission (权限)** | 具体的操作能力 `resource:action` | `name`, `resource`, `action`, `module_id` |
| **Role (角色)** | 模块与权限的集合，分配的最小单位 | `name`, `description` (+ 关联的 Modules, Permissions) |
| **User (用户)** | 登录账号 | `username`, `password`, `email`, `status` |
| **ServiceAccount (服务账号)** | 服务间 API 调用 | `name`, `api_key` (创建时返回一次) |

### 通配符 `*:*`

- 拥有 `*:*` 权限的用户是**超级管理员**
- 自动获得所有资源的 CRUD 权限
- 门户自动展示所有启用模块

### 模块可见性推导

用户能看到的模块由**两条路径取并集**：

1. **角色→权限→模块**：角色有权限 `payment:read`（该权限的 `module_id` 指向支付模块）→ 用户能看到支付模块
2. **角色→模块（直接绑定）**：角色直接绑定了支付模块 → 用户能看到支付模块

推荐使用**路径2**简化配置：新建角色 → 分配模块 → 用户即有模块入口。

---

## 快速入门

### 场景一：给新用户分配已有模块（最简单）

```
① 角色管理 → 找到已有角色（如「支付操作员」），确认已分配模块和权限
② 用户管理 → 新建用户 → 点击「分配角色」→ 勾选该角色
```

### 场景二：新模块上线全流程

```
① 模块管理 → 新增模块「支付管理」 code=payment
② 权限管理 → 新建权限 payment:read, payment:create (模块选「支付管理」)
③ 角色管理 → 新建角色「支付操作员」
   → 点击「分配模块与权限」→ ①选模块 ②勾权限 → 保存
④ 用户管理 → 新建/编辑用户 → 分配角色「支付操作员」
```

### 场景三：只让用户看到模块，不给操作权限

```
① 角色管理 → 新建角色「访客」
② 点击「分配模块与权限」→ 只选模块，不勾权限 → 保存
③ 用户管理 → 给用户分配「访客」角色
```

---

## 页面操作指南

### 1. 模块门户 (/portal)

登录后首先到达的页面，以卡片网格展示所有可访问模块。

- 顶部栏显示当前用户 + 退出按钮
- 卡片点击：
  - 有外部 URL → 跳转到 `/module/:code`（iframe 加载外部系统）
  - `system_mgmt` 编码 → 跳转到 `/dashboard`（RBAC 子系统）
  - 无 URL + 无内置路由 → 提示"未配置入口地址"

### 2. RBAC 子系统 (/dashboard → Layout)

左侧可折叠侧边栏，菜单根据用户权限自动过滤：

| 菜单项 | 路径 | 所需权限 |
|--------|------|---------|
| 系统概览 | /dashboard | 始终可见 |
| 用户管理 | /users | `user:read` |
| 角色管理 | /roles | `role:read` |
| 权限管理 | /permissions | `permission:read` |
| 模块管理 | /modules | `module:read` |

侧栏底部有"返回门户"按钮。

#### 2.1 系统概览 (Dashboard)

- 4 个统计卡片（用户/角色/权限/模块总数）
- 4 个快捷入口按钮（灰显无权限的）

#### 2.2 用户管理 (/users)

| 操作 | 说明 |
|------|------|
| 新建用户 | 填写用户名(2-64字符)、密码(≥6位)、邮箱(选填) |
| 编辑用户 | 修改邮箱、启用/禁用 |
| 改密 | 输入旧密码 + 新密码 |
| 分配角色 | 穿梭框选择，覆盖式更新 |
| 删除 | 确认后软删除 |

**表格列**：ID、用户名、邮箱、状态（启用/禁用标签）、创建时间

#### 2.3 角色管理 (/roles)

| 操作 | 说明 |
|------|------|
| 新建/编辑角色 | 填写名称(2-64字符)、描述 |
| **分配模块与权限** | **合并弹窗**，一步完成模块+权限分配 |
| 删除 | 确认后删除（有关联用户时不允许删） |

**表格列**：ID、名称、可访问模块（绿色标签）、关联权限（蓝色标签）、描述、创建时间

**分配模块与权限弹窗**：

```
① 选择可访问的模块（穿梭框）
      ↓ 选中的模块自动出现下方
② 选择各模块下的操作权限（复选框分组）
   - 每模块一行：[全选] [清空]
   - 权限格式：权限名称 (resource:action)
      ↓ 点击保存
   一次性提交模块 + 权限
```

#### 2.4 权限管理 (/permissions)

| 操作 | 说明 |
|------|------|
| 新建权限 | 名称 + 资源(resource) + 操作(action) + 描述 + 所属模块(选填) |
| 编辑 | 可修改所有字段 |
| 删除 | 有关联角色时不允许删 |

**操作类型下拉选项**：create, read, update, delete, manage, publish, *

**表格列**：ID、名称、资源（蓝色标签）、操作（绿色标签）、描述、创建时间

#### 2.5 模块管理 (/modules)

| 操作 | 说明 |
|------|------|
| 新建模块 | 名称 + 编码 + 图标 + 描述 + 排序 + 入口地址(选填) |
| 编辑 | 可修改所有字段 |
| 删除 | 有关联权限时不允许删 |

**图标支持格式**：

| 格式 | 示例 | 效果 |
|------|------|------|
| Element UI | `el-icon-s-order` | Element UI 图标 |
| Font Awesome | `fa fa-paypal` | 需项目引入 FA |
| 图片 URL | `https://cdn.example.com/logo.png` | `<img>` 渲染 |
| Emoji | `💰` | 直接显示 |

**表格列**：ID、名称+图标、编码（标签）、图标预览、描述、入口地址（蓝色/灰色）、排序、状态（启用/禁用标签）、创建时间

**编码规范**：小写字母开头，只能包含字母数字下划线 `/^[a-z][a-z0-9_]*$/`

### 3. 外部模块容器 (/module/:code)

全屏 iframe 加载外部系统，顶部有"返回门户"按钮。

Token 传递方式：
- **URL 参数**：`iframeSrc = url?rbac_token=<JWT_TOKEN>`
- **postMessage**：`{ type: 'RBAC_TOKEN', token, username }`

加载失败显示错误提示 + 重新加载 + 返回门户按钮。

---

## 外部模块集成

### 步骤

```
① 模块管理 → 新建模块，填写入口地址(url)为外部系统前端 URL
② 权限管理 → 新建该模块对应的权限（resource:action）
③ 角色管理 → 新建角色 → 分配模块与权限
④ 用户管理 → 给用户分配角色
```

### 外部系统接收 Token

**URL 参数（自动传递）**：

```javascript
const p = new URLSearchParams(location.search)
const token = p.get('rbac_token')
if (token) {
  localStorage.setItem('token', token)
  history.replaceState(null, '', location.pathname)
}
```

**postMessage 监听（备选）**：

```javascript
window.addEventListener('message', (e) => {
  if (e.data?.type === 'RBAC_TOKEN') {
    localStorage.setItem('token', e.data.token)
  }
})
```

### 外部系统后端鉴权

```bash
POST /api/v1/auth/introspect
Content-Type: application/json

{
  "token": "<用户token>",
  "resource": "payment",
  "action": "refund"
}

# 返回: {"code":0, "data":{"active":true, "user_id":3, "username":"lisi"}}
# active=false 表示 token 无效或用户无此权限
```

### Nginx 部署示例

```nginx
# RBAC 主站
server {
    listen 80;
    server_name admin.example.com;
    location / { root /opt/api-rbac/web/dist; try_files $uri /index.html; }
    location /api/ { proxy_pass http://127.0.0.1:8087; }
}

# 外部业务系统
server {
    listen 80;
    server_name payment.example.com;
    location / { root /opt/payment-web/dist; try_files $uri /index.html; }
    location /api/ { proxy_pass http://127.0.0.1:8091; }
}
```

模块入口地址配置为 `http://payment.example.com`。

---

## 完整 API 参考

### 公共接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/v1/auth/login` | 登录，body: `{account, password}` |
| POST | `/api/v1/auth/refresh` | 刷新 token，body: `{refresh_token}` |
| POST | `/api/v1/auth/introspect` | Token 自省，body: `{token, resource?, action?}` |

### 认证接口（需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/logout` | 登出（无状态，客户端丢 token） |
| POST | `/api/v1/auth/verify` | 验证 token |
| POST | `/api/v1/auth/check` | 单项权限检查 body: `{resource, action}` |
| POST | `/api/v1/auth/batch-check` | 批量权限检查（≤50项） |
| GET | `/api/v1/auth/menu` | 获取用户权限列表 |
| GET | `/api/v1/auth/modules` | 获取用户可见模块（含权限） |

### 用户管理

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/v1/users` | `user:read` |
| GET | `/api/v1/users/:id` | `user:read` |
| POST | `/api/v1/users` | `user:create` |
| PUT | `/api/v1/users/:id` | `user:update` |
| DELETE | `/api/v1/users/:id` | `user:delete` |
| PUT | `/api/v1/users/:id/password` | `user:update` |
| POST | `/api/v1/users/:id/roles` | `user:update` |
| DELETE | `/api/v1/users/:id/roles/:roleId` | `user:update` |

### 角色管理

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/v1/roles` | `role:read` |
| GET | `/api/v1/roles/:id` | `role:read` |
| POST | `/api/v1/roles` | `role:create` |
| PUT | `/api/v1/roles/:id` | `role:update` |
| DELETE | `/api/v1/roles/:id` | `role:delete` |
| POST | `/api/v1/roles/:id/permissions` | `role:update` |
| DELETE | `/api/v1/roles/:id/permissions/:permId` | `role:update` |
| POST | `/api/v1/roles/:id/modules` | `role:update` |
| DELETE | `/api/v1/roles/:id/modules/:modId` | `role:update` |

### 权限管理

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/v1/permissions` | `permission:read` |
| GET | `/api/v1/permissions/:id` | `permission:read` |
| POST | `/api/v1/permissions` | `permission:create` |
| PUT | `/api/v1/permissions/:id` | `permission:update` |
| DELETE | `/api/v1/permissions/:id` | `permission:delete` |

### 服务账号管理

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/v1/service-accounts` | `service_account:read` |
| GET | `/api/v1/service-accounts/:id` | `service_account:read` |
| POST | `/api/v1/service-accounts` | `service_account:create` |
| PUT | `/api/v1/service-accounts/:id` | `service_account:update` |
| DELETE | `/api/v1/service-accounts/:id` | `service_account:delete` |

### 模块管理

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/v1/modules` | `module:read` |
| GET | `/api/v1/modules/:id` | `module:read` |
| POST | `/api/v1/modules` | `module:create` |
| PUT | `/api/v1/modules/:id` | `module:update` |
| DELETE | `/api/v1/modules/:id` | `module:delete` |

### 统一响应格式

**成功**: `{"code": 0, "message": "success", "data": {...}}`
**分页**: `{"code": 0, "message": "success", "data": {"list": [...], "total": N, "page": N, "page_size": N}}`
**错误**: `{"code": 1003, "message": "无权限"}`

### 错误码

| Code | 含义 | HTTP |
|------|------|------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 未授权 | 401 |
| 1003 | 无权限 | 403 |
| 1004 | 资源不存在 | 404 |
| 1005 | 服务器错误 | 500 |
| 1006 | 资源已存在 | 409 |
| 1007 | Token 过期 | 401 |
| 1008 | Token 无效 | 401 |
| 1009 | 密码错误 | 401 |
| 1010 | 用户已禁用 | 401 |

---

## SDK 与中间件

### Go SDK

```go
import "github.com/laazua/api-rbac/pkg/client"

client := client.NewRBACClient("http://localhost:8087", "username", "password")

// 检查权限
allowed, _ := client.CheckPermission("user", "delete")

// Gin 中间件
r.Use(client.PermissionGuard("user", "read"))

// 带熔断和缓存的中间件
r.Use(client.ResilientGuard("user", "read", rbacClient,
    client.WithFailMode(client.FailModeCache),
))
```

### Python SDK

文件: `sdk/python/rbac_client.py`

```python
from rbac_client import RBACClient, FailMode

client = RBACClient("http://localhost:8087", "username", "password")
if client.check_permission("user", "delete"):
    ...
```

支持熔断器 + 本地缓存 + 自动 Token 刷新。

### Node.js SDK

文件: `sdk/nodejs/src/index.js`

```javascript
const { RBACClient, permissionGuard } = require('@laazua/api-rbac-sdk')

// Express 中间件
app.use(permissionGuard(client, 'user', 'read'))
```

### 服务账号（Service Account）

用于服务间调用，使用 X-API-Key 而非 JWT 认证：

```bash
curl -H "X-API-Key: rbac_sa_abc123..." \
  http://rbac:8087/api/v1/auth/introspect \
  -d '{"token": "user_token", "resource": "order", "action": "create"}'
```

服务账号通过 `RequirePermission` 中间件时**自动跳过权限检查**（视为可信内部服务）。

---

## 配置说明

### 主配置文件

文件: `config/config.yaml`（启动参数 `-c` 可指定路径）

```yaml
server:
  mode: debug        # debug | release
  port: 8087

db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "xxx"
  dbname: api_rbac
  charset: utf8mb4

jwt:
  secret: "your-secret-key"   # 生产环境务必修改
  expire_hour: 2              # Access Token 过期时间
  refresh_expire_day: 7       # Refresh Token 过期时间

redis:
  host: 127.0.0.1             # 可选，不可用时自动降级
  port: 6379
  password: ""
  db: 0
  pool_size: 10

cors:
  allow_origins:
    - "*"                      # 生产环境限制为具体域名
```

### 首次运行

1. 启动服务，检测到无 admin 用户时进入交互式初始化
2. 设置超级管理员密码（≥6位）
3. 自动创建：
   - 默认模块"系统管理" `system_mgmt`
   - 通配符权限 `*:*`
   - 超级管理员角色 + admin 用户
   - 默认服务账号 "default"（API Key 仅显示一次）

### 前端配置

文件: `web/vite.config.js`

```javascript
server: {
  port: 8088,
  proxy: { '/api': 'http://localhost:8087' }
}
```

`npm run dev` 启动前端开发服务器（8088端口），API 代理到后端 8087。

`npm run build` 构建到 `web/dist/`，由后端 Gin 静态文件服务或 Nginx 托管。

---

## 常见场景示例

### 新业务系统上线全流程

> 上线"数据分析"系统，给分析团队开通权限。

```bash
# 1. 注册模块
POST /api/v1/modules
{"name":"数据分析","code":"analytics","icon":"el-icon-s-data","url":"http://analytics:8090","sort":3}

# 2. 创建权限
POST /api/v1/permissions
{"name":"查看报表","resource":"analytics","action":"read","module_id":<模块ID>}
POST /api/v1/permissions
{"name":"导出报表","resource":"analytics","action":"export","module_id":<模块ID>}

# 3. 创建角色 + 绑定
POST /api/v1/roles
{"name":"数据分析师","description":"数据分析团队"}
POST /api/v1/roles/<角色ID>/modules
{"module_ids":[<模块ID>]}
POST /api/v1/roles/<角色ID>/permissions
{"permission_ids":[<权限1ID>,<权限2ID>]}

# 4. 分配角色
POST /api/v1/users/<用户ID>/roles
{"role_ids":[<角色ID>]}
```

### 权限分级示例

```
角色A: 支付操作员              角色B: 支付审核员
├── 模块: 支付管理            ├── 模块: 支付管理
├── payment:read              ├── payment:read
└── payment:create            ├── payment:audit
                              └── payment:refund

用户张三 → 角色A → 可查看+创建支付，不能退款
用户李四 → 角色B → 可查看+审核+退款，不能创建
用户王五 → 角色A+角色B → 拥有全部权限
```

### 数据库表关系

```
users ──< user_roles >── roles ──< role_permissions >── permissions ──< module_id ── modules
                    └──< role_modules >──────────────────────────────────────────┘
```

### 常用命令

```bash
cd /opt/codes/api-rbac

# 后端
go run ./cmd/server                          # 启动服务
go build -o api-rbac ./cmd/server            # 编译
go test ./...                                # 测试
go fmt ./... && go vet ./...                 # 格式化+检查

# 前端
npm --prefix ./web run dev                   # 开发模式 (8088)
npm --prefix ./web run build                 # 构建

# 通过启动参数指定配置
./api-rbac -c /path/to/config.yaml
```
