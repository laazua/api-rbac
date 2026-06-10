package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	jwtpkg "api-rbac/pkg/jwt"
	"api-rbac/pkg/response"
	"api-rbac/pkg/errcode"
)

// AuthRequired JWT 认证中间件，排除登录接口
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, errcode.Unauthorized)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, errcode.TokenInvalid)
			c.Abort()
			return
		}

		claims, err := jwtpkg.Parse(parts[1])
		if err != nil {
			if err == jwtpkg.ErrTokenExpired {
				response.Error(c, errcode.TokenExpired)
			} else {
				response.Error(c, errcode.TokenInvalid)
			}
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
