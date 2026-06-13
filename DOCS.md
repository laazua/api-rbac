# api-rbac 权限管理系统 — 完整技术文档

> **版本**: 2.0 | **语言**: Go 1.21+ | **数据库**: MySQL 8.0+ | **缓存**: Redis (可选) | **许可**: MIT

---

## 目录

- [1. 项目概述](#1-项目概述)
- [2. 核心设计原理](#2-核心设计原理)
- [3. 权限模型](#3-权限模型)
- [4. 服务架构](#4-服务架构)
- [5. 完整 API 参考](#5-完整-api-参考)
- [6. 部署运维](#6-部署运维)
- [7. 多语言 SDK 使用](#7-多语言-sdk-使用)
- [8. 业务系统集成示例](#8-业务系统集成示例)
- [9. 前端权限集成方案](#9-前端权限集成方案)
- [10. 安全最佳实践](#10-安全最佳实践)
- [11. 韧性设计：RBAC 宕机时的安全防护](#11-韧性设计rbac-宕机时的安全防护)
- [12. 常见问题排查 (FAQ)](#12-常见问题排查-faq)
- [附录 A: 错误码对照表](#附录-a-错误码对照表)
- [附录 B: 项目文件清单](#附录-b-项目文件清单)
- [附录 C: Go 服务内部详解](#附录-c-go-服务内部详解)

---

## 1. 项目概述

### 1.1 是什么

**api-rbac** 是一个独立的、通用的**基于角色的访问控制 (RBAC) 微服务**。它以 HTTP REST API 形式运行，与业务代码**完全解耦**，为任何编程语言的业务系统提供统一的权限管理能力。

### 1.2 解决什么问题

```
传统方式:                               api-rbac 方式:
每个业务系统自己实现登录+权限          统一权限服务, 业务系统只调 HTTP API

┌──────────┐ ┌──────────┐              ┌──────────┐ ┌──────────┐ ┌──────────┐
│ Python   │ │ Java     │              │ Python   │ │ Java     │ │ Node.js  │
│ 业务系统  │ │ 业务系统  │              │ 业务系统  │ │ 业务系统  │ │ 业务系统  │
│ ┌──────┐ │ │ ┌──────┐ │              └────┬─────┘ └────┬─────┘ └────┬─────┘
│ │用户表 │ │ │ │用户表 │ │                   │ HTTP       │ HTTP       │ HTTP
│ │角色表 │ │ │ │角色表 │ │                   │            │            │
│ │权限表 │ │ │ │权限表 │ │              ┌────┴────────────┴────────────┴────┐
│ │JWT...│ │ │ │JWT...│ │              │         api-rbac (Go)              │
│ └──────┘ │ │ └──────┘ │              │  用户/角色/权限/JWT — 统一管理        │
└──────────┘ └──────────┘              └────────────────────────────────────┘
   ❌ 重复开发  ❌ 数据分散                 ✅ 一次开发  ✅ 数据集中  ✅ 多语言
```

### 1.3 核心能力

| 能力 | 说明 |
|------|------|
| **用户认证** | 用户名/邮箱 + 密码登录，bcrypt 加密，JWT 签发 |
| **用户管理** | 用户 CRUD，状态启用/禁用，密码修改 |
| **角色管理** | 角色 CRUD，角色名称唯一 |
| **权限管理** | 权限 CRUD，resource + action 抽象模型，支持通配符 `*` |
| **用户-角色绑定** | 多对多，覆盖式分配和单条移除 |
| **角色-权限绑定** | 多对多，覆盖式分配和单条移除 |
| **权限检查** | 单次检查 / 批量检查 / Token 自省 (验证+鉴权一步完成) |
| **Token 刷新** | Access Token 2h + Refresh Token 7d，自动续期 |
| **服务间认证** | API Key (Service Account)，无需用户 Token |
| **权限缓存** | Redis 缓存权限数据，性能从 ~10ms → ~1ms |
| **软删除** | GORM DeletedAt，数据可恢复 |

### 1.4 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| HTTP 框架 | Gin |
| ORM | GORM + MySQL |
| 认证 | JWT (HS256) + bcrypt |
| 缓存 | Redis (go-redis/v9) |
| 配置 | Viper (YAML) |
| 文档 | Swagger (swaggo) |

---

## 2. 核心设计原理

### 2.1 解耦原则

api-rbac 只做一件事：**回答 "用户 X 是否拥有对资源 Y 执行操作 Z 的权限？"**

```
api-rbac 知道:                          api-rbac 不知道:
─────────────────                      ─────────────────
✅ 用户表 (users)                       ❌ 什么是"服务器"
✅ 角色表 (roles)                       ❌ 什么是"订单"
✅ 权限表 (permissions)                 ❌ 什么是"发布"
✅ 用户-角色关系 (user_roles)           ❌ 什么是"告警"
✅ 角色-权限关系 (role_permissions)     ❌ 任何业务数据
✅ JWT 加密/解密                        ❌ 任何业务逻辑
✅ bcrypt 密码验证
```

### 2.2 抽象权限模型

权限不绑定任何业务实体，而是完全抽象的 **资源 (resource) + 操作 (action)** 字符串对：

```
业务系统定义:                  api-rbac 存储:
────────────────────          ──────────
"能否删除用户"      →         { resource: "user",   action: "delete" }
"能否重启服务器"    →         { resource: "server", action: "restart" }
"能否执行发布"      →         { resource: "deployment", action: "execute" }
"能否确认告警"      →         { resource: "alert",  action: "ack" }
```

这就像 Linux 文件系统的 `rwx` 权限 —— 操作系统不知道你文件的内容，只管权限位。

### 2.3 数据流

```
  ┌──────────┐       ┌──────────────┐       ┌──────────┐
  │ 前端/客户端│       │  业务系统      │       │ api-rbac │
  └────┬─────┘       └──────┬───────┘       └────┬─────┘
       │                    │                    │
       │ 1. POST /login     │                    │
       │ ──────────────────→│                    │
       │                    │ 2. POST /auth/login│
       │                    │ ──────────────────→│
       │                    │                    │ 3. 查用户, 验密码
       │                    │                    │ 4. 签发 JWT
       │                    │ 5. {token,...}     │
       │                    │ ←──────────────────│
       │ 6. {token,...}     │                    │
       │ ←──────────────────│                    │
       │                    │                    │
       │ 7. POST /servers/restart (Bearer token) │
       │ ──────────────────→│                    │
       │                    │ 8. POST /auth/check│
       │                    │    {resource,action}│
       │                    │ ──────────────────→│
       │                    │                    │ 9. 解析 JWT
       │                    │                    │ 10. 查权限 (Redis/DB)
       │                    │ 11. {allowed:true} │
       │                    │ ←──────────────────│
       │                    │ 12. 执行业务逻辑     │
       │ 13. {success}      │                    │
       │ ←──────────────────│                    │
```

---

## 3. 权限模型

### 3.1 核心实体

```
User (用户)           Role (角色)            Permission (权限)
┌─────────────┐      ┌─────────────┐        ┌──────────────────────┐
│ id          │      │ id          │        │ id                   │
│ username    │      │ name        │        │ name    (权限名称)     │
│ password    │ 1:N  │ description │ 1:N    │ resource (资源标识)    │
│ email       │──────│             │────────│ action   (操作标识)    │
│ status      │      └─────────────┘        │ description          │
└─────────────┘                             └──────────────────────┘
      │                                              │
      │ user_roles (多对多)                            │ role_permissions (多对多)
      └──────────────────────────────────────────────┘
```

### 3.2 通配符规则

| 权限定义 | 匹配范围 |
|----------|---------|
| `*:*` | **全部权限** — 匹配所有 resource 和所有 action (超级管理员) |
| `server:*` | 服务器模块的**全部操作** (读/重启/停止/...) |
| `*:read` | **所有模块**的只读操作 |
| `server:restart` | 精确匹配 — 仅匹配服务器的重启操作 |

```go
// 权限匹配逻辑 (internal/service/permission_check.go)
// 优先级: *:* > resource:* > *:action > resource:action

hasPerm("*", "*")           → true  // 超级管理员
hasPerm("server", "*")      → true  // 模块管理员
hasPerm("*", "read")        → true  // 全局只读
hasPerm("server", "restart")→ true  // 精确匹配
```

### 3.3 命名规范建议

```
resource: 小写英文单词, 下划线分隔
  ✅ server, k8s_cluster, billing_invoice
  ❌ Server, 服务器

action: 标准 CRUD 或自定义动词
  标准: read, create, update, delete
  自定义: restart, execute, rollback, ack, download, export, approve
```

---

## 4. 服务架构

### 4.1 Go 服务分层结构

```
cmd/server/main.go            # 入口: 启动 + 依赖注入 + 优雅关闭
│
internal/
├── handler/                  # HTTP 处理器 — 参数绑定与响应
│   ├── auth_handler.go       #   /auth/*
│   ├── user_handler.go       #   /users/*
│   ├── role_handler.go       #   /roles/*
│   ├── permission_handler.go #   /permissions/*
│   └── service_account_handler.go  # /service-accounts/*
│
├── service/                  # 业务逻辑层
│   ├── auth_service.go       #   登录验证 + bcrypt
│   ├── user_service.go       #   用户 CRUD + 角色分配
│   ├── role_service.go       #   角色 CRUD + 权限分配
│   ├── permission_service.go #   权限 CRUD
│   ├── permission_check.go   #   权限检查 (含缓存)
│   └── service_account_service.go  # 服务账号管理
│
├── repository/               # 数据访问层 (GORM)
│   ├── user_repo.go
│   ├── role_repo.go
│   ├── permission_repo.go
│   └── service_account_repo.go
│
├── model/                    # 数据模型
│   ├── base.go               #   BaseModel (ID, CreatedAt, UpdatedAt, DeletedAt)
│   ├── user.go               #   User + 请求结构体
│   ├── role.go               #   Role + 请求结构体
│   ├── permission.go         #   Permission + 请求结构体
│   ├── service_account.go    #   ServiceAccount
│   └── introspect.go         #   自省请求/响应
│
├── middleware/               # 中间件
│   ├── auth.go               #   JWT + API Key 认证
│   ├── cors.go               #   跨域处理
│   ├── logger.go             #   请求日志
│   └── permission.go         #   权限校验中间件
│
└── router/router.go          # 路由定义

pkg/
├── errcode/errcode.go        # 统一错误码 (0 ~ 1011)
├── response/response.go      # 统一 JSON 响应格式
├── jwt/jwt.go                # JWT 生成/解析 (Access + Refresh)
└── client/                   # Go SDK
    ├── rbac_client.go        #   HTTP 客户端
    └── middleware.go          #   Gin PermissionGuard

config/
├── config.go                 # 配置结构体 + Viper 加载
├── config.yaml               # 运行配置
└── config.example.yaml       # 配置模板

migrations/
├── 001_init.sql              # 初始表结构参考
└── 002_soft_delete.sql       # 软删除迁移参考
```

### 4.2 认证中间件链

```
请求进入
  │
  ├─ /health, /auth/login, /auth/refresh, /auth/introspect
  │    → 无需认证, 直接进入 Handler
  │
  └─ 其他所有路径
       │
       ▼
  middleware.AuthRequired(saRepo)
       │
       ├─ X-API-Key 头部?
       │    → 查 service_accounts 表 (SHA256 比对)
       │    → 设置 auth_type="apikey" + service_account_id
       │    → RequirePermission 检测到 apikey 类型 → 直接放行
       │
       └─ Authorization: Bearer <token>?
            → 解析 JWT → 安全类型断言 → 设置 user_id, username
       │
       ▼
  (可选) middleware.RequirePermission(permCheckSvc, resource, action)
       → apikey 认证? → 直接放行 (受信任内部服务)
       → JWT 认证? → getUserID(c) 安全提取 → 调 CheckPermission
       → 优先从 Redis 缓存读取, miss 则查 DB
       → 通配符匹配 (*:*, resource:*, *:action) → 返回 true/false
       │
       ▼
  Handler → Service → Repository → DB
```

### 4.3 数据表结构

```
┌─────────────────────────────────────────────────────────────┐
│  users                                                       │
│  id | username | password | email | status | created_at      │
│     |          | (bcrypt) |       | (0/1)  | updated_at     │
│     |          |          |       |        | deleted_at      │
├─────────────────────────────────────────────────────────────┤
│  roles                                                       │
│  id | name | description | created_at | updated_at | deleted_at│
├─────────────────────────────────────────────────────────────┤
│  permissions                                                 │
│  id | name | resource | action | description | ... | deleted_at│
├─────────────────────────────────────────────────────────────┤
│  user_roles (多对多关联表)                                     │
│  user_id (PK) | role_id (PK)                                 │
├─────────────────────────────────────────────────────────────┤
│  role_permissions (多对多关联表)                                │
│  role_id (PK) | permission_id (PK)                           │
├─────────────────────────────────────────────────────────────┤
│  service_accounts                                            │
│  id | name | api_key_hash | status | description | deleted_at│
│     |      | (SHA256)     | (0/1)  |                         │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. 完整 API 参考

> 所有接口前缀: `/api/v1` | 运行地址: `http://localhost:8087` | Swagger: `/swagger/index.html`

### 5.1 认证接口 (无需 Token)

#### POST /auth/login — 用户登录

```bash
curl -X POST http://localhost:8087/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"admin","password":"your_password"}'
```

响应 `200`:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOi...",          // Access Token (2h)
    "refresh_token": "eyJhbGciOi...",  // Refresh Token (7d)
    "expires_in": 7200,                // 过期秒数
    "user_id": 1,
    "username": "admin"
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account | string | ✅ | 用户名或邮箱 |
| password | string | ✅ | 明文密码 |

#### POST /auth/refresh — 刷新 Token

```bash
curl -X POST http://localhost:8087/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"eyJhbGciOi..."}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refresh_token | string | ✅ | 登录时获取的 refresh_token |

响应: `{ code: 0, data: { token, refresh_token, expires_in } }`

#### POST /auth/introspect — Token 自省

**推荐外部服务使用此接口**，一次调用完成 Token 验证 + 权限检查。

```bash
curl -X POST http://localhost:8087/api/v1/auth/introspect \
  -H "Content-Type: application/json" \
  -d '{"token":"eyJhbGciOi...","resource":"order","action":"read"}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| token | string | ✅ | 需要验证的 JWT |
| resource | string | | 可选，需要检查的资源 |
| action | string | | 可选，需要检查的操作 |

响应:
```json
{
  "code": 0,
  "data": {
    "active": true,      // Token 有效 + 有权限
    "user_id": 1,
    "username": "admin"
  }
}
// 或
{ "code": 0, "data": { "active": false } }  // Token 无效/过期/无权限
```

### 5.2 认证接口 (需要 Token)

以下接口需携带 `Authorization: Bearer <token>` 或 `X-API-Key: <apikey>` 头部。

#### POST /auth/verify — 验证 Token

```bash
curl -X POST http://localhost:8087/api/v1/auth/verify \
  -H "Authorization: Bearer <token>"
```
响应: `{ code: 0, data: { user_id: 1, username: "admin" } }`

#### POST /auth/check — 检查单个权限

```bash
curl -X POST http://localhost:8087/api/v1/auth/check \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"resource":"user","action":"delete"}'
```
有权限: `{ code: 0, data: { allowed: true } }`
无权限: `{ code: 1003, message: "无权限", data: { allowed: false } }`

#### POST /auth/batch-check — 批量检查权限

```bash
curl -X POST http://localhost:8087/api/v1/auth/batch-check \
  -H "Authorization: Bearer <token>" \
  -d '{"permissions":[{"resource":"user","action":"read"},{"resource":"user","action":"delete"}]}'
```
响应: `{ code: 0, data: { results: { "user:read": true, "user:delete": false } } }`

> 单次最多 50 个权限项

#### GET /auth/menu — 获取用户全部权限

```bash
curl http://localhost:8087/api/v1/auth/menu \
  -H "Authorization: Bearer <token>"
```
响应:
```json
{ "code": 0, "data": { "permissions": {
    "server": ["read","restart","stop"],
    "deployment": ["read","execute"]
}}}
```
> 前端用此接口获取权限数据，实现菜单/按钮的动态显隐

### 5.3 用户管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/users?page=1&page_size=10&keyword=` | `user:read` | 分页列表 |
| `GET` | `/users/:id` | `user:read` | 用户详情(含角色+权限) |
| `POST` | `/users` | `user:create` | 创建用户 |
| `PUT` | `/users/:id` | `user:update` | 更新用户 (email, status) |
| `DELETE` | `/users/:id` | `user:delete` | 删除用户(软删除) |
| `PUT` | `/users/:id/password` | `user:update` | 修改密码 (需旧密码) |
| `POST` | `/users/:id/roles` | `user:update` | 分配角色 (覆盖式) |
| `DELETE` | `/users/:id/roles/:roleId` | `user:update` | 移除单个角色 |

创建用户:
```bash
curl -X POST http://localhost:8087/api/v1/users \
  -H "Authorization: Bearer <token>" \
  -d '{"username":"zhangsan","password":"123456","email":"zhang@example.com"}'
```

分配角色:
```bash
curl -X POST http://localhost:8087/api/v1/users/2/roles \
  -H "Authorization: Bearer <token>" \
  -d '{"role_ids":[1,2]}'
```

### 5.4 角色管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/roles?page=1&page_size=10&keyword=` | `role:read` | 分页列表 |
| `GET` | `/roles/:id` | `role:read` | 角色详情(含权限) |
| `POST` | `/roles` | `role:create` | 创建角色 |
| `PUT` | `/roles/:id` | `role:update` | 更新角色 |
| `DELETE` | `/roles/:id` | `role:delete` | 删除角色 |
| `POST` | `/roles/:id/permissions` | `role:update` | 分配权限 (覆盖式) |
| `DELETE` | `/roles/:id/permissions/:permId` | `role:update` | 移除单个权限 |

### 5.5 权限管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/permissions?page=1&page_size=10&keyword=` | `permission:read` | 分页列表 |
| `GET` | `/permissions/:id` | `permission:read` | 权限详情 |
| `POST` | `/permissions` | `permission:create` | 创建权限 |
| `PUT` | `/permissions/:id` | `permission:update` | 更新权限 |
| `DELETE` | `/permissions/:id` | `permission:delete` | 删除权限 |

创建权限示例:
```bash
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"删除用户","resource":"user","action":"delete","description":"允许删除其他用户"}'
```

### 5.6 服务账号管理

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/service-accounts?page=1` | `service_account:read` | 列表 |
| `GET` | `/service-accounts/:id` | `service_account:read` | 详情 |
| `POST` | `/service-accounts` | `service_account:create` | 创建 (返回 API Key) |
| `PUT` | `/service-accounts/:id` | `service_account:update` | 更新 |
| `DELETE` | `/service-accounts/:id` | `service_account:delete` | 删除 |

创建服务账号:
```bash
curl -X POST http://localhost:8087/api/v1/service-accounts \
  -H "Authorization: Bearer <token>" \
  -d '{"name":"order-service","description":"订单服务调用账号"}'
# 响应包含 api_key — 仅显示这一次!
```

使用 API Key 调用:
```bash
curl -H "X-API-Key: rbac_sa_xxxx" http://localhost:8087/api/v1/auth/check \
  -d '{"resource":"user","action":"read"}'
```

### 5.7 统一响应格式

成功:
```json
{ "code": 0, "message": "success", "data": { ... } }
```

分页:
```json
{ "code": 0, "message": "success", "data": { "list": [...], "total": 100, "page": 1, "page_size": 10 } }
```

错误:
```json
{ "code": 1003, "message": "无权限" }
```

HTTP 状态码会对应业务错误码:
- `200`: 成功
- `400`: 参数错误 (1001)
- `401`: 未授权 (1002, 1007, 1008)
- `403`: 无权限 (1003)
- `404`: 资源不存在 (1004)
- `409`: 资源已存在 (1006)
- `500`: 服务器内部错误 (1005, 1011)

### 5.8 其他端点

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `GET` | `/health` | 无 | 健康检查 → `{"status":"ok"}` |
| `GET` | `/swagger/*any` | 无 | Swagger UI |

---

## 6. 部署运维

### 6.1 快速启动

```bash
# 1. 创建数据库
mysql -u root -p -e "CREATE DATABASE api_rbac CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 编辑配置
cp config/config.example.yaml config/config.yaml
vim config/config.yaml  # 修改 db 连接信息 + jwt.secret

# 3. 编译
go build -o api-rbac ./cmd/server

# 4. 启动 (首次运行会提示设置 admin 密码)
./api-rbac
# 输出: 服务启动于 http://0.0.0.0:8087
```

### 6.2 配置说明

```yaml
# config/config.yaml

server:
  mode: debug          # debug | release (生产用 release)
  port: 8087           # 监听端口

db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "your_password"
  dbname: api_rbac
  charset: utf8mb4

redis:
  host: 127.0.0.1      # 可选: Redis 连接信息
  port: 6379
  password: ""
  db: 0
  pool_size: 10        # 连接池大小

jwt:
  secret: "长随机字符串-务必修改"  # 生产环境必改
  expire_hour: 2                 # Access Token 过期 (小时)
  refresh_expire_day: 7          # Refresh Token 过期 (天)

cors:
  allow_origins:
    - "*"               # 生产改为具体域名: ["https://ops.example.com"]
```

### 6.3 系统服务化 (systemd)

```ini
# /etc/systemd/system/api-rbac.service
[Unit]
Description=api-rbac Permission Service
After=network.target mysql.service redis.service

[Service]
Type=simple
User=api-rbac
WorkingDirectory=/opt/api-rbac
ExecStart=/opt/api-rbac/api-rbac -c /opt/api-rbac/config/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 6.4 首次运行

首次启动时，系统检测到 `admin` 用户不存在，会进入交互式初始化流程：

```
========================================
  检测到首次运行，请设置超级管理员密码
========================================
请输入超级管理员密码 (不少于6位):
请再次输入确认密码:
========================================
  ✅ 超级管理员初始化完成
     用户名: admin
========================================
```

> 容器化部署时，可通过环境变量 `ADMIN_PASSWORD` 或 stdin 管道传入密码。

### 6.5 健康检查

```bash
curl http://localhost:8087/health
# → {"status":"ok"}
```

### 6.6 Redis 缓存

Redis 是可选的。如果连接成功，日志显示:
```
✅ Redis 连接成功，权限缓存已启用
```

如果连接失败，系统降级为直接查数据库:
```
⚠️  Redis 连接失败 (...)，权限缓存已禁用，将直接查询数据库
```

---

## 7. 多语言 SDK 使用

### 7.1 SDK 概览

| 语言 | 目录 | 最低版本 | 核心依赖 |
|------|------|----------|---------|
| **Go** | `pkg/client/` | Go 1.21 | 无 (标准库) |
| **Python** | `sdk/python/` | Python 3.8 | 无 (urllib) |
| **Node.js** | `sdk/nodejs/` | Node.js 14 | 无 (fetch) |
| **Java 8+** | `sdk/java/` | Java 8 | 无 (javax.servlet 可选) |
| **Java 17+** | `sdk/java17/` | Java 17 | 无 (jakarta.servlet 可选) |

### 7.2 功能矩阵

| 功能 | Go | Python | Node.js | Java 8+ | Java 17+ |
|------|:--:|:------:|:-------:|:-------:|:--------:|
| 登录 (`login`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| 刷新 Token (`refresh`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| 验证 Token (`verify`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| 检查权限 (`checkPermission`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| 批量检查 (`batchCheck`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Token 自省 (`introspect`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| 获取菜单 (`getMenu`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Web 中间件/拦截器 | Gin | Flask | Express | Servlet+Spring | Jakarta+Spring |

### 7.3 Go SDK

```go
import "api-rbac/pkg/client"

func main() {
    c := client.NewRBACClient("http://localhost:8087/api/v1")

    // 登录
    login, _ := c.Login("admin", "password")
    token := login.Data.Token

    // 检查权限
    check, _ := c.CheckPermission(token, "user", "delete")
    allowed := check.Data.Allowed

    // 批量检查
    batch, _ := c.BatchCheckPermission(token, []client.CheckItem{
        {Resource: "user", Action: "read"},
        {Resource: "user", Action: "delete"},
    })

    // Token 自省
    intro, _ := c.Introspect(token, "order", "read")

    // 刷新
    refresh, _ := c.Refresh(login.Data.RefreshToken)
}

// Gin 中间件
r := gin.Default()
rbacClient := client.NewRBACClient("http://localhost:8087/api/v1")
admin := r.Group("/admin")
admin.Use(client.PermissionGuard(rbacClient, "admin", "access"))
```

### 7.4 Python SDK

```python
from rbac_client import RBACClient

client = RBACClient("http://localhost:8087/api/v1")

# 登录
result = client.login("admin", "password")
token = result["token"]

# 检查权限
allowed = client.check_permission(token, "user", "delete")

# 批量检查
perms = client.batch_check(token, [("user", "read"), ("user", "delete")])
# → {"user:read": True, "user:delete": False}

# Token 自省
info = client.introspect(token, "order", "read")

# 刷新 Token
new_tokens = client.refresh(result["refresh_token"])

# Flask 装饰器
def require_permission(resource, action):
    def decorator(f):
        @wraps(f)
        def wrapper(*args, **kwargs):
            token = request.headers.get("Authorization", "").replace("Bearer ", "")
            if not client.check_permission(token, resource, action):
                return jsonify({"code": 403, "message": "无权限"}), 403
            return f(*args, **kwargs)
        return wrapper
    return decorator
```

### 7.5 Node.js SDK

```js
const { RBACClient, permissionGuard } = require('rbac-client');
const client = new RBACClient('http://localhost:8087/api/v1');

// 登录
const result = await client.login('admin', 'password');
const token = result.token;

// 检查权限
const allowed = await client.checkPermission(token, 'user', 'delete');

// 批量检查
const results = await client.batchCheck(token, [
  ['user', 'read'], ['user', 'delete']
]);

// Express 中间件
app.delete('/orders/:id', permissionGuard(client, 'order', 'delete'), handler);
```

### 7.6 Java 8+ SDK

```java
RBACClient client = new RBACClient("http://localhost:8087/api/v1");

// 登录
RBACClient.LoginResult result = client.login("admin", "password");
String token = result.getToken();

// 检查权限
boolean allowed = client.checkPermission(token, "user", "delete");

// 批量检查
Map<String, Boolean> perms = client.batchCheck(token, Arrays.asList(
    new CheckItem("user", "read"),
    new CheckItem("user", "delete")
));

// Spring Boot 拦截器 (需配置 WebConfig)
// 配合 @RequirePermission 注解使用
```

### 7.7 Java 17+ SDK

```java
var client = new RBACClient("http://localhost:8087/api/v1");

// 登录 — 返回 record
var result = client.login("admin", "password");
// result.token(), result.refreshToken(), result.expiresIn(), result.userId()

// 检查权限
if (client.checkPermission(result.token(), "user", "delete")) { ... }

// 批量检查
var perms = client.batchCheck(result.token(), List.of(
    new CheckItem("user", "read"),
    new CheckItem("user", "delete")
));

// 前端权限判断
var allPerms = client.getMenu(result.token());
if (PermissionUtil.hasPerm(allPerms, "server", "restart")) { ... }
if (PermissionUtil.hasAnyPerm(allPerms, "server")) { ... }

// Spring Boot 3.x @RequirePermission 注解
@PostMapping("/api/servers/{id}/restart")
@RequirePermission(resource = "server", action = "restart")
public Map<String, Object> restart(@PathVariable long id) { ... }
```

---

## 8. 业务系统集成示例

项目提供两个完整的业务系统示例，演示不同语言如何与 api-rbac 解耦集成。

### 8.1 Python Flask 运维管理系统

**目录**: `examples/python-ops-system/`

**运行**:
```bash
cd examples/python-ops-system
pip install flask requests
./setup_permissions.sh    # 初始化权限
python app.py             # 启动于 :5000
# 前端: http://localhost:5000
```

**核心文件**:
| 文件 | 说明 |
|------|------|
| `app.py` | Flask 主程序，~240行 |
| `static/index.html` | 前端 SPA (~450行)，动态菜单/按钮权限 |
| `setup_permissions.sh` | 初始化 8 个权限 + 2 个角色 |

**鉴权模式**:
```python
# 一行装饰器完成鉴权
@app.route("/api/servers/<id>/restart", methods=["POST"])
@require_permission("server", "restart")    # ← 仅此一行
def restart_server(id):
    # 纯业务逻辑, 零权限代码
    ...
```

### 8.2 Java Spring Boot 运维管理系统

**目录**: `examples/java-ops-system/`

**运行**:
```bash
cd examples/java-ops-system
./setup_permissions.sh    # 初始化权限
mvn spring-boot:run       # 启动于 :8080
```

**核心文件**:
| 文件 | 说明 |
|------|------|
| `Application.java` | Spring Boot 入口 |
| `AppConfig.java` | RBACClient Bean 配置 |
| `WebConfig.java` | 拦截器: 自动校验 @RequirePermission |
| `AuthController.java` | 登录/刷新 Token (转发 RBAC) |
| `ServerController.java` | 服务器管理 (server:read/restart/stop) |
| `DeploymentController.java` | 发布管理 (deployment:read/execute/rollback) |
| `AlertController.java` | 告警管理 (alert:read/ack) |

**鉴权模式**:
```java
// 一行注解完成鉴权
@PostMapping("/{id}/restart")
@RequirePermission(resource = "server", action = "restart")  // ← 仅此一行
public Map<String, Object> restart(@PathVariable long id) {
    // 纯业务逻辑, 零权限代码
    ...
}
```

### 8.3 集成最佳实践

**新增业务模块只需两步**:

1. **在 api-rbac 创建权限** (通过管理后台或 API):
```bash
curl -X POST http://localhost:8087/api/v1/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"查看日志","resource":"log","action":"read"}'
```

2. **在业务代码标注权限**:
```python
# Python
@app.route("/api/logs")
@require_permission("log", "read")
def list_logs(): ...

# Java
@GetMapping("/api/logs")
@RequirePermission(resource = "log", action = "read")
public Map<String, Object> list() { ... }
```

**不需要: 修改数据库、修改用户表、写任何权限判断逻辑。**

---

## 9. 前端权限集成方案

### 9.1 三层控制模型

```
第三层: 后端 API 鉴权 (安全底线)
  → @require_permission / @RequirePermission → api-rbac /auth/check → 403
  → 即使前端绕过, 后端也会拦截

第二层: 前端按钮显隐 (UX 优化)
  → hasPerm("server", "restart") ? 显示按钮 : 隐藏按钮
  → 无权限用户看不到操作按钮

第一层: 前端菜单显隐 (导航控制)
  → hasAnyPerm("server") ? 显示菜单 : 隐藏菜单
  → 用户完全没有该模块权限时, 菜单项不渲染
```

### 9.2 实现步骤

**步骤 1**: 登录后加载权限

```javascript
// 前端登录成功后
const resp = await fetch('/api/user/permissions', {
  headers: { 'Authorization': `Bearer ${token}` }
});
state.permissions = (await resp.json()).data;
// → {"server":["read","restart"], "deployment":["read"], "alert":["read","ack"]}
```

后端 `/api/user/permissions` 实际转发调用 api-rbac `GET /auth/menu`。

**步骤 2**: 定义权限判断函数

```javascript
function hasPerm(resource, action) {
  const p = state.permissions;
  if (p['*']?.includes('*')) return true;          // 超级管理员
  if (p[resource]?.includes('*')) return true;     // 模块管理员
  if (p['*']?.includes(action)) return true;       // 全局操作
  return p[resource]?.includes(action) ?? false;   // 精确匹配
}

function hasAnyPerm(resource) {
  const p = state.permissions;
  return p['*'] || (p[resource]?.length > 0);
}
```

**步骤 3**: 菜单动态渲染

```javascript
const menuConfig = [
  { key: 'servers', label: '服务器管理', permission: 'server' },
  { key: 'alerts',  label: '告警管理',   permission: 'alert' },
];

function renderMenu() {
  for (const item of menuConfig) {
    if (item.permission && !hasAnyPerm(item.permission)) {
      continue;  // ← 无权限，跳过不渲染
    }
    // 渲染菜单项
    nav.appendChild(createMenuItem(item));
  }
}
```

**步骤 4**: 按钮动态显隐

```javascript
async function loadServers() {
  const canRestart = hasPerm('server', 'restart');
  const canStop = hasPerm('server', 'stop');

  // 渲染表格时, 根据权限决定是否渲染操作列
  let html = '<table>...';
  if (canRestart) html += '<button onclick="restart()">🔄 重启</button>';
  if (canStop)    html += '<button onclick="stop()">⏹️ 停止</button>';
}
```

### 9.3 不同框架等价写法

**Vue 3**:
```vue
<el-menu-item v-if="hasAnyPerm('server')">服务器管理</el-menu-item>
<el-button v-if="hasPerm('server', 'restart')" @click="restart">重启</el-button>
```

**React**:
```jsx
{hasAnyPerm('server') && <NavItem to="/servers">服务器管理</NavItem>}
{hasPerm('server', 'restart') && <Button onClick={restart}>重启</Button>}
```

**Java (后端渲染)**:
```java
if (PermissionUtil.hasPerm(perms, "server", "restart")) {
    // 渲染重启按钮的 HTML
}
```

> ⚠️ **安全提醒**: 前端显隐只是 UX 优化。真正的安全由后端 `@require_permission` / `@RequirePermission` 保证。即使用户通过浏览器 DevTools 绕过前端限制直接发送 HTTP 请求，api-rbac 也会返回 403。

---

## 10. 安全最佳实践

### 10.1 生产环境配置检查清单

```
[ ] jwt.secret → 改为 32 位以上随机字符串
[ ] server.mode → 改为 release
[ ] cors.allow_origins → 改为具体域名 (不要用 *)
[ ] db.password → 通过环境变量传入, 不要在配置文件中明文
[ ] Redis 密码 → 设置密码
[ ] HTTPS → 在反向代理层 (Nginx/Caddy) 启用 TLS
[ ] admin 密码 → 首次启动后立即修改默认密码
[ ] Token 过期时间 → Access Token 2h, Refresh Token 7d
```

### 10.2 网络安全架构

```
公网                        内网                         内网
┌──────────┐    HTTPS    ┌──────────┐    HTTP      ┌──────────┐
│ 浏览器    │───────────→│ Nginx    │───────────→│ api-rbac │
│ (SPA)    │←───────────│ (TLS终结) │←───────────│ :8087    │
└──────────┘             └──────────┘             └──────────┘
                               │
                               │ HTTP (内网)
                               ▼
                         ┌──────────┐
                         │ 业务系统  │
                         │ :5000    │
                         └──────────┘

原则:
- api-rbac 只监听内网地址 (127.0.0.1 或内网 IP)
- 不直接暴露到公网
- 业务系统通过 API Gateway / 反向代理统一入口
- 前端永远不直接调 api-rbac (通过业务系统中转)
```

### 10.3 Token 安全

```
存储:
  ✅ localStorage (SPA, 配合 XSS 防护)
  ✅ httpOnly Secure Cookie (SSR, 最佳)
  ❌ URL 参数 (会泄漏到日志/浏览器历史)

传输:
  ✅ 始终通过 HTTPS
  ✅ Authorization: Bearer <token> 头部
  ❌ 不要在 URL 中传递

刷新:
  ✅ Refresh Token 长时效 (7d)
  ✅ 前端拦截器: 检测 401 → 自动用 refresh_token 换新
  ✅ Access Token 短时效 (2h)

服务间调用:
  ✅ 用 X-API-Key 头部, 不用用户 Token
  ✅ API Key 在数据库中存储 SHA256 哈希 (不是明文)
```

### 10.4 权限命名规范

```
安全建议:
  ✅ resource:action 格式, 小写英文
  ✅ 用具体动词: execute, restart, approve, export
  ❌ 用过于宽泛的动词: manage, admin (拆分为具体操作)
  ❌ 用中文 (编码和比对容易出错)
```

---

---

## 11. 韧性设计：RBAC 宕机时的安全防护

### 11.1 问题分析

api-rbac 与业务系统分开部署，当 RBAC 服务宕机时，**默认行为是拒绝所有鉴权请求（fail-closed）**，这保证了安全性但影响了可用性。

```
正常模式:                               RBAC 宕机 (默认 FailMode.DENY):
───────                                 ──────────
请求 → @RequirePermission              请求 → @RequirePermission
  ├─ POST /auth/check ──→ RBAC OK        ├─ POST /auth/check ──→ RBAC ❌ (超时/连接拒绝)
  │  ← {allowed: true}                   │
  └─ 执行业务逻辑 ✅                      └─ 返回 502 "权限服务不可用" ❌ (安全, 但业务中断)
```

### 11.2 韧性方案设计

SDK 已内置三层防护机制：

```
第一层: 本地权限缓存 (FailMode.CACHE)
  → 登录时预加载用户权限到进程内存
  → RBAC 正常时，每次成功校验后异步更新缓存
  → 缓存 key = JWT payload 中解码的 user_id (非 token 字符串, 防止跨用户混淆)
  → Go 中间件: 所有路由共享同一个全局缓存 (避免重复填充)

第二层: 熔断器 (Circuit Breaker)
  → 连续 N 次失败后自动进入熔断状态
  → 熔断期间直接走本地缓存，不再等待 RBAC 超时
  → 30 秒后自动探测恢复

第三层: 安全兜底 (永远 fail-closed)
  → 本地缓存无该用户数据时 → 拒绝 (不冒安全风险)
  → 缓存过期 (默认 5 分钟) 后 → 拒绝
  → FailMode.DENY 模式下 → 永远拒绝
```

### 11.3 Python SDK 韧性用法

```python
from rbac_client import RBACClient, FailMode

# 韧性模式 (推荐用于生产环境)
rbac = RBACClient(
    "http://localhost:8087/api/v1",
    timeout=5,                          # 单次请求 5s 超时
    fail_mode=FailMode.CACHE,           # 故障时用本地缓存
    cache_ttl=300,                      # 缓存有效期 5 分钟
    circuit_breaker_threshold=5,        # 连续 5 次失败触发熔断
    circuit_breaker_recovery=30,        # 熔断 30s 后探测恢复
)

# 业务装饰器: 自动处理降级
def require_permission(resource, action):
    def decorator(f):
        @wraps(f)
        def wrapper(*args, **kwargs):
            token = request.headers.get("Authorization", "").replace("Bearer ", "")
            if not token:
                return jsonify({"code": 401, "message": "未登录"}), 401
            try:
                if not rbac.check_permission(token, resource, action):
                    return jsonify({"code": 403, "message": "无权限"}), 403
            except RuntimeError:
                # RBAC 完全不可达 + 缓存也不可用 → 安全拒绝
                return jsonify({"code": 502, "message": "权限服务不可用"}), 502
            return f(*args, **kwargs)
        return wrapper
    return decorator
```

### 11.4 Java 17+ SDK 韧性用法

```java
// 韧性模式
var rbac = new RBACClient(
    "http://localhost:8087/api/v1",
    FailMode.CACHE,    // 故障时用本地缓存
    300,               // 缓存 300 秒
    5,                 // 连续 5 次失败熔断
    30                 // 30 秒后探测恢复
);

// 登录时自动缓存权限
var login = rbac.login("admin", "password");

// RBAC 正常 → 远程校验; RBAC 宕机 → 自动降级到本地缓存
if (rbac.checkPermission(login.token(), "server", "restart")) {
    // 执行业务
}
```

### 11.5 Go SDK 韧性用法

```go
import "github.com/laazua/api-rbac/pkg/client"

// 使用 ResilientGuard 替代原 PermissionGuard
// 所有 ResilientGuard 实例共享同一个全局缓存和熔断器
r.Use(client.ResilientGuard(
    rbacClient,
    client.FailModeCache,  // 故障时用缓存
    300,                    // 缓存 300 秒
    "server", "restart",
))
r.Use(client.ResilientGuard(
    rbacClient,
    client.FailModeCache,
    300,
    "server", "stop",
))
// 以上 2 个路由共享同一个本地缓存 — 第一个成功请求填充缓存后，第二个路由也可命中
```

### 11.6 故障模式选择指南

| 场景 | 推荐 FailMode | 理由 |
|------|--------------|------|
| **金融/支付系统** | `DENY` | 任何不确定状态都应拒绝，安全优先 |
| **内部管理系统** | `CACHE` | 可接受短暂降级，可用性优先 |
| **读多写少场景** | `CACHE` (配合长 TTL) | 权限变更频率低，缓存命中率高 |
| **高安全合规场景** | `DENY` | 不可接受任何未经验证的访问 |

### 11.7 关键安全保证

> ⚠️ **无论何种 FailMode，以下安全底线永不突破:**

```
1. 本地缓存中没有该用户数据 → 拒绝 (不猜)
2. 本地缓存超过 TTL (默认 300s) → 拒绝 (不信任过期数据)
3. FailMode.DENY 模式下 → RBAC 宕机即拒绝一切
4. 熔断恢复后的第一次请求 → 必须通过 RBAC 校验 (不自动信任缓存)
5. 登录接口 → 必须通过 RBAC (不走缓存, 不降级)
6. Token 验证接口 → 必须通过 RBAC (不走缓存, 不降级)
7. 缓存 key = JWT payload 中的 user_id (非 token 字符串, 防跨用户混淆)
8. Go 中间件: 全局缓存, 所有路由共享, 熔断器全局统一
9. API Key 认证: RequirePermission 直接放行 (受信任内部服务)
```

---

## 12. 常见问题排查 (FAQ)

### Q1: 业务系统启动报警 "权限服务异常"

```
原因: 业务系统无法连接 api-rbac
排查:
  1. curl http://localhost:8087/health 确认 api-rbac 正常运行
  2. 检查 RBAC_URL 配置是否正确 (包含 /api/v1)
  3. 检查两个服务是否在同一网络 (Docker 内注意容器名)
```

### Q2: 登录成功但调用接口返回 401

```
原因: Token 未携带或格式错误
排查:
  1. 检查 Header: Authorization: Bearer <token> (注意空格和大小写)
  2. Access Token 默认 2h 过期, 检查是否过期
  3. 使用 /auth/introspect 验证 Token 有效性
```

### Q3: 所有操作返回 403 "无权限"

```
原因: 用户未被分配角色, 或角色未绑定权限
排查链路:
  1. 查看用户绑定了哪些角色:  GET /users/:id (响应含 roles 字段)
  2. 查看角色绑定了哪些权限: GET /roles/:id (响应含 permissions 字段)
  3. 检查 resource/action 字符串是否与代码中完全一致 (大小写敏感!)
```

### Q4: 新增模块后权限不生效

```
排查:
  1. 确认 api-rbac 中已创建权限 (检查 resource + action 拼写)
  2. 确认角色已绑定该权限
  3. 确认用户已分配该角色
  4. 检查代码中的 resource/action 与 api-rbac 中完全一致
     (例: "server" != "Server" != "servers")
```

### Q5: Token 频繁过期

```
解决方案:
  1. 使用 /auth/refresh 接口静默续期
  2. 前端拦截器: 在 Access Token 过期前 5 分钟自动刷新
  3. 增大 expire_hour (config.yaml 中配置)
```

### Q6: 高并发下权限检查慢

```
原因: 未启用 Redis 缓存
解决:
  1. 启动 Redis
  2. 在 config.yaml 中配置 redis 连接
  3. 重启 api-rbac, 确认日志: "Redis 连接成功，权限缓存已启用"
  4. 缓存后每次鉴权从 ~10ms (3表JOIN) 降到 ~1ms (Redis GET)
```

### Q7: Java 8 SDK 能在 Java 21 上用吗?

```
能。Java 向后兼容, Java 8 编译的代码可以在 Java 21/25 上正常运行。
推荐 Java 17+ 项目使用 sdk/java17/ 版 (利用 HttpClient + Record)。
```

### Q8: 如何恢复误删的用户?

```
api-rbac 使用软删除 (GORM DeletedAt)。
恢复方法:
  UPDATE users SET deleted_at = NULL WHERE id = <user_id>;
```

---

## 附录 A: 错误码对照表

| 错误码 | 常量 | HTTP 状态 | 消息 |
|--------|------|-----------|------|
| 0 | `Success` | 200 | success |
| 1001 | `InvalidParams` | 400 | 参数错误 |
| 1002 | `Unauthorized` | 401 | 未授权 |
| 1003 | `Forbidden` | 403 | 无权限 |
| 1004 | `NotFound` | 404 | 资源不存在 |
| 1005 | `InternalError` | 500 | 服务器内部错误 |
| 1006 | `AlreadyExists` | 409 | 资源已存在 |
| 1007 | `TokenExpired` | 401 | Token已过期 |
| 1008 | `TokenInvalid` | 401 | Token无效 |
| 1009 | `PasswordWrong` | 401 | 密码错误 |
| 1010 | `UserDisabled` | 401 | 用户已被禁用 |
| 1011 | `DBError` | 500 | 数据库错误 |

---

## 附录 B: 项目文件清单

```
api-rbac/
├── cmd/server/main.go                    # 服务入口
├── config/                               # 配置
├── internal/
│   ├── handler/                          # 6 个 HTTP Handler
│   ├── service/                          # 6 个 Service
│   ├── repository/                       # 4 个 Repository
│   ├── model/                            # 6 个模型文件
│   ├── middleware/                       # 4 个中间件
│   ├── router/                           # 路由定义
│   └── cache/                            # Redis 缓存
├── pkg/
│   ├── errcode/                          # 错误码 + HTTP 状态映射
│   ├── response/                         # 统一响应格式
│   ├── jwt/                              # JWT 生成/解析
│   └── client/                           # Go SDK
├── migrations/                           # SQL 参考
├── sdk/                                  # 多语言 SDK
│   ├── python/                           # Python 3.8+
│   ├── nodejs/                           # Node.js 14+
│   ├── java/                             # Java 8+
│   └── java17/                           # Java 17+
├── examples/
│   ├── python-ops-system/               # Python Flask 完整示例
│   └── java-ops-system/                  # Java Spring Boot 完整示例
├── docs/                                 # Swagger 文档
├── web/                                  # Vue2 管理前端
├── CLAUDE.md
├── README.md
├── usage.md                              # 使用说明 (早期版)
└── DOCS.md                               # ← 本文档
```

---

## 附录 C: Go 服务内部详解

### C.1 依赖注入链 (main.go)

```
main.go 初始化顺序:

1. config.Load()           → 配置
2. jwtpkg.Init()           → JWT 初始化
3. gorm.Open(mysql...)    → 数据库连接
4. redis.NewClient()       → Redis 连接 (可选)
5. db.AutoMigrate()        → 自动建表
6. initSuperAdmin()        → 首次运行创建 admin 用户
7. repository.New*Repo()   → 数据访问层
8. service.New*Service()   → 业务逻辑层
9. handler.New*Handler()   → HTTP 处理器
10. initDefaultServiceAccount()  → 首次运行创建默认服务账号
11. router.Setup()         → 路由注册
12. srv.ListenAndServe()   → 启动 HTTP
13. gracefulShutdown()     → 等待 SIGTERM
```

### C.2 权限检查流程 (permission_check.go)

```go
func (s *PermissionCheckService) CheckPermission(userID uint, resource, action string) (bool, error) {
    // 1. 尝试从 Redis 缓存获取
    perms, err := s.cache.GetUserPermissions(ctx, userID)
    if err == nil && perms != nil {
        return matchPermissionMap(perms, resource, action), nil  // ← 命中缓存
    }

    // 2. 未命中 → 查 DB
    user, err := s.userRepo.FindByID(userID)    // Preload(Roles.Permissions)
    // → SELECT * FROM users
    //   LEFT JOIN user_roles ON user_roles.user_id = users.id
    //   LEFT JOIN roles ON roles.id = user_roles.role_id
    //   LEFT JOIN role_permissions ON role_permissions.role_id = roles.id
    //   LEFT JOIN permissions ON permissions.id = role_permissions.permission_id

    // 3. 聚合权限 + 回填缓存
    permMap := aggregatePermissions(user.Roles)
    s.cache.SetUserPermissions(ctx, userID, permMap)  // TTL: 5分钟

    return matchPermissionMap(permMap, resource, action), nil
}
```

### C.3 缓存失效机制

```go
// 用户角色变更时 → 清除该用户的缓存
func (s *UserService) AssignRoles(id uint, roleIDs []uint) error {
    // ... 更新角色绑定
    s.cache.InvalidateUser(context.Background(), id)  // DEL rbac:user:{id}:perms
}

// 角色权限变更时 → 清除所有拥有该角色的用户的缓存
func (s *RoleService) AssignPermissions(id uint, permIDs []uint) error {
    // ... 更新权限绑定
    userIDs := s.roleRepo.FindUserIDsByRoleID(id)     // SELECT user_id FROM user_roles
    s.cache.InvalidateUsers(context.Background(), userIDs)
}
```

### C.4 JWT 结构

```go
type Claims struct {
    UserID    uint   `json:"user_id"`
    Username  string `json:"username"`
    TokenType string `json:"token_type"`  // "access" 或 "refresh"
    jwt.RegisteredClaims                  // ExpiresAt, IssuedAt
}

// Access Token:  HS256 签名, TokenType="access",  默认 2h
// Refresh Token: HS256 签名, TokenType="refresh", 默认 7d
// /auth/refresh 只接受 TokenType="refresh" 的 Token
```

---

> 📄 **文档版本**: 2.0 | **生成日期**: 2026-06-13 | **适用 api-rbac 版本**: 2.0+
>
> 本文档覆盖 api-rbac 权限管理系统的完整原理、API 参考、多语言 SDK 用法和集成示例。如需进一步技术支持，请参考项目 Swagger 文档 (`/swagger/index.html`) 或各 `sdk/` 和 `examples/` 目录下的详细 README。
