package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/laazua/api-rbac/internal/repository"
	jwtpkg "github.com/laazua/api-rbac/pkg/jwt"
	"github.com/laazua/api-rbac/pkg/response"
	"github.com/laazua/api-rbac/pkg/errcode"
)

// AuthRequired 返回 JWT + API Key 双认证中间件
// 支持两种认证方式:
// 1. X-API-Key 头部 — 服务间调用
// 2. Authorization: Bearer <token> — 用户请求
func AuthRequired(saRepo *repository.ServiceAccountRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先检查 X-API-Key（服务间调用）
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			sa, err := saRepo.FindByApiKeyHash(hashApiKey(apiKey))
			if err != nil {
				response.ErrorWithMsg(c, errcode.Unauthorized, "无效的API Key")
				c.Abort()
				return
			}
			// API Key 认证: 仅标记认证方式，不设置 user_id
			// RequirePermission 中间件检测到 apikey 类型直接放行
			c.Set("auth_type", "apikey")
			c.Set("service_account_id", sa.ID)
			c.Set("service_account_name", sa.Name)
			c.Next()
			return
		}

		// Bearer JWT 认证
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

		c.Set("auth_type", "jwt")
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// hashApiKey 对 API Key 做 SHA256 哈希，与服务账号表中存储的哈希比对
func hashApiKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
