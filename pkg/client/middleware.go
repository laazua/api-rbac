package client

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PermissionGuard 返回一个 Gin 中间件，用于校验请求是否有指定权限
// 用法示例:
//
//	r := gin.Default()
//	r.Use(client.PermissionGuard(client.NewRBACClient("http://localhost:8087/api/v1"), "user", "delete"))
func PermissionGuard(rbacClient *RBACClient, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 1002, "message": "未提供认证Token"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 1008, "message": "Token格式无效"})
			c.Abort()
			return
		}

		token := parts[1]

		resp, err := rbacClient.CheckPermission(token, resource, action)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 1005, "message": "权限检查服务不可用"})
			c.Abort()
			return
		}

		if resp.Code != 0 || !resp.Data.Allowed {
			c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "无权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}
