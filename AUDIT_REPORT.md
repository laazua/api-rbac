# api-rbac 安全与代码质量审计报告

> **审计日期**: 2026-06-15 | **审计范围**: 全项目 (38 Go 源文件 + 15 前端文件 + 多语言 SDK) | **项目版本**: 2.0+

---

## 执行摘要

本次审计覆盖了 api-rbac 项目的 **全部 38 个 Go 源文件** + **15 个前端文件** + **6 个 SDK/示例目录**，识别出 **6 个严重安全漏洞**、**8 个高危风险**、**10 个中危问题**、**12 个代码优化建议**。

**最严重的问题**：API Key 认证完全绕过权限检查，任何持有有效 API Key 的服务账号拥有系统全部操作权限；通配符权限返回硬编码集合导致新增权限不被超级管理员覆盖；Refresh Token 不校验用户状态导致禁用用户仍可维持访问长达 7 天。

---

## 项目架构与数据流总结

### 后端分层架构

```
cmd/server/main.go          → 入口：DI 组装 + AutoMigrate + initSuperAdmin + 优雅关闭
internal/router/router.go   → 路由：公开接口 + JWT/APIKey 认证 + 权限中间件
internal/middleware/        → 认证(JWT+APIKey双重) + 鉴权(RequirePermission) + CORS + 日志
internal/handler/           → 6 个 Handler：参数绑定 → Service → 统一响应
internal/service/           → 6 个 Service：业务逻辑 + 缓存失效
internal/repository/        → 5 个 Repository：GORM 数据访问 + LIKE 转义
internal/cache/             → Redis 权限缓存 (TTL 5min)
pkg/jwt/                    → JWT 生成/解析 (HS256, Access 2h + Refresh 7d)
pkg/client/                 → Go SDK (HTTP客户端 + PermissionGuard + ResilientGuard)
```

### 前后端交互流

```
浏览器 (Vue2 SPA)
  │  POST /api/v1/auth/login
  ▼
api-rbac Go 服务 → bcrypt 验证 → 签发 JWT → 返回 {token, refresh_token}
  │
  │  GET /api/v1/users?page=1  (Authorization: Bearer <token>)
  ▼
middleware.AuthRequired → 解析 JWT → c.Set("user_id", ...)
middleware.RequirePermission("user", "read") → 查缓存/DB → 通配符匹配
Handler → Service → Repository → GORM → MySQL
  │
  ▼
统一 JSON 响应 → 前端渲染
```

### 权限检查核心流程

```
CheckPermission(userID, resource, action)
  ├─ Redis 缓存命中? → matchPermissionMap() → true/false
  └─ 缓存 miss → DB Preload(Roles.Permissions) → aggregatePermissions()
       ├─ 含 *:* → 返回硬编码权限集合
       └─ 不含 → 聚合所有 resource→[]action
       └─ 回填 Redis (TTL 5min)
```

---

## 🔴 严重风险 (必须立即修复)

### 1. API Key 认证完全绕过权限检查 — [CRITICAL]

**文件**: [internal/middleware/auth.go:31-37](internal/middleware/auth.go#L31-L37), [internal/middleware/permission.go:16-18](internal/middleware/permission.go#L16-L18)

**问题**: 使用 `X-API-Key` 认证时，`AuthRequired` 只设置 `auth_type=apikey` 和 `service_account_id`，**不设置 `user_id`**。`RequirePermission` 中间件检测到 `auth_type == "apikey"` 后**直接放行，不执行任何权限检查**。

```go
// middleware/auth.go:33 — API Key 认证不设置 user_id
c.Set("auth_type", "apikey")
c.Set("service_account_id", sa.ID)
// 没有 c.Set("user_id", ...) !!!

// middleware/permission.go:16-18 — 检测到 apikey 直接放行
if authType, _ := c.Get("auth_type"); authType == "apikey" {
    c.Next()  // ← 完全绕过权限检查!
    return
}
```

**影响**: 任何持有有效 API Key 的服务账号拥有**全部 API 的无限制访问权**，包括创建/删除用户、修改角色权限、创建新的 API Key 等。攻击者获取任意一个 API Key 即可完全控制整个系统。

**修复建议**: 服务账号应绑定角色和权限，或至少限制可访问的 endpoint 白名单。

```go
// 方案 A: 为 ServiceAccount 增加角色绑定
type ServiceAccount struct {
    BaseModel
    Name        string
    ApiKeyHash  string
    Roles       []Role `gorm:"many2many:service_account_roles;"`
}

// middleware/auth.go — 设置虚拟 user_id 用于权限检查
c.Set("auth_type", "apikey")
c.Set("user_id", sa.ID)  // 用 ServiceAccount ID 做权限查询
c.Set("is_service_account", true)

// middleware/permission.go — 移除直接放行逻辑，统一走权限检查
```

**优先级**: P0 — 必须立即修复

---

### 2. aggregatePermissions 通配符返回硬编码集合 — [CRITICAL]

**文件**: [internal/service/permission_check.go:85-93](internal/service/permission_check.go#L85-L93)

**问题**: 当用户拥有 `*:*` 超级管理员权限时，`aggregatePermissions` 返回一个**硬编码的权限集合**，而非真正的通配符语义。

```go
if hasWildcard(permMap) {
    return map[string][]string{
        "user":            {"read", "create", "update", "delete"},
        "role":            {"read", "create", "update", "delete"},
        "permission":      {"read", "create", "update", "delete"},
        "module":          {"read", "create", "update", "delete"},
        "service_account": {"read", "create", "update", "delete"},
    }
}
```

**影响**: 业务系统新增的任何权限（如 `log:read`、`deployment:execute`）**超级管理员都无法匹配**，因为硬编码列表中没有这些权限。虽然 `HasWildcard()` 单独查库解决了 `/auth/modules` 接口的问题，但 `CheckPermission` 和 `GetUserPermissions` 仍受此影响。更严重的是，这个硬编码 map 会被**回填到 Redis 缓存**中，污染缓存数据。

**修复建议**: `CheckPermission` 不应依赖展开后的 map，应在匹配逻辑中直接处理通配符。

```go
func (s *PermissionCheckService) CheckPermission(userID uint, resource, action string) (bool, error) {
    // 先检查是否为超级管理员（查缓存/DB）
    isSuperAdmin, err := s.checkWildcardCached(userID)
    if err != nil {
        return false, err
    }
    if isSuperAdmin {
        return true, nil  // 直接返回 true，不展开权限列表
    }

    perms, err := s.getUserPermissionsCached(userID)
    if err != nil {
        return false, err
    }
    return matchPermissionMap(perms, resource, action), nil
}
```

**优先级**: P0 — 必须立即修复

---

### 3. Refresh Token 不校验用户状态 — [CRITICAL]

**文件**: [internal/handler/auth_handler.go:89-119](internal/handler/auth_handler.go#L89-L119)

**问题**: `/auth/refresh` 仅解析 Refresh Token 的 JWT 签名和 TokenType，**不检查用户是否仍然存在、是否被禁用**。Refresh Token 有效期 7 天，在这期间即使管理员禁用了该用户，用户仍可通过 Refresh Token 不断获取新的 Access Token。

```go
func (h *AuthHandler) Refresh(c *gin.Context) {
    claims, err := jwtpkg.ParseRefreshToken(req.RefreshToken)
    // ↑ 只验证 JWT 签名 + TokenType，不查用户状态

    newToken, _ := jwtpkg.Generate(claims.UserID, claims.Username)
    newRefreshToken, _ := jwtpkg.GenerateRefreshToken(claims.UserID, claims.Username)
    // ↑ 直接签发新 Token，不检查用户是否被禁用/删除
}
```

**影响**: 管理员禁用用户后，该用户仍可在 7 天内（Refresh Token 有效期内）持续访问系统。结合无令牌撤销机制，这是目前唯一的"登出"手段也被绕过。

**修复建议**:

```go
func (h *AuthHandler) Refresh(c *gin.Context) {
    claims, err := jwtpkg.ParseRefreshToken(req.RefreshToken)
    // ... error handling ...

    // 检查用户状态
    user, err := h.authService.GetUserByID(claims.UserID)
    if err != nil || user.Status == 0 {
        response.ErrorWithMsg(c, errcode.UserDisabled, "用户已被禁用或不存在")
        return
    }

    // 继续签发新 Token...
}
```

**优先级**: P0 — 必须立即修复

---

### 4. 无令牌撤销机制 (JWT jti 缺失) — [CRITICAL]

**文件**: [pkg/jwt/jwt.go:15-20](pkg/jwt/jwt.go#L15-L20)

**问题**: JWT Claims 中**没有 `jti` (JWT ID)** 字段，无法实现令牌黑名单/撤销。登出接口 (`POST /auth/logout`) 是**空操作** — 仅返回 `{"code":0}`，不做任何服务端处理。

```go
type Claims struct {
    UserID    uint   `json:"user_id"`
    Username  string `json:"username"`
    TokenType string `json:"token_type"`
    jwt.RegisteredClaims  // 含 ExpiresAt, IssuedAt，但没有 ID (jti)
}
```

**影响**: 
- 用户登出后 Token 仍有效直到过期（最长 2 小时）
- 无法在检测到 Token 泄露后主动撤销
- 密码修改后旧 Token 仍然有效

**修复建议**: 添加 Redis 令牌黑名单或使用 jti + 版本号机制。

```go
type Claims struct {
    UserID    uint   `json:"user_id"`
    Username  string `json:"username"`
    TokenType string `json:"token_type"`
    JTI       string `json:"jti"`  // 新增
    jwt.RegisteredClaims
}

// Logout 时: 将 jti 加入 Redis 黑名单 (TTL = Token 剩余有效期)
// Refresh 时: 将旧的 refresh token jti 加入黑名单
// 中间件 AuthRequired: 解析后检查 jti 是否在黑名单
```

**优先级**: P0 — 必须立即规划并在下个版本实现

---

### 5. 配置文件含明文敏感信息 + 默认 JWT 密钥 — [CRITICAL]

**文件**: [config/config.yaml](config/config.yaml)

**问题**:
```yaml
db:
  host: 192.168.165.88
  password: "abc123456"                              # ← 明文内网数据库密码

jwt:
  secret: "your-secret-key-change-in-production"     # ← 默认密钥，如未修改则任何人均可伪造 JWT

redis:
  host: 192.168.165.88                               # ← 内网 IP 暴露

cors:
  allow_origins:
    - "*"                                             # ← 允许任意源跨域
```

虽然 `.gitignore` 中有 `config.yaml` 和 `config/config.yaml`，但这个文件**当前仍存在于磁盘**。默认 JWT 密钥是**教科书级别安全漏洞**。

**影响**: 如果生产部署使用了默认 JWT 密钥，攻击者可以**任意伪造任何用户的 JWT Token**，获得系统完全控制权。

**修复建议**:
- 立即轮换 JWT 密钥
- 通过环境变量覆盖敏感配置（`viper.AutomaticEnv()` 已启用但需显式绑定）
- 生产环境 CORS 改为具体域名
- 确认 config.yaml 已被 gitignore 生效且未被追踪

---

### 6. 登录接口无速率限制 — [CRITICAL]

**文件**: [internal/router/router.go:45](internal/router/router.go#L45)

**问题**: `POST /auth/login` 没有任何速率限制。攻击者可以无限尝试用户名/密码组合进行暴力破解。

```go
auth := api.Group("/auth")
{
    auth.POST("/login", authH.Login)    // ← 无速率限制
    auth.POST("/refresh", authH.Refresh)
    auth.POST("/introspect", authH.Introspect)
}
```

**影响**: 暴力破解用户密码。bcrypt 虽然慢速（cost=10），但不足以防止分布式暴力破解。

**修复建议**: 增加基于 IP + 账号的速率限制中间件。

```go
// 使用 golang.org/x/time/rate 或 gin 的限流中间件
auth.POST("/login", rateLimiter(5, time.Minute), authH.Login)
```

---

## 🟠 高危风险 (应尽快修复)

### 7. ResilientMiddleware 缓存 key 使用 Token 字符串而非 user_id — [HIGH]

**文件**: [pkg/client/resilient_middleware.go:155](pkg/client/resilient_middleware.go#L155)

**问题**: 全局缓存 `resilientCacheStore` 使用**原始 token 字符串**作为 key，与文档声称的 "key = JWT payload 中的 user_id" **直接矛盾**。

```go
resilientCacheStore[token] = cacheEntry{...}  // ← key 是 token 字符串!
```

**影响**:
- Token 刷新后缓存全部失效，必须重新通过 RBAC 远程校验才能填充
- 如果 RBAC 此时已宕机且 cache TTL 过期，即使之前已缓存也拒绝访问
- 不同 token（同用户）无法共享缓存，降低缓存命中率

**修复建议**: 解码 JWT payload（无需验证签名，只取 user_id）或要求调用方传入 userId。

```go
// 解码 JWT payload 取 user_id（不验证签名，仅提取声明）
func extractUserID(token string) (string, error) {
    parts := strings.Split(token, ".")
    if len(parts) != 3 { return "", errors.New("invalid token") }
    payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
    var claims struct { UserID uint `json:"user_id"` }
    json.Unmarshal(payload, &claims)
    return fmt.Sprintf("user:%d", claims.UserID), nil
}
```

**优先级**: P1 — 应尽快修复，与文档保持一致

---

### 8. ServiceAccountRepo.List LIKE 注入漏洞 — [HIGH]

**文件**: [internal/repository/service_account_repo.go:45](internal/repository/service_account_repo.go#L45)

**问题**: 其他所有 Repository 的 LIKE 查询都使用了 `escapeLike()` 转义 `%` 和 `_`，但 `ServiceAccountRepo.List` **遗漏了转义**。

```go
// 危险 — 未转义 LIKE 通配符
query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
```

对比其他 Repo 的正确做法:
```go
// 正确 — 使用 escapeLike
escaped := escapeLike(keyword)
query = query.Where("name LIKE ? OR description LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
```

**影响**: 攻击者可以通过搜索 `%` 字符绕过关键词匹配逻辑，虽然不会直接导致数据泄露，但可能绕过前端搜索限制获取全量数据。

**修复建议**: 统一使用 `escapeLike(keyword)`。

---

### 9. 登录泄露用户状态 — 用户枚举漏洞 — [HIGH]

**文件**: [internal/service/auth_service.go:48-50](internal/service/auth_service.go#L48-L50)

**问题**: 当用户名/邮箱存在但被禁用时，返回错误 "用户已被禁用"，而非统一的 "用户名或密码错误"。

```go
if user.Status == 0 {
    return nil, errors.New("用户已被禁用")  // ← 泄露了账户存在且被禁用
}
```

实际响应:
- 账户不存在 → `"用户名或密码错误"` (HTTP 401, code 1009)  
- 账户存在但密码错误 → `"用户名或密码错误"` (HTTP 401, code 1009)
- 账户存在但被禁用 → `"用户已被禁用"` (HTTP 401, code 1009) ← **可区分**

**影响**: 攻击者可以通过错误消息判断哪些用户名/邮箱在系统中已注册。

**修复建议**: 被禁用用户统一返回 "用户名或密码错误"。

```go
if user.Status == 0 {
    return nil, errors.New("用户名或密码错误")  // 统一错误消息
}
```

---

### 10. RoleHandler 两处丢弃 ParseUint 错误 — [HIGH]

**文件**: [internal/handler/role_handler.go:243-244](internal/handler/role_handler.go#L243-L244), [internal/handler/role_handler.go:266-267](internal/handler/role_handler.go#L266-L267)

**问题**: `RemoveModule` 和 `RemovePermission` 方法中使用 `_` 丢弃 `strconv.ParseUint` 的返回值。

```go
func (h *RoleHandler) RemoveModule(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 64)     // ← 错误被丢弃
    modID, _ := strconv.ParseUint(c.Param("modId"), 10, 64) // ← 错误被丢弃
    // 解析失败时 id=0, modID=0，可能导致意外删除或误操作
```

**影响**: 如果 URL 参数格式错误，会以 `id=0` 执行操作，可能产生意外副作用。

**修复建议**: 检查错误并返回 400。

```go
id, err := strconv.ParseUint(c.Param("id"), 10, 64)
if err != nil {
    response.Error(c, errcode.InvalidParams)
    return
}
```

---

### 11. Swagger 生产环境无保护 — [HIGH]

**文件**: [internal/router/router.go:38](internal/router/router.go#L38)

**问题**: `/swagger/*any` 在任何运行模式下（包括 `release`）都无条件可访问，无认证要求。

```go
r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

**影响**: 生产环境中暴露完整 API 文档和测试界面，增加攻击面。

**修复建议**: 生产模式禁用或在 Swagger 路由前加认证。

```go
if cfg.Server.Mode != gin.ReleaseMode {
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

---

### 12. bcrypt.DefaultCost = 10 — [HIGH]

**文件**: [internal/service/auth_service.go:75](internal/service/auth_service.go#L75), [internal/service/user_service.go:34](internal/service/user_service.go#L34), [cmd/server/main.go:259](cmd/server/main.go#L259)

**问题**: 整个项目使用 `bcrypt.DefaultCost`（值为 10）。OWASP 2025 推荐最低 cost 为 12。

**修复建议**: 全局统一使用常量。

```go
const BcryptCost = 12

bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
```

**注意**: 提高 cost 会增加登录延迟（cost=10 ~100ms, cost=12 ~400ms），需配合速率限制。

---

### 13. 无请求体大小限制 — [HIGH]

**问题**: Gin 框架默认不限制请求体大小。攻击者可以发送超大 JSON payload 耗尽服务器内存。特别是 `POST /auth/batch-check`（虽然限制 50 条但无 body size limit）和 `POST /auth/login`。

**修复建议**:

```go
r := gin.Default()
r.MaxMultipartMemory = 8 << 20  // 8MB
// 或使用中间件限制 Content-Length
```

---

### 14. AssignRoles 无数组长度限制 — [HIGH]

**文件**: [internal/model/user.go:43](internal/model/user.go#L43)

**问题**: `AssignRolesRequest.RoleIDs` 只标记 `binding:"required"`，没有 `max` 限制。攻击者可以发送包含数千个 role_id 的请求。

```go
type AssignRolesRequest struct {
    RoleIDs []uint `json:"role_ids" binding:"required"`  // ← 无 max 限制
}
```

**修复建议**:

```go
RoleIDs []uint `json:"role_ids" binding:"required,min=1,max=100"`
```

---

## 🟡 中危风险 (建议修复)

### 15. 错误处理使用字符串比较而非哨兵错误

**文件**: [internal/handler/user_handler.go:154](internal/handler/user_handler.go#L154), [internal/handler/user_handler.go:192](internal/handler/user_handler.go#L192)

多处使用 `strings.Contains(err.Error(), "不存在")` 或 `err.Error() == "旧密码错误"` 判断错误类型。这是脆弱的设计 — 重构错误消息时会静默失效。

**修复建议**: 定义包级哨兵错误。

```go
var (
    ErrUserNotFound    = errors.New("用户不存在")
    ErrWrongPassword   = errors.New("旧密码错误")
    ErrRoleNotFound    = errors.New("角色不存在")
    // ...
)
```

---

### 16. 包级可变全局变量

**文件**: [pkg/jwt/jwt.go:22-26](pkg/jwt/jwt.go#L22-L26)

```go
var (
    secret           string       // ← 可变全局变量，无法并发安全地轮换
    expireHour       int
    refreshExpireDay int
)
```

使用包级变量的 `Init()` 模式使得密钥轮换几乎不可能（需重启服务），且无法进行单元测试隔离。

**修复建议**: 将 JWT 相关函数改为方法，通过结构体传递配置。

---

### 17. JWT 使用 HS256 而非 RS256

**文件**: [pkg/jwt/jwt.go:49](pkg/jwt/jwt.go#L49)

HMAC-SHA256 要求签名和验证使用同一密钥。在微服务架构中，如果多个服务需要验证 JWT，每个服务都需要知道密钥，增加了密钥泄露风险。

**修复建议**: 考虑迁移到 RS256 (RSA) 或 ES256 (ECDSA) 非对称签名，使业务服务仅需公钥即可验证。

---

### 18. 零测试覆盖率

**验证**: 项目中没有任何 `*_test.go` 文件。

**影响**: 重构和修复安全漏洞时缺乏安全网，可能引入回归。

**建议**: 优先为以下模块编写测试:
1. `pkg/jwt/jwt.go` — Token 生成/解析
2. `internal/service/permission_check.go` — 通配符匹配逻辑
3. `internal/service/auth_service.go` — 登录验证
4. `internal/middleware/permission.go` — 权限中间件

---

### 19. 无结构化日志

**文件**: [internal/middleware/logger.go:21](internal/middleware/logger.go#L21)

```go
log.Printf("[%d] %s %s | %v", status, method, path, latency)
```

无 request ID、user ID、client IP、trace ID 等关键字段。

**修复建议**: 引入结构化日志库 (如 `slog` 标准库 或 `zerolog`)。

---

### 20. 无审计日志

安全关键操作没有审计记录:
- 登录成功/失败 (含 IP、User-Agent)
- 权限变更 (角色分配、权限绑定)
- 用户 CRUD
- API Key 创建

**修复建议**: 在 Service 层关键操作处添加审计日志写入。

---

### 21. PageSize 参数 binding max=1000 但 repo 截断为 100

**文件**: [internal/model/user.go:59-61](internal/model/user.go#L59-L61) vs [internal/repository/user_repo.go:74-76](internal/repository/user_repo.go#L74-L76)

binding 允许 `max=1000`，但 repository 层静默截断为 100。API 消费者请求 `page_size=500` 可能只收到 100 条，且无警告。

**修复建议**: 统一为 100 或在响应中告知截断。

---

### 22. 数据库密码在 DSN 中明文传输

**文件**: [config/config.go:44-46](config/config.go#L44-L46)

Go MySQL 驱动对密码无特殊保护，DSN 日志可能泄露密码。

**修复建议**: 使用 `go-sql-driver/mysql` 的 `AllowNativePasswords` 或配置文件加密方案。

---

### 23. 前端 Token 存储在 localStorage — XSS 风险

**文件**: [web/src/views/Login.vue:55](web/src/views/Login.vue#L55), [web/src/api/index.js:13](web/src/api/index.js#L13)

Token 存储在 `localStorage`，任何 XSS 漏洞可直接读取 Token。

**修复建议**: 考虑使用 `httpOnly Secure SameSite` Cookie 存储 Token（需要后端配合设置 Cookie）。

---

### 24. 初始化 API Key 明文输出到标准日志

**文件**: [cmd/server/main.go:398](cmd/server/main.go#L398)

```go
log.Printf("     API Key: %s", apiKey)
```

此 API Key（默认服务账号）拥有完全系统访问权。日志系统通常会采集 stdout，导致密钥泄露。

**修复建议**: 只输出 hash 或截断显示。

```go
log.Printf("     API Key (仅显示后4位): ...%s", apiKey[len(apiKey)-4:])
```

---

## 🔵 代码优化建议

### 25. GORM AutoMigrate 替代版本化迁移

生产环境中 `AutoMigrate` 无法处理列重命名、类型变更等复杂迁移。建议保留 `migrations/` 目录下的 SQL 脚本作为标准方案。

### 26. Go SDK GetMenu 重复设置 Authorization Header

**文件**: [pkg/client/rbac_client.go:210-216](pkg/client/rbac_client.go#L210-L216)

```go
req.Header.Set("Authorization", "Bearer "+token) // 第一次
for k, v := range headers {
    req.Header.Set(k, v)                           // headers 中也有 "Authorization"，覆盖第一次
}
```

可以删除第 210 行的手动设置。

### 27. auth_service.go 嵌套错误处理可读性差

**文件**: [internal/service/auth_service.go:30-46](internal/service/auth_service.go#L30-L46)

三层嵌套的 `if err != nil { if errors.Is(...) { ... } }` 难以维护。建议提取辅助函数扁平化。

### 28. 连接池配置硬编码

**文件**: [cmd/server/main.go:88-90](cmd/server/main.go#L88-L90)

```go
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

建议通过 config.yaml 配置这些值。

### 29. Redis 密码为空时无提示

`redis.password: ""` 是默认配置，如果 Redis 无密码保护且暴露在网络上，存在安全风险。建议 Redis 无密码时打印警告。

### 30. CORS 中间件对 `*` 模式缺少 `Vary: Origin` Header

当 `allow_origins` 不包含 `*` 时，CORS 中间件正确设置了 `Vary: Origin`。但当包含 `*` 时未设置，可能导致代理缓存问题。

### 31. User model Status 使用 int 而非枚举

**文件**: [internal/model/user.go:8](internal/model/user.go#L8)

```go
Status int `json:"status" gorm:"type:tinyint;default:1;comment:1=启用 0=禁用"`
```

使用 `int` 而非命名常量，降低了代码可读性。

### 32. `initSuperAdmin` 支持环境变量 `ADMIN_PASSWORD` 但代码中未实现

DOCS.md 提到可通过 `ADMIN_PASSWORD` 环境变量传入密码，但 `readPasswordFromTerminal()` 函数只支持交互式输入。

### 33. `ensureDefaultModule` 更新通配符权限的 module_id 时忽略错误

**文件**: [cmd/server/main.go:363-365](cmd/server/main.go#L363-L365)

```go
if err := tx.Model(&model.Permission{}).
    Where("resource = ? AND action = ?", "*", "*").
    Update("module_id", sysModule.ID).Error; err != nil {
    log.Printf("[初始化] 更新已有权限的模块关联失败（可忽略）: %v", err)
}
```

此错误被忽略可能导致超级管理员的权限不归属任何模块，影响前端模块门户展示。

### 34. 前端路由守卫仅依赖 localStorage 权限判断

**文件**: [web/src/router/index.js:92-98](web/src/router/index.js#L92-L98)

前端路由守卫只检查 `localStorage` 中的权限数据做菜单显隐，真正的安全由后端保证。但客户端 `permissions` 数据可能过期（刷新后未重新获取），导致菜单显示与实际权限不一致。

### 35. `handleLogin` 获取菜单失败被静默忽略

**文件**: [web/src/views/Login.vue:58-61](web/src/views/Login.vue#L58-L61)

```javascript
try {
    const menuRes = await getMenu()
    localStorage.setItem('permissions', JSON.stringify(menuRes.data.permissions || {}))
} catch { /* 忽略 */ }
```

如果获取菜单失败，`permissions` 不会被设置，导致所有需要权限检查的路由守卫失败，用户被重定向到 Dashboard。

### 36. 建议增加 Request ID 中间件

在请求入口生成 UUID 作为 `X-Request-ID`，贯穿整个调用链（日志、响应 Header），便于问题追踪。

---

## ✅ 安全实践 (已有的良好实践)

以下方面项目做得很好：

| 类别 | 实现 |
|------|------|
| **密码安全** | bcrypt 哈希存储，`json:"-"` 防止序列化泄露 |
| **API Key 安全** | SHA256 哈希存储，创建时仅返回一次明文 |
| **JWT 类型区分** | `TokenType` 字段区分 Access/Refresh，`ParseRefreshToken` 拒绝 Access Token |
| **软删除** | GORM DeletedAt，数据可恢复 |
| **LIKE 注入防护** | `escapeLike()` 函数转义 `%` `_` `\`（除 ServiceAccountRepo 外均已使用） |
| **分页保护** | pageSize ≤ 100, offset ≤ 10000 |
| **批量限制** | batch-check 限制 50 项 |
| **缓存失效** | 用户角色变更 → 失效该用户缓存；角色权限变更 → 失效所有关联用户缓存 |
| **Redis 降级** | Redis 连接失败时降级为直接查库，不影响服务可用性 |
| **优雅关闭** | `SIGINT/SIGTERM` → 30s 等待现有请求完成 |
| **CORS 白名单模式** | 非 `*` 时精确匹配 Origin + 设置 `Vary: Origin` |
| **API Key 禁用检查** | `FindByApiKeyHash` 查询条件包含 `status = 1` |
| **角色删除保护** | 有关联用户时禁止删除角色 |
| **权限删除保护** | 有关联角色时禁止删除权限 |
| **模块删除保护** | 有关联权限时禁止删除模块 |
| **软删除恢复** | 创建同名资源时自动恢复已软删除记录 |
| **覆盖式分配** | `Association.Replace` 覆盖式更新关联，避免残留旧数据 |
| **韧性设计** | FailMode DENY/CACHE + 熔断器 + 本地缓存 |
| **类型断言安全** | middleware/permission.go 使用 `v, ok := x.(uint)` |
| **前端统一错误处理** | axios 拦截器统一处理 401/403 等状态码 |
| **多语言 SDK** | Go/Python/Node.js/Java 8+/Java 17+ 全覆盖 |

---

## 修复优先级路线图

### Week 1 (紧急 — 阻塞上线)

1. **[P0]** 修复 API Key 完全绕过权限检查 — 增加服务账号角色绑定
2. **[P0]** 修复 aggregatePermissions 硬编码通配符 — 使用真正的通配符匹配
3. **[P0]** Refresh Token 增加用户状态检查
4. **[P0]** 轮换 config.yaml 中所有默认密钥/密码
5. **[P0]** 增加登录接口速率限制
6. **[P0]** 确认 config.yaml 已被 git 忽略且未被追踪

### Week 2 (尽快)

7. **[P1]** ResilientMiddleware 缓存 key 改为 user_id
8. **[P1]** ServiceAccountRepo.List 增加 LIKE 转义
9. **[P1]** 登录错误消息一致化（防用户枚举）
10. **[P1]** RoleHandler.RemoveModule/RemovePermission 错误处理
11. **[P1]** Swagger 生产环境禁用
12. **[P1]** 请求体大小限制
13. **[P1]** AssignRoles 数组长度限制
14. **[P1]** bcrypt cost 提升至 12

### Week 3-4 (建议)

15. **[P2]** 哨兵错误替换字符串比较
16. **[P2]** JWT 包重构（消除全局变量）
17. **[P2]** 结构化日志 + 审计日志
18. **[P2]** 编写核心模块单元测试
19. **[P2]** 令牌撤销机制 (jti + 黑名单)

### 持续改进

20. **[P3]** 优化代码可读性（嵌套错误处理等）
21. **[P3]** 连接池参数可配置化
22. **[P3]** 前端安全增强（httpOnly Cookie 评估）
23. **[P3]** initSuperAdmin 支持环境变量密码
24. **[P3]** 评估 RS256 JWT 迁移可行性

---

> 📄 **审计工具**: Claude Code + rbac-audit Skill | **审计者**: laazua
> 
> 如需对任何发现进行深入分析或获取修复代码的完整实现，请提出具体要求。
