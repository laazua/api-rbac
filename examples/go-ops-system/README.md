# Go 运维管理系统 — api-rbac 解耦集成示例

> 演示 Go 业务系统如何使用 `pkg/client` SDK 与 api-rbac 集成，实现权限与业务完全解耦。

## 项目结构

```
go-ops-system/
├── go.mod          # 独立模块, replace → ../../ (引用 api-rbac SDK)
├── main.go         # 完整业务系统 (~280 行)
└── README.md
```

## 设计亮点

- **零外部 Web 框架** — 仅使用 Go 标准库 `net/http`
- **内嵌韧性层** — 熔断器 + 本地缓存，RBAC 宕机时自动降级
- **一行中间件完成鉴权** — `requirePermission("server", "restart")`

## 运行

```bash
# 1. 启动 api-rbac (:8087)
cd /opt/codes/api-rbac && go run ./cmd/server

# 2. 初始化权限 (使用 Python 示例的脚本)
cd examples/go-ops-system
bash ../python-ops-system/setup_permissions.sh

# 3. 启动 Go 运维系统 (:8081)
go run .
```

## 测试

```bash
# 登录
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"account":"admin","password":"your_password"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

# 查看服务器 (server:read)
curl http://localhost:8081/api/servers -H "Authorization: Bearer $TOKEN"

# 重启服务器 (server:restart)
curl -X POST http://localhost:8081/api/servers/restart -H "Authorization: Bearer $TOKEN"

# 无权限操作 → 403
curl -X POST http://localhost:8081/api/servers/stop -H "Authorization: Bearer $TOKEN"
```

## 核心代码

### 权限中间件 (~80 行)

```go
func requirePermission(resource, action string) func(http.Handler) http.Handler {
    // 内嵌: 远程校验 → 熔断器 → 本地缓存 → 安全拒绝
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r)
            // 1. 尝试远程校验
            resp, err := rbacClient.CheckPermission(token, resource, action)
            if err == nil && resp.Code == 0 {
                go populateCache(token) // 异步更新本地缓存
                if resp.Data.Allowed { next.ServeHTTP(w, r); return }
                writeJSON(w, 403, "无权限"); return
            }
            // 2. 失败 → 更新熔断计数器
            failCount++; if failCount >= 5 { circuitOpen = true }
            // 3. 走本地缓存
            if checkFromCache(token, resource, action) { next.ServeHTTP(w, r); return }
            // 4. 缓存也无 → 安全拒绝
            writeJSON(w, 502, "权限服务不可用")
        })
    }
}
```

### 路由注册

```go
mux := http.NewServeMux()

// 无需认证
mux.HandleFunc("/api/auth/login", handleLogin)

// 业务接口 — 中间件自动鉴权
mux.Handle("/api/servers",
    requirePermission("server", "read")(http.HandlerFunc(handleServers)))
mux.Handle("/api/servers/restart",
    requirePermission("server", "restart")(http.HandlerFunc(handleServerRestart)))
mux.Handle("/api/deployments/execute",
    requirePermission("deployment", "execute")(http.HandlerFunc(handleDeployments)))
```

## 与 Python/Java 示例对比

| | Python | Java | **Go** |
|---|---|---|---|
| Web 框架 | Flask | Spring Boot | **net/http (标准库)** |
| SDK | `sdk/python/` | `sdk/java17/` | **`pkg/client/` (内嵌)** |
| 鉴权方式 | `@require_permission` 装饰器 | `@RequirePermission` 注解 | **`requirePermission()` 中间件** |
| 韧性层 | ✅ FailMode.CACHE | ✅ FailMode.CACHE | **✅ 内嵌熔断+缓存** |
| 端口 | :5000 | :8080 | **:8081** |
