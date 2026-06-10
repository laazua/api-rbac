package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"api-rbac/config"
	"api-rbac/internal/handler"
	"api-rbac/internal/middleware"
	"api-rbac/internal/service"
)

func Setup(
	authH *handler.AuthHandler,
	userH *handler.UserHandler,
	roleH *handler.RoleHandler,
	permH *handler.PermissionHandler,
	cfg *config.Config,
	permCheckSvc *service.PermissionCheckService,
) *gin.Engine {
	r := gin.Default()

	// 全局中间件
	r.Use(middleware.CORS(cfg.CORS))
	r.Use(middleware.Logger())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		// 认证接口 - 无需 JWT
		auth := api.Group("/auth")
		{
			auth.POST("/login", authH.Login)
		}

		// 以下接口需要认证
		authed := api.Group("")
		authed.Use(middleware.AuthRequired())
		{
			// 登出 & Token验证 & 权限检查 & 菜单
			authed.POST("/auth/logout", authH.Logout)
			authed.POST("/auth/verify", authH.Verify)
			authed.POST("/auth/check", authH.Check)
			authed.GET("/auth/menu", authH.Menu)

			// ---- 用户管理 ----
			users := authed.Group("/users")
			users.Use(middleware.RequirePermission(permCheckSvc, "user", "read"))
			{
				users.GET("", userH.List)
				users.GET("/:id", userH.GetByID)
			}
			// 用户写操作需要额外权限
			usersWrite := authed.Group("/users")
			{
				usersWrite.POST("", middleware.RequirePermission(permCheckSvc, "user", "create"), userH.Create)
				usersWrite.PUT("/:id", middleware.RequirePermission(permCheckSvc, "user", "update"), userH.Update)
				usersWrite.DELETE("/:id", middleware.RequirePermission(permCheckSvc, "user", "delete"), userH.Delete)
				usersWrite.PUT("/:id/password", middleware.RequirePermission(permCheckSvc, "user", "update"), userH.ChangePassword)
				usersWrite.POST("/:id/roles", middleware.RequirePermission(permCheckSvc, "user", "update"), userH.AssignRoles)
				usersWrite.DELETE("/:id/roles/:roleId", middleware.RequirePermission(permCheckSvc, "user", "update"), userH.RemoveRole)
			}

			// ---- 角色管理 ----
			roles := authed.Group("/roles")
			roles.Use(middleware.RequirePermission(permCheckSvc, "role", "read"))
			{
				roles.GET("", roleH.List)
				roles.GET("/:id", roleH.GetByID)
			}
			rolesWrite := authed.Group("/roles")
			{
				rolesWrite.POST("", middleware.RequirePermission(permCheckSvc, "role", "create"), roleH.Create)
				rolesWrite.PUT("/:id", middleware.RequirePermission(permCheckSvc, "role", "update"), roleH.Update)
				rolesWrite.DELETE("/:id", middleware.RequirePermission(permCheckSvc, "role", "delete"), roleH.Delete)
				rolesWrite.POST("/:id/permissions", middleware.RequirePermission(permCheckSvc, "role", "update"), roleH.AssignPermissions)
				rolesWrite.DELETE("/:id/permissions/:permId", middleware.RequirePermission(permCheckSvc, "role", "update"), roleH.RemovePermission)
			}

			// ---- 权限管理 ----
			perms := authed.Group("/permissions")
			perms.Use(middleware.RequirePermission(permCheckSvc, "permission", "read"))
			{
				perms.GET("", permH.List)
				perms.GET("/:id", permH.GetByID)
			}
			permsWrite := authed.Group("/permissions")
			{
				permsWrite.POST("", middleware.RequirePermission(permCheckSvc, "permission", "create"), permH.Create)
				permsWrite.PUT("/:id", middleware.RequirePermission(permCheckSvc, "permission", "update"), permH.Update)
				permsWrite.DELETE("/:id", middleware.RequirePermission(permCheckSvc, "permission", "delete"), permH.Delete)
			}
		}
	}

	return r
}
