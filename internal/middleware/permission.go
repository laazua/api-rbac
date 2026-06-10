package middleware

import (
	"github.com/gin-gonic/gin"

	"api-rbac/internal/service"
	"api-rbac/pkg/errcode"
	"api-rbac/pkg/response"
)

// RequirePermission 返回一个 Gin 中间件，校验当前用户是否拥有指定资源的指定操作权限
// 用法: router.Use(middleware.RequirePermission(permCheckService, "user", "read"))
func RequirePermission(permCheckService *service.PermissionCheckService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, errcode.Unauthorized)
			c.Abort()
			return
		}

		allowed, err := permCheckService.CheckPermission(userID.(uint), resource, action)
		if err != nil || !allowed {
			response.Error(c, errcode.Forbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
