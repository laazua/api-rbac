// 运维管理系统 — Go + Gin 实现
//
// 完整演示如何将 api-rbac 作为独立的权限管理微服务与 Go 业务系统集成。
// 使用项目内置 SDK (pkg/client) 的 ResilientGuard 中间件实现韧性权限校验。
//
// 运行: go run . (确保 api-rbac 已启动在 :8087)

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-ops-system/internal/handler"
	"go-ops-system/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/laazua/api-rbac/pkg/client"
)

const (
	rbacURL     = "http://localhost:8087/api/v1"
	opsPort     = ":8083"
	cacheTTLSec = 300 // 权限缓存 5 分钟
)

var rbacClient = client.NewRBACClient(rbacURL)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS — 允许前端开发模式跨域
	r.Use(corsMiddleware())

	// ================================================================
	// 初始化 Handler
	// ================================================================
	authH := handler.NewAuthHandler(rbacClient)
	serverH := handler.NewServerHandler()
	deployH := handler.NewDeploymentHandler()
	alertH := handler.NewAlertHandler()

	// ================================================================
	// 公开路由 (无需认证)
	// ================================================================
	r.POST("/api/auth/login", authH.Login)
	r.POST("/api/auth/refresh", authH.Refresh)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ================================================================
	// 认证路由 (需要 Token 验证, 不检查具体权限)
	// ================================================================
	auth := r.Group("/api")
	auth.Use(middleware.ExtractUserInfo(rbacClient))
	{
		// 获取用户全部权限 (供前端菜单/按钮控制)
		auth.GET("/auth/permissions", authH.GetPermissions)
	}

	// ================================================================
	// 业务路由 (需要 Token + 具体权限)
	//
	// 使用 ResilientGuard 中间件实现:
	//   - 远程调用 api-rbac 校验权限
	//   - 5 次连续失败后自动熔断 30 秒
	//   - 熔断期间走本地缓存 (5 分钟 TTL)
	//   - FailModeCache: RBAC 宕机时降级使用缓存
	// ================================================================

	// --- 服务器管理 ---
	serverGroup := r.Group("/api/servers")
	serverGroup.Use(middleware.ExtractUserInfo(rbacClient))
	{
		// GET /api/servers → server:read
		serverGroup.GET("",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "server", "read"),
			serverH.List)
		// POST /api/servers → server:create
		serverGroup.POST("",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "server", "create"),
			serverH.Create)
		// DELETE /api/servers/:id → server:delete
		serverGroup.DELETE("/:id",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "server", "delete"),
			serverH.Delete)
	}

	// 服务器操作 (独立路由, 避免与 :id 冲突)
	serverOps := r.Group("/api/servers")
	serverOps.Use(middleware.ExtractUserInfo(rbacClient))
	{
		// POST /api/servers/restart → server:restart
		serverOps.POST("/restart",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "server", "restart"),
			serverH.Restart)
		// POST /api/servers/stop → server:stop
		serverOps.POST("/stop",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "server", "stop"),
			serverH.Stop)
	}

	// --- 发布管理 ---
	deployGroup := r.Group("/api/deployments")
	deployGroup.Use(middleware.ExtractUserInfo(rbacClient))
	{
		// GET /api/deployments → deployment:read
		deployGroup.GET("",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "deployment", "read"),
			deployH.List)
		// GET /api/deployments/:id → deployment:read
		deployGroup.GET("/:id",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "deployment", "read"),
			deployH.GetByID)
		// POST /api/deployments → deployment:execute
		deployGroup.POST("",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "deployment", "execute"),
			deployH.Execute)
		// POST /api/deployments/rollback → deployment:rollback
		deployGroup.POST("/rollback",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "deployment", "rollback"),
			deployH.Rollback)
	}

	// --- 告警管理 ---
	alertGroup := r.Group("/api/alerts")
	alertGroup.Use(middleware.ExtractUserInfo(rbacClient))
	{
		// GET /api/alerts → alert:read
		alertGroup.GET("",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "alert", "read"),
			alertH.List)
		// POST /api/alerts/ack → alert:ack
		alertGroup.POST("/ack",
			client.ResilientGuard(rbacClient, client.FailModeCache, cacheTTLSec, "alert", "ack"),
			alertH.Ack)
	}

	// ================================================================
	// 静态文件服务 — 生产模式: 直接 serve 前端构建产物
	// 开发模式: 前端用 Vite dev server (:5173), 不需要此功能
	// ================================================================
	distPath := "./web/dist"
	if _, err := os.Stat(distPath); err == nil {
		r.NoRoute(func(c *gin.Context) {
			// SPA fallback: 非 API 路由返回 index.html (支持前端 hash router)
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"code": 1004, "message": "接口不存在"})
				return
			}
			c.File(distPath + "/index.html")
		})
		r.Static("/assets", distPath+"/assets")
		r.StaticFile("/", distPath+"/index.html")
		log.Println("  📦 已启用静态文件服务: web/dist/")
	} else {
		log.Println("  ℹ️  未找到 web/dist/, 仅提供 API 服务 (前端请用 Vite dev server)")
	}

	// ================================================================
	// 优雅关闭
	// ================================================================
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("正在关闭运维管理系统...")
		os.Exit(0)
	}()

	// ================================================================
	// 启动
	// ================================================================
	log.Println("========================================")
	log.Println("  运维管理系统 (Go + Gin)")
	log.Printf("  RBAC 服务: %s", rbacURL)
	log.Println("========================================")
	log.Println("  业务端点:")
	log.Println("    POST /api/auth/login              — 登录")
	log.Println("    POST /api/auth/refresh            — 刷新 Token")
	log.Println("    GET  /api/auth/permissions        — 获取用户权限")
	log.Println("    GET  /api/servers                 — 服务器列表 [server:read]")
	log.Println("    POST /api/servers                 — 创建服务器 [server:create]")
	log.Println("    DELETE /api/servers/:id           — 删除服务器 [server:delete]")
	log.Println("    POST /api/servers/restart         — 重启服务器 [server:restart]")
	log.Println("    POST /api/servers/stop            — 停止服务器 [server:stop]")
	log.Println("    GET  /api/deployments             — 发布列表 [deployment:read]")
	log.Println("    POST /api/deployments             — 执行发布 [deployment:execute]")
	log.Println("    POST /api/deployments/rollback    — 回滚发布 [deployment:rollback]")
	log.Println("    GET  /api/alerts                  — 告警列表 [alert:read]")
	log.Println("    POST /api/alerts/ack              — 确认告警 [alert:ack]")
	log.Println("    GET  /health                      — 健康检查")
	log.Println("========================================")

	log.Printf("🚀 运维管理系统启动于 http://0.0.0.0%s", opsPort)
	if err := r.Run(opsPort); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}

// corsMiddleware CORS 中间件 — 允许前端开发模式跨域
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
