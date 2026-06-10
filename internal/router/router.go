package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"api-rbac/config"
	"api-rbac/internal/handler"
	"api-rbac/internal/middleware"
)

func Setup(
	authH *handler.AuthHandler,
	userH *handler.UserHandler,
	roleH *handler.RoleHandler,
	permH *handler.PermissionHandler,
	cfg *config.Config,
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
			// 登出
			authed.POST("/auth/logout", authH.Logout)
			// Token验证与权限检查 (业务系统集成接口)
			authed.POST("/auth/verify", authH.Verify)
			authed.POST("/auth/check", authH.Check)

			// 用户管理
			users := authed.Group("/users")
			{
				users.GET("", userH.List)
				users.POST("", userH.Create)
				users.GET("/:id", userH.GetByID)
				users.PUT("/:id", userH.Update)
				users.DELETE("/:id", userH.Delete)
				users.PUT("/:id/password", userH.ChangePassword)
				users.POST("/:id/roles", userH.AssignRoles)
				users.DELETE("/:id/roles/:roleId", userH.RemoveRole)
			}

			// 角色管理
			roles := authed.Group("/roles")
			{
				roles.GET("", roleH.List)
				roles.POST("", roleH.Create)
				roles.GET("/:id", roleH.GetByID)
				roles.PUT("/:id", roleH.Update)
				roles.DELETE("/:id", roleH.Delete)
				roles.POST("/:id/permissions", roleH.AssignPermissions)
				roles.DELETE("/:id/permissions/:permId", roleH.RemovePermission)
			}

			// 权限管理
			permissions := authed.Group("/permissions")
			{
				permissions.GET("", permH.List)
				permissions.POST("", permH.Create)
				permissions.GET("/:id", permH.GetByID)
				permissions.PUT("/:id", permH.Update)
				permissions.DELETE("/:id", permH.Delete)
			}
		}
	}

	return r
}
