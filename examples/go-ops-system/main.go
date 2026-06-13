// 运维管理系统 — Go net/http 实现
//
// 完整演示如何将 api-rbac 作为独立的权限管理微服务与 Go 业务系统集成。
// 使用项目内置 SDK (pkg/client), 支持韧性降级。
//
// 运行: go run . (确保 api-rbac 已启动在 :8087)

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"api-rbac/pkg/client"
)

// ================================================================
// 配置
// ================================================================

const rbacURL = "http://localhost:8087/api/v1"

var rbacClient = client.NewRBACClient(rbacURL)

// ================================================================
// 权限校验中间件 (韧性模式)
// ================================================================

func requirePermission(resource, action string) func(http.Handler) http.Handler {
	failCount := 0
	circuitOpen := false
	circuitOpened := time.Time{}
	cache := map[string]struct {
		perms map[string][]string
		ts    time.Time
	}{}
	var mu sync.Mutex

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeJSON(w, 401, 1002, "未提供认证Token")
				return
			}

			// 熔断检查
			mu.Lock()
			if circuitOpen {
				if time.Since(circuitOpened) > 30*time.Second {
					circuitOpen = false
					failCount = 0
					mu.Unlock()
				} else {
					mu.Unlock()
					// 走本地缓存
					if checkFromCache(cache, token, resource, action) {
						next.ServeHTTP(w, r)
						return
					}
					writeJSON(w, 403, 1003, "权限服务不可用(熔断), 缓存中无权限数据")
					return
				}
			} else {
				mu.Unlock()
			}

			// 远程校验
			resp, err := rbacClient.CheckPermission(token, resource, action)
			if err == nil && resp.Code == 0 {
				// 成功 → 异步加载权限到缓存
				mu.Lock()
				failCount = 0
				mu.Unlock()
				go populateCache(cache, token)
				if resp.Data.Allowed {
					next.ServeHTTP(w, r)
					return
				}
				writeJSON(w, 403, 1003, "无权限: "+resource+":"+action)
				return
			}

			// 失败 → 记录 + 走缓存
			mu.Lock()
			failCount++
			if failCount >= 5 {
				circuitOpen = true
				circuitOpened = time.Now()
				log.Printf("⚠️ RBAC 熔断触发 (连续 %d 次失败)", failCount)
			}
			mu.Unlock()

			if checkFromCache(cache, token, resource, action) {
				next.ServeHTTP(w, r)
				return
			}
			writeJSON(w, 502, 1005, "权限服务不可用, 缓存中无权限数据")
		})
	}
}

func extractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

func checkFromCache(cache map[string]struct {
	perms map[string][]string
	ts    time.Time
}, token, resource, action string) bool {
	entry, ok := cache[token]
	if !ok || time.Since(entry.ts) > 5*time.Minute {
		return false
	}
	// 通配符匹配
	if actions, ok := entry.perms["*"]; ok {
		for _, a := range actions {
			if a == "*" || a == action {
				return true
			}
		}
	}
	if actions, ok := entry.perms[resource]; ok {
		for _, a := range actions {
			if a == "*" || a == action {
				return true
			}
		}
	}
	return false
}

func populateCache(cache map[string]struct {
	perms map[string][]string
	ts    time.Time
}, token string) {
	resp, err := rbacClient.GetMenu(token)
	if err != nil || resp.Code != 0 {
		return
	}
	var mu sync.Mutex // local lock just for this cache map
	mu.Lock()
	cache[token] = struct {
		perms map[string][]string
		ts    time.Time
	}{resp.Data.Permissions, time.Now()}
	mu.Unlock()
}

// ================================================================
// 业务模型
// ================================================================

type Server struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Status string `json:"status"`
}

type Deployment struct {
	ID      int    `json:"id"`
	Project string `json:"project"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type Alert struct {
	ID      int    `json:"id"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Acked   bool   `json:"acked"`
}

// 模拟数据库
var (
	servers     = []Server{{1, "web-01", "10.0.1.10", "running"}, {2, "web-02", "10.0.1.11", "stopped"}, {3, "db-01", "10.0.2.10", "running"}}
	deployments = []Deployment{{1, "web-app", "v2.3.1", "success"}, {2, "api-service", "v1.5.0", "failed"}}
	alerts      = []Alert{{1, "critical", "CPU 使用率 95%", false}, {2, "warning", "磁盘使用率 80%", true}}
	nextSrvID   atomic.Int64
	nextDepID   atomic.Int64
)

func init() {
	nextSrvID.Store(3)
	nextDepID.Store(2)
}

// ================================================================
// HTTP Handler — 登录 (转发 RBAC)
// ================================================================

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, 1001, "Method Not Allowed")
		return
	}
	var body struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	resp, err := rbacClient.Login(body.Account, body.Password)
	if err != nil || resp.Code != 0 {
		writeJSON(w, 401, 1009, "用户名或密码错误")
		return
	}
	writeJSON(w, 200, 0, resp.Data)
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	resp, err := rbacClient.Refresh(body.RefreshToken)
	if err != nil || resp.Code != 0 {
		writeJSON(w, 401, 1007, "刷新令牌无效")
		return
	}
	writeJSON(w, 200, 0, resp.Data)
}

// ================================================================
// HTTP Handler — 服务器管理
// ================================================================

func handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, 0, servers)
	default:
		writeJSON(w, 405, 1001, "Method Not Allowed")
	}
}

func handleServerRestart(w http.ResponseWriter, r *http.Request) {
	// ... 解析 ID + 重启逻辑 (简化)
	writeJSON(w, 200, 0, map[string]string{"message": "服务器重启成功"})
}

func handleServerStop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, 0, map[string]string{"message": "服务器已停止"})
}

// ================================================================
// HTTP Handler — 发布管理
// ================================================================

func handleDeployments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, 0, deployments)
	case http.MethodPost:
		var body struct {
			Project string `json:"project"`
			Version string `json:"version"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		id := int(nextDepID.Add(1))
		d := Deployment{id, body.Project, body.Version, "success"}
		deployments = append(deployments, d)
		writeJSON(w, 200, 0, d)
	default:
		writeJSON(w, 405, 1001, "Method Not Allowed")
	}
}

func handleDeploymentRollback(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, 0, map[string]string{"message": "发布已回滚"})
}

// ================================================================
// HTTP Handler — 告警管理
// ================================================================

func handleAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, 0, alerts)
}

func handleAlertAck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, 0, map[string]string{"message": "告警已确认"})
}

// ================================================================
// 工具函数
// ================================================================

func writeJSON(w http.ResponseWriter, httpStatus int, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": getMsg(code),
		"data":    data,
	})
}

func getMsg(code int) string {
	if code == 0 {
		return "success"
	}
	msgs := map[int]string{
		401: "未授权", 403: "无权限", 404: "资源不存在",
		405: "方法不允许", 502: "服务不可用", 1001: "参数错误",
		1002: "未授权", 1003: "无权限", 1005: "服务内部错误",
		1007: "Token过期", 1008: "Token无效", 1009: "密码错误",
	}
	if m, ok := msgs[code]; ok {
		return m
	}
	return "未知错误"
}

// ================================================================
// 路由注册
// ================================================================

func main() {
	mux := http.NewServeMux()

	// 无需认证
	mux.HandleFunc("/api/auth/login", handleLogin)
	mux.HandleFunc("/api/auth/refresh", handleRefresh)

	// 服务器管理
	mux.Handle("/api/servers",
		requirePermission("server", "read")(http.HandlerFunc(handleServers)))
	mux.Handle("/api/servers/restart",
		requirePermission("server", "restart")(http.HandlerFunc(handleServerRestart)))
	mux.Handle("/api/servers/stop",
		requirePermission("server", "stop")(http.HandlerFunc(handleServerStop)))

	// 发布管理
	mux.Handle("/api/deployments",
		requirePermission("deployment", "read")(http.HandlerFunc(handleDeployments)))
	mux.Handle("/api/deployments/execute",
		requirePermission("deployment", "execute")(http.HandlerFunc(handleDeployments)))
	mux.Handle("/api/deployments/rollback",
		requirePermission("deployment", "rollback")(http.HandlerFunc(handleDeploymentRollback)))

	// 告警管理
	mux.Handle("/api/alerts",
		requirePermission("alert", "read")(http.HandlerFunc(handleAlerts)))
	mux.Handle("/api/alerts/ack",
		requirePermission("alert", "ack")(http.HandlerFunc(handleAlertAck)))

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, 0, map[string]string{"status": "ok"})
	})

	log.Println("========================================")
	log.Println("  运维管理系统 (Go net/http)")
	log.Printf("  RBAC 服务: %s", rbacURL)
	log.Println("========================================")
	log.Println("  端点:")
	log.Println("    POST /api/auth/login             — 登录")
	log.Println("    GET  /api/servers                 — 服务器列表 (server:read)")
	log.Println("    POST /api/servers/restart         — 重启服务器 (server:restart)")
	log.Println("    POST /api/servers/stop            — 停止服务器 (server:stop)")
	log.Println("    POST /api/deployments/execute     — 执行发布 (deployment:execute)")
	log.Println("    POST /api/deployments/rollback    — 回滚发布 (deployment:rollback)")
	log.Println("    GET  /api/alerts                  — 告警列表 (alert:read)")
	log.Println("    POST /api/alerts/ack              — 确认告警 (alert:ack)")
	log.Println("========================================")

	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
