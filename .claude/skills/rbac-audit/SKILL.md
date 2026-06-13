---
name: rbac-audit
description: 全面审查 api-rbac 项目的安全漏洞、编码质量与性能优化，生成审计报告。
---

# RBAC 项目安全与代码质量审计

你是一名 Go 安全审计专家，专注于 RBAC（基于角色的访问控制）系统的安全审查和代码优化。

## 项目概述

api-rbac 是一个独立的、通用的 RBAC 权限管理微服务（Go + Gin），以 HTTP REST API 形式运行，与业务代码完全解耦，为任何编程语言的业务系统提供统一的权限管理能力。

### 核心能力

- 用户认证 (JWT + bcrypt) + Token 刷新 (Access 2h / Refresh 7d)
- 用户/角色/权限 CRUD + 多对多绑定
- 单次/批量权限检查 + Token 自省 (introspect)
- 服务间认证 (API Key / Service Account)
- Redis 权限缓存 (可选,降级可用)
- 软删除 (GORM DeletedAt)
- 多语言 SDK: Go / Python / Node.js / Java 8+ / Java 17+
- 韧性设计: FailMode (DENY/CACHE) + 熔断器 + 本地缓存

### 项目技术栈

- **语言:** Go 1.21+
- **模块路径:** `github.com/laazua/api-rbac`
- **Web框架:** Gin v1.12
- **ORM:** GORM v1.31 + MySQL
- **认证:** JWT (golang-jwt/jwt/v5, HMAC-SHA256) + bcrypt
- **缓存:** Redis (go-redis/v9, 可选)
- **配置:** Viper (spf13/viper, YAML)
- **文档:** Swagger (swaggo)

---

## 完整分层架构

```
cmd/server/main.go             # 入口: 启动 + DI + 优雅关闭 + initSuperAdmin
config/                        # 配置 (YAML + Viper)
internal/
  handler/                     # HTTP 处理器 (6个)
  │ ├── auth_handler.go        #   /auth/login,refresh,introspect,check,batch-check,verify,menu
  │ ├── user_handler.go        #   /users CRUD + 角色绑定
  │ ├── role_handler.go        #   /roles CRUD + 权限绑定
  │ ├── permission_handler.go  #   /permissions CRUD
  │ └── service_account_handler.go  # /service-accounts CRUD
  service/                     # 业务逻辑层 (6个)
  │ ├── auth_service.go        #   登录验证 (bcrypt) + Token 签发
  │ ├── user_service.go        #   用户 CRUD + 缓存失效
  │ ├── role_service.go        #   角色 CRUD + 批量缓存失效
  │ ├── permission_service.go  #   权限 CRUD
  │ ├── permission_check.go   #   权限检查 (缓存优先 + 通配符 + 批量)
  │ └── service_account_service.go  # API Key 生成 (SHA256)
  repository/                  # 数据访问层 (4个, GORM)
  │ ├── user_repo.go
  │ ├── role_repo.go           #   含 FindUserIDsByRoleID
  │ ├── permission_repo.go
  │ └── service_account_repo.go
  model/                       # 数据模型 (6个)
  │ ├── base.go                #   BaseModel (ID + CreatedAt + UpdatedAt + DeletedAt)
  │ ├── user.go                #   User + LoginRequest/RefreshTokenRequest/...
  │ ├── role.go
  │ ├── permission.go          #   Permission + BatchCheckPermissionRequest
  │ ├── service_account.go
  │ └── introspect.go          #   IntrospectRequest/Response
  middleware/                   # 中间件 (4个)
  │ ├── auth.go                #   JWT + API Key 双认证
  │ ├── cors.go
  │ ├── logger.go
  │ └── permission.go          #   RequirePermission 鉴权中间件
  router/router.go             # 路由注册
  cache/permission_cache.go   # Redis 权限缓存 (TTL + 批量失效)
pkg/
  errcode/errcode.go           # 错误码 (0~1011) + HTTP状态映射
  response/response.go         # 统一 JSON 响应 (含分页)
  jwt/jwt.go                   # JWT 生成/解析 (Access + Refresh + TokenType)
  client/                      # Go SDK (外部业务系统集成)
  │ ├── rbac_client.go         #   HTTP 客户端 (login/refresh/check/batch/introspect/getMenu)
  │ ├── middleware.go           #   PermissionGuard (Gin)
  │ └── resilient_middleware.go #   ResilientGuard (FailMode + 熔断 + 本地缓存)
config/
  ├── config.go                # 配置结构体 (Server/DB/Redis/JWT/CORS)
  ├── config.yaml              # 运行配置 (敏感: 应被 .gitignore)
  └── config.example.yaml      # 配置模板
migrations/                    # SQL 参考 (GORM AutoMigrate 为主)
  ├── 001_init.sql
  └── 002_soft_delete.sql
sdk/                           # 多语言 SDK
  ├── python/rbac_client.py    # Python 3.8+ (含 FailMode + 熔断)
  ├── nodejs/src/index.js      # Node.js 14+
  ├── java/                    # Java 8+ (javax.servlet + HttpURLConnection)
  └── java17/                  # Java 17+ (jakarta.servlet + HttpClient + Record)
examples/
  ├── python-ops-system/       # Python Flask 完整示例 (含前端 SPA)
  ├── java-ops-system/         # Java Spring Boot 完整示例
  └── go-ops-system/           # Go net/http 完整示例 (含韧性中间件)
web/                           # Vue 2 + Element UI 管理前端
```

---

## 审计流程 (必须严格按顺序执行)

### 第一阶段: 信息收集

1. 读取 `DOCS.md` 了解项目全局架构和设计原理
2. 读取 `config/config.yaml` 检查是否有**明文敏感信息** (DB密码、JWT密钥、内网IP、Redis密码)
3. 检查 `config/config.yaml` 是否在 `.gitignore` 中
4. 运行 `git log --oneline -15` 了解最近的变更历史
5. 检查 `go.mod` 中的依赖版本是否存在已知漏洞 (对比 CVE 数据库)
6. 列出所有 Go 源文件: `find . -name "*.go" ! -path "./web/*" ! -path "./docs/*" ! -path "./examples/*"`

### 第二阶段: 逐领域安全审查

#### A. 认证与授权 (CRITICAL)

- [ ] JWT secret 是否为默认值或硬编码 (`config.yaml` 中 `jwt.secret`)
- [ ] JWT 是否缺少 `jti` (JWT ID) — 无 jti 则无法实现令牌撤销
- [ ] JWT Claims 是否有 `TokenType` 字段区分 access/refresh (`pkg/jwt/jwt.go`)
- [ ] `/auth/refresh` 是否拒绝 access token (只接受 refresh token)
- [ ] 密钥轮换是否可行 — 当前 `pkg/jwt/jwt.go` 使用包级变量
- [ ] 登出是否真正使令牌失效 — 当前为无状态 JWT，无黑名单
- [ ] 密码是否使用 bcrypt 哈希存储 (`json:"-"` 标记)
- [ ] bcrypt cost 是否 ≥ 12 (`internal/service/auth_service.go`)
- [ ] 登录接口是否有速率限制 — 当前无
- [ ] 登录错误消息是否一致 (防用户枚举: "用户名或密码错误")
- [ ] `initSuperAdmin` 在无 TTY 环境是否挂起 (`cmd/server/main.go`)
- [ ] API Key (Service Account) 的权限模型是否安全 — `middleware/auth.go` 中 `user_id=0`
- [ ] API Key 是否使用 SHA256 哈希存储 (仅创建时返回一次明文)
- [ ] `RequirePermission` 中间件是否正确处理 API Key 认证类型

#### B. 输入验证与注入 (HIGH)

- [ ] SQL LIKE 查询中 `%` / `_` 是否正确转义 (`repository/*_repo.go`)
- [ ] `strconv.ParseUint` / `strconv.Atoi` 的错误是否检查 (严禁 `_` 丢弃)
- [ ] 类型断言是否有安全检查 (`v, ok := x.(uint)` vs `x.(uint)`)
- [ ] 请求体大小是否有限制 (Gin 默认无限制)
- [ ] 分页参数上界: `pageSize ≤ 100`, `offset ≤ 10000`
- [ ] 批量操作数组长度限制: `batch-check` 限制 50, `AssignRoles` 无限制
- [ ] 是否存在路径穿越风险

#### C. 配置与部署 (HIGH)

- [ ] `config.yaml` 是否包含明文密码且被 git 跟踪
- [ ] CORS 是否设置为 `["*"]`
- [ ] `server.mode` 是否为 `release` (生产环境)
- [ ] 是否缺少 HTTPS/TLS 支持
- [ ] 数据库 DSN 是否包含 `parseTime=True&loc=Local`
- [ ] 健康检查 `/health` 是否暴露敏感信息
- [ ] Swagger 在生产环境是否可访问 (`/swagger/*any` 无模式保护)
- [ ] Redis 配置是否正确 (Host/Port/Password/DB/PoolSize)

#### D. 代码质量 (MEDIUM)

- [ ] 是否存在 `_` 丢弃错误返回值
- [ ] 是否存在硬编码错误字符串 (应使用哨兵错误 `errors.Is`)
- [ ] 是否存在包级可变全局变量 (`pkg/jwt/jwt.go` 的 secret/expireHour)
- [ ] GORM `AutoMigrate` 是否替代了版本化迁移
- [ ] 日志是否结构化 (requestID, userID, clientIP, latency)
- [ ] 是否缺少审计日志 (登录成功/失败、权限变更等)
- [ ] 错误响应 HTTP 状态码是否正确映射 (`errcode.ToHTTPStatus`)
- [ ] 是否存在未使用的 import 或死代码注释
- [ ] 是否删除 `permission_check.go:110-111` 的过时注释

#### E. 并发与性能 (LOW)

- [ ] 是否存在 goroutine 泄漏 (Redis 连接、缓存操作)
- [ ] 数据库连接池配置: `MaxOpenConns` / `MaxIdleConns` / `ConnMaxLifetime`
- [ ] 是否有 N+1 查询 (权限检查 Preload 三层关联)
- [ ] 热点路径缓存: `PermissionCache` (Redis) + SDK 本地缓存 (韧性层)

#### F. 客户端 SDK (pkg/client/)

- [ ] 请求字段名与服务端一致: Login 用 `account` (不是 `username`)
- [ ] 超时配置是否合理 (默认 10s)
- [ ] 是否支持批量检查和 Token 自省
- [ ] 是否提供韧性中间件 `ResilientGuard` (FailMode + 熔断 + 本地缓存)
- [ ] 错误信息是否向调用方泄露内部实现细节

#### G. 多语言 SDK 与示例 (新增)

- [ ] Python SDK (`sdk/python/`): FailMode + 熔断器 + 本地缓存是否实现正确
- [ ] Java 8+ SDK (`sdk/java/`): HttpURLConnection 超时和错误处理
- [ ] Java 17+ SDK (`sdk/java17/`): HttpClient + Record + FailMode
- [ ] Node.js SDK (`sdk/nodejs/`): fetch 超时 + 错误处理
- [ ] 各 SDK 的 `checkPermission` 在 403 时返回 `false` 而非抛异常
- [ ] 示例代码 (`examples/`) 是否正确演示韧性降级

#### H. 韧性设计 (新增) — RBAC 宕机时的安全防护

- [ ] SDK 默认 FailMode 是否为 DENY (安全优先)
- [ ] FailMode.CACHE 模式下，缓存无数据是否安全拒绝
- [ ] 缓存 TTL 是否合理 (默认 300s)
- [ ] 熔断器阈值是否合理 (默认 5 次)
- [ ] 熔断恢复时间是否合理 (默认 30s)
- [ ] 登录接口是否不受缓存降级影响 (必须直连 RBAC)
- [ ] 本地缓存的 `checkFromCache` 是否与 `matchPermission` 逻辑一致 (通配符)

---

## 关键检查点速查表 (新增项以 ★ 标记)

| 优先级 | 文件 | 关注点 |
|--------|------|--------|
| P0 | `config/config.yaml` | 明文密码、默认JWT密钥、CORS `*`、Redis密码 |
| P0 | `pkg/jwt/jwt.go` | 包级变量、无 jti、无密钥轮换 |
| P0 | `cmd/server/main.go` | 无速率限制、initSuperAdmin TTY挂起、连接池未配置 |
| P0 | `internal/middleware/auth.go` | ★ API Key `user_id=0` 权限模型 |
| P0 | `internal/service/permission_check.go` | ★ 通配符返回硬编码集合、过时注释 |
| P1 | `pkg/client/rbac_client.go` | `username` vs `account` 字段名、★ 超时/重试 |
| P1 | `internal/repository/*_repo.go` | LIKE 通配符未转义 |
| P1 | `internal/handler/*_handler.go` | `_` 丢弃 ParseUint、类型断言无安全检查 |
| P1 | `internal/handler/auth_handler.go` | ★ Logout 无令牌撤销、Refresh 不检查用户状态 |
| P1 | ★ `internal/cache/permission_cache.go` | Redis 连接失败时行为、TTL 合理性 |
| P1 | ★ `pkg/client/resilient_middleware.go` | 熔断逻辑正确性、缓存 populate 时机 |
| P2 | `pkg/response/response.go` | HTTP 状态码映射 |
| P2 | 全项目 | 零测试覆盖率、无结构化日志、bcrypt cost=10 |
| P2 | ★ `sdk/` 目录 | SDK 版本间一致性、FailMode 默认值 |
| P2 | ★ `examples/` 目录 | 示例是否正确演示 fail-closed |

---

## 第三阶段: 代码优化建议

对发现的问题代码，提供具体的修复方案:

1. **安全漏洞** → 必须修复, 提供完整的修复代码
2. **代码异味** → 建议修复, 提供重构方案
3. **性能优化** → 可选修复, 评估收益

---

## 第四阶段: 生成审计报告

输出为 `AUDIT_REPORT.md`，使用以下格式:

```markdown
# api-rbac 安全与代码质量审计报告
> **审计日期** | **审计范围** | **项目版本**

## 🔴 严重风险 (必须立即修复)
## 🟠 高危风险 (应尽快修复)
## 🟡 中危风险 (建议修复)
## 🔵 代码优化建议
## ✅ 安全实践 (已有的良好实践)
## 修复优先级路线图
```

---

## 输出要求

- 所有回复使用**中文**
- 每条发现必须包含: **文件路径:行号**、**风险等级**、**问题描述**、**修复建议**
- 修复建议应包含可直接使用的代码示例
- 审计报告末尾给出**优先级排序的修复路线图** (分 Week 1-4)
- 报告写入项目根目录 `AUDIT_REPORT.md`
