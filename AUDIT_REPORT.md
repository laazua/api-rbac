# api-rbac 安全与代码质量审计报告

> **审计日期**: 2026-06-13 | **审计范围**: 41 个 Go 源文件 | **项目版本**: 2.0

---

## 🔴 严重风险 (必须立即修复)

### CRIT-01: 配置文件包含明文敏感信息且被 Git 跟踪

**文件**: `config/config.yaml:1-28` | **风险**: 认证绕过 / 数据泄露

```
问题:
  - 数据库密码明文: "abc123456"
  - JWT secret 为默认值: "your-secret-key-change-in-production"
  - CORS 配置为 ["*"]
  - Redis 无密码
  - 内网 IP 暴露: 192.168.165.88
```

**影响**: 任何人获取此文件即可连接数据库、伪造 JWT Token、完全接管系统。

**修复**: 
```yaml
# config/config.yaml — 使用环境变量替代敏感值
db:
  password: ${DB_PASSWORD}        # 从环境变量读取
jwt:
  secret: ${JWT_SECRET}           # 从环境变量读取
redis:
  password: ${REDIS_PASSWORD}

# .env 文件 (加入 .gitignore)
DB_PASSWORD=<production-password>
JWT_SECRET=<64-char-random-string>
```
并确认 `config.yaml` 在 `.gitignore` 中。

---

### CRIT-02: JWT 使用包级可变全局变量

**文件**: `pkg/jwt/jwt.go:16-18` | **风险**: 并发安全隐患 / 热重载无法实现

```go
var (
    secret     string           // ← 包级可变变量，无并发保护
    expireHour int
    refreshExpireDay int
)
```

**影响**: 虽然当前 Init() 在启动时调用一次，但如果有热重载需求，并发读写会导致数据竞争。更重要的是，**无法支持密钥轮换**——轮换密钥需要同时接受新旧两个密钥。

**修复**:
```go
type JWTManager struct {
    secrets         [][]byte     // 支持多个密钥 (轮换)
    currentSecret   int          // 当前活跃密钥索引
    expireHour      int
    refreshExpireDay int
    mu              sync.RWMutex
}

func NewJWTManager(secrets []string, expireHour, refreshDay int) *JWTManager {
    mgr := &JWTManager{expireHour: expireHour, refreshExpireDay: refreshDay}
    for _, s := range secrets {
        mgr.secrets = append(mgr.secrets, []byte(s))
    }
    return mgr
}

// Parse 时依次用所有密钥尝试解析，支持密钥轮换过渡期
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
    var lastErr error
    for _, key := range m.secrets {
        claims, err := parseWithKey(tokenString, key)
        if err == nil { return claims, nil }
        lastErr = err
    }
    return nil, lastErr
}
```

---

### CRIT-03: API Key 认证授予超级管理员权限

**文件**: `internal/middleware/auth.go:34-35` | **风险**: 权限提升

```go
// 服务账号视为已认证，user_id 设为 0（超级管理员级别）
c.Set("user_id", uint(0))
```

**影响**: 所有 Service Account 被授予 `user_id=0`。虽然当前系统中 admin 用户是 `user_id=1`，但 `user_id=0` 不是任何合法用户的 ID。在 `CheckPermission` 中会查询 `FindByID(0)` 返回 "用户不存在"，导致 Check 接口返回错误但中间件 RequirePermission 的代码逻辑是 `err != nil || !allowed` → 返回 Forbidden。所以 API Key 实际上**无法通过权限检查**（除 `/auth/verify` 等不需要权限的端点）。

但这仍然是一个严重的设计缺陷——API Key 的行为不明确，且在 `/auth/verify` 和 `/auth/menu` 等端点上返回了错误数据。

**修复**:
```go
// 为 ServiceAccount 创建专属的伪用户或直接跳过 user_id 设置
if apiKey != "" {
    sa, err := saRepo.FindByApiKeyHash(hashApiKey(apiKey))
    if err != nil {
        response.ErrorWithMsg(c, errcode.Unauthorized, "无效的API Key")
        c.Abort()
        return
    }
    // 服务账号拥有全部权限（受信任的内部服务）
    c.Set("auth_type", "apikey")
    c.Set("service_account_id", sa.ID)
    c.Set("service_account_name", sa.Name)
    // 不设置 user_id，RequirePermission 检测到 apikey 类型直接放行
    c.Next()
    return
}
```

同时在 `RequirePermission` 中间件中：
```go
func RequirePermission(...) gin.HandlerFunc {
    return func(c *gin.Context) {
        // API Key 认证的直接放行 (受信任的内部服务)
        if authType, _ := c.Get("auth_type"); authType == "apikey" {
            c.Next()
            return
        }
        // ... 原有的 JWT 权限检查逻辑
    }
}
```

---

### CRIT-04: 无登录速率限制，易受暴力破解

**文件**: `internal/handler/auth_handler.go:33` | **风险**: 暴力破解

```go
func (h *AuthHandler) Login(c *gin.Context) {
    // 没有任何速率限制 — 可以无限次尝试密码
```

**影响**: 攻击者可无限尝试用户名/密码组合，配合字典攻击可在短时间内破解弱密码。

**修复**: 引入 Gin 速率限制中间件（推荐 `github.com/ulule/limiter` 或基于 Redis 的令牌桶）:

```go
import "github.com/ulule/limiter/v3"
import "github.com/ulule/limiter/v3/drivers/store/redis"

func SetupLoginRateLimiter(rdb *redis.Client) gin.HandlerFunc {
    rate := limiter.Rate{Period: 1 * time.Minute, Limit: 5} // 每分钟最多 5 次
    store, _ := redis.NewStore(rdb)
    instance := limiter.New(store, rate)

    return func(c *gin.Context) {
        ctx := c.Request.Context()
        limiterCtx, _ := instance.Get(ctx, c.ClientIP())
        if limiterCtx.Reached {
            response.ErrorWithMsg(c, errcode.InternalError, "请求过于频繁，请稍后再试")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

### CRIT-05: ParseUint 错误被丢弃 (_ 忽略)

**文件**: `internal/handler/user_handler.go:243-244` | **风险**: 参数注入

```go
func (h *UserHandler) RemoveRole(c *gin.Context) {
    id, _ := strconv.ParseUint(c.Param("id"), 10, 64)       // ← 错误被丢弃!
    roleID, _ := strconv.ParseUint(c.Param("roleId"), 10, 64) // ← 错误被丢弃!
```

**影响**: 如果 `id` 或 `roleId` 不是有效数字，`ParseUint` 返回 0。`RemoveRole(0, 0)` 会被传递到数据库层，可能导致意外的数据操作。

**修复**:
```go
func (h *UserHandler) RemoveRole(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        response.Error(c, errcode.InvalidParams)
        return
    }
    roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 64)
    if err != nil {
        response.Error(c, errcode.InvalidParams)
        return
    }
    // ...
}
```

---

### CRIT-06: initSuperAdmin 在无 TTY 环境挂起

**文件**: `cmd/server/main.go:263-272` | **风险**: 容器化部署失败

```go
func readPasswordFromTerminal() (string, error) {
    passwordBytes, err := term.ReadPassword(int(syscall.Stdin))  // ← 非 TTY 时挂起!
```

**影响**: Docker/K8s 部署时，stdin 不是终端，`term.ReadPassword` 会失败或挂起，导致服务无法启动。

**修复**:
```go
func initSuperAdmin(db *gorm.DB) error {
    // ... 检查 admin 是否存在 ...

    var password string
    var err error

    // 优先从环境变量读取 (容器友好)
    if envPass := os.Getenv("ADMIN_PASSWORD"); envPass != "" {
        password = envPass
    } else if term.IsTerminal(int(syscall.Stdin)) {
        password, err = readPasswordFromTerminal()
    } else {
        password, err = readPasswordFromStdin()
    }
    // ...
}
```

---

## 🟠 高危风险 (应尽快修复)

### HIGH-01: SQL LIKE 通配符未转义

**文件**: `internal/repository/user_repo.go:54` | **风险**: LIKE 注入

```go
query = query.Where("username LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
```

**影响**: 用户输入 `%` 或 `_` 会改变 LIKE 语义：
- 输入 `%` → 匹配所有用户
- 输入 `admin%` → 可探测其他用户

**修复**:
```go
func escapeLike(s string) string {
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, "%", "\\%")
    s = strings.ReplaceAll(s, "_", "\\_")
    return s
}

func (r *UserRepo) List(page, pageSize int, keyword string) ([]model.User, int64, error) {
    if keyword != "" {
        escaped := escapeLike(keyword)
        query = query.Where("username LIKE ? OR email LIKE ?",
            "%"+escaped+"%", "%"+escaped+"%")
    }
}
```
同样的问题存在于 `role_repo.go` 和 `permission_repo.go`。

---

### HIGH-02: 类型断言缺少安全检查

**文件**: `internal/handler/auth_handler.go:149` | **风险**: 运行时 panic

```go
allowed, err := h.permService.CheckPermission(userID.(uint), req.Resource, req.Action)
```

**影响**: 如果 `userID` 不是 `uint` 类型（例如中间件没有正确设置），`panic` 会导致整个请求 goroutine 崩溃（Gin 有 recovery 中间件，但会产生 500 错误）。

**修复**:
```go
uid, ok := userID.(uint)
if !ok {
    response.Error(c, errcode.Unauthorized)
    return
}
```

---

### HIGH-03: 通配符权限返回硬编码集合

**文件**: `internal/service/permission_check.go:57-62` | **风险**: 新增模块无法匹配

```go
if hasWildcard(permMap) {
    return map[string][]string{
        "user":       {"read", "create", "update", "delete"},
        "role":       {"read", "create", "update", "delete"},
        "permission": {"read", "create", "update", "delete"},
    }, nil
}
```

**影响**: 当有 `*:*` 通配符时，`GetUserPermissions` 返回硬编码的 3 个模块。新增的业务模块（如 `server`、`deployment`、`alert`）不会出现在返回结果中，导致前端无法正确显示菜单。

**修复**: 从数据库查询所有已注册的权限，动态构建:
```go
if hasWildcard(permMap) {
    allPerms, _ := s.permRepo.FindAll()  // 新增
    result := make(map[string][]string)
    for _, p := range allPerms {
        result[p.Resource] = appendIfMissing(result[p.Resource], p.Action)
    }
    return result, nil
}
```

---

### HIGH-04: 无请求 ID / 审计日志

**文件**: 全项目 | **风险**: 无法追踪安全事件

当前日志格式（`middleware/logger.go`）:
```
[200] POST /api/v1/auth/login | 12ms
```
缺少: 请求 ID、用户 ID、客户端 IP、请求体摘要。

**修复**: 在 Logger 中间件中注入 Request ID:
```go
func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := uuid.New().String()
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)

        start := time.Now()
        c.Next()

        userID, _ := c.Get("user_id")
        log.Printf("[%s] rid=%s uid=%v ip=%s %s %s | %d | %v",
            time.Now().Format(time.RFC3339),
            requestID, userID, c.ClientIP(),
            c.Request.Method, c.Request.URL.Path,
            c.Writer.Status(), time.Since(start))
    }
}
```

---

### HIGH-05: 无 Token 撤销机制

**文件**: `pkg/jwt/jwt.go` + `internal/handler/auth_handler.go:63` | **风险**: 登出后 Token 仍有效

```go
func (h *AuthHandler) Logout(c *gin.Context) {
    response.Success(c, nil)  // ← 无状态 JWT，没有做任何撤销
}
```

**影响**: 用户登出后，已签发的 Token 在过期前仍然有效。如果 Token 泄露，无法主动撤销。

**修复**: 增加 JWT 黑名单（Redis Set），登出时将 `jti` (JWT ID) 加入黑名单:
```go
// JWT Claims 增加 jti
type Claims struct {
    JTI       string `json:"jti"`      // JWT ID，用于撤销
    UserID    uint   `json:"user_id"`
    // ...
}

// 登出时
func (h *AuthHandler) Logout(c *gin.Context) {
    token := extractToken(c)
    claims, _ := jwtpkg.Parse(token)
    redis.SAdd(ctx, "jwt:blacklist", claims.JTI)
    redis.ExpireAt(ctx, "jwt:blacklist", claims.ExpiresAt.Time)
    response.Success(c, nil)
}

// 验证时检查
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
    claims, err := parseWithKey(tokenString, key)
    if err != nil { return nil, err }
    // 检查黑名单
    exists, _ := redis.SIsMember(ctx, "jwt:blacklist", claims.JTI).Result()
    if exists { return nil, errors.New("令牌已被撤销") }
    return claims, nil
}
```

---

## 🟡 中危风险 (建议修复)

### MED-01: 零测试覆盖率

**文件**: 全项目 | **风险**: 回归缺陷

项目无任何 `_test.go` 文件。RBAC 系统的鉴权逻辑必须通过测试保证正确性。

**建议**: 优先添加以下测试:
1. `pkg/jwt/jwt_test.go` — JWT 生成/解析/过期/刷新类型检查
2. `internal/service/permission_check_test.go` — 通配符匹配逻辑
3. `internal/middleware/auth_test.go` — 认证中间件
4. `internal/handler/auth_handler_test.go` — 登录/刷新/鉴权接口

---

### MED-02: bcrypt cost 使用默认值 10

**文件**: `internal/service/auth_service.go` (多处 `bcrypt.DefaultCost = 10`)

当前 bcrypt cost=10。在 2026 年的硬件上，建议使用 cost ≥ 12。

**修复**:
```go
const bcryptCost = 12  // 生产环境建议 12-14

bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
```

---

### MED-03: GORM AutoMigrate 替代版本化迁移

**文件**: `cmd/server/main.go:99-105` | **风险**: 生产环境表结构变更不可控

`AutoMigrate` 只增不减（不会删除列），且无法回滚。生产环境应使用版本化迁移。

**建议**: 引入 `golang-migrate/migrate` 或 `pressly/goose`:
```go
import "github.com/golang-migrate/migrate/v4"
m, _ := migrate.New("file://migrations", cfg.DB.DSN())
m.Up()
```

---

### MED-04: Swagger 在生产环境可访问

**文件**: `internal/router/router.go:37` | **风险**: API 结构泄露

```go
r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

Swagger UI 在所有模式下都可访问，暴露了完整的 API 结构。

**建议**:
```go
if cfg.Server.Mode != "release" {
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

---

### MED-05: 无 HTTP 请求体大小限制

**文件**: 全项目 | **风险**: 内存耗尽 (DoS)

Gin 默认无请求体大小限制，攻击者可发送超大 JSON 耗尽内存。

**修复**:
```go
r := gin.Default()
r.MaxMultipartMemory = 8 << 20  // 8 MB
// 或使用中间件
r.Use(func(c *gin.Context) {
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1 MB
    c.Next()
})
```

---

### MED-06: Server Mode 仍为 debug

**文件**: `config/config.yaml:2` | **风险**: 堆栈信息泄露

```yaml
server:
  mode: debug   # ← 生产环境应改为 release
```

debug 模式会在 Panic 时返回完整的 goroutine 堆栈和源码行号。

**修复**: 生产环境改为 `release`；条件编译时检查环境变量:
```go
if os.Getenv("GIN_MODE") == "" {
    gin.SetMode(cfg.Server.Mode)  // 从配置读取
}
```

---

## 🔵 代码优化建议

### OPT-01: 硬编码错误字符串应改为哨兵错误

**文件**: `internal/service/auth_service.go:31,41,45,53`

```go
return "", nil, errors.New("用户名或密码错误")  // 硬编码字符串
return "", nil, errors.New("用户已被禁用")
```

**建议**: 使用哨兵错误变量，方便调用方比较:
```go
var (
    ErrInvalidCredentials = errors.New("用户名或密码错误")
    ErrUserDisabled       = errors.New("用户已被禁用")
)
```

---

### OPT-02: 分页参数应在 Handler 层统一处理

**文件**: 多个 handler + service

当前 page/pageSize 的默认值在 Service 层处理，应在 Handler 层完成:
```go
func normalizePage(page, pageSize int) (int, int) {
    if page <= 0 { page = 1 }
    if pageSize <= 0 || pageSize > 100 { pageSize = 10 }
    return page, pageSize
}
```

---

### OPT-03: remove 文件残留的注释死代码

**文件**: `internal/service/permission_check.go:110-111`

```go
// 需要 import model 在函数里用了 model.Role
// 实际该文件 import 里没有 model，需要加上
```

这是一条过时的开发注释（import 已添加），应删除。

---

### OPT-04: 数据库连接池未显式配置

**文件**: `cmd/server/main.go`

当前依赖 GORM 默认连接池（MaxOpenConns=0 无限制）。在高并发下可能导致 MySQL 连接耗尽。

**修复**:
```go
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

---

## ✅ 安全实践 (已有的良好实践)

| 实践 | 位置 | 说明 |
|------|------|------|
| 密码 bcrypt 哈希 | `auth_service.go` | 使用 bcrypt 存储密码，`json:"-"` 防止泄露 |
| 统一错误响应 | `pkg/response/` | 所有错误码统一管理 |
| 软删除 | `model/base.go` | `gorm.DeletedAt` 防止误删 |
| JWT TokenType 校验 | `pkg/jwt/jwt.go` | Refresh Token 只能用于 refresh 端点 |
| 登录信息隐藏 | `auth_service.go:28-45` | 错误消息统一为"用户名或密码错误" |
| CORS 可配置 | `middleware/cors.go` | 支持动态 Origin 匹配 |
| 优雅关闭 | `cmd/server/main.go:163` | SIGTERM 30s 超时处理 |
| Redis 降级 | `cmd/server/main.go:91-96` | Redis 不可用时自动降级 |
| SHA256 存储 API Key | `service_account_service.go` | 数据库只存哈希 |
| API Key 仅显示一次 | `service_account_handler.go:36` | 创建后无法再次获取明文 |

---

## 修复优先级路线图

| 阶段 | 编号 | 问题 | 工作量 |
|------|------|------|--------|
| **Week 1** | CRIT-01 | 敏感信息移除 + 环境变量 | 1h |
| | CRIT-04 | 登录速率限制 | 2h |
| | CRIT-05 | 修复 `_` 错误丢弃 | 30min |
| | CRIT-06 | 非 TTY 环境密码读取 | 1h |
| | MED-06 | 生产环境改为 release | 5min |
| **Week 2** | CRIT-02 | JWT 密钥管理重构 | 3h |
| | CRIT-03 | API Key 权限模型修复 | 2h |
| | HIGH-01 | LIKE 通配符转义 | 30min |
| | HIGH-02 | 类型断言安全检查 | 1h |
| **Week 3** | HIGH-04 | 结构化日志 + 请求 ID | 2h |
| | HIGH-05 | JWT 黑名单 / 撤销 | 3h |
| | OPT-04 | 数据库连接池配置 | 15min |
| **Week 4** | HIGH-03 | 通配符权限动态化 | 2h |
| | MED-01 | 核心模块单元测试 | 1d |
| | MED-02 | bcrypt cost 提升 | 5min |
| | OPT-03 | 删除死代码注释 | 1min |

---

> 📄 **审计引擎**: RBAC Audit Skill v1.0 | **审计范围**: Go 源码 41 文件 | **发现**: 6 严重 / 5 高危 / 6 中危 / 4 优化
