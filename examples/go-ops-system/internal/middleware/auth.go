// 运维系统 — 认证中间件
// 从请求头提取 JWT Token 并验证, 将用户信息注入 Gin Context
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/laazua/api-rbac/pkg/client"
	"github.com/laazua/api-rbac/pkg/errcode"
	"github.com/laazua/api-rbac/pkg/response"
)

const (
	// ContextKeyUserID Token 中解析出的用户 ID
	ContextKeyUserID = "ops_user_id"
	// ContextKeyUsername Token 中解析出的用户名
	ContextKeyUsername = "ops_username"
	// ContextKeyToken 原始 Token 字符串
	ContextKeyToken = "ops_token"
)

// ExtractUserInfo 提取并验证 JWT Token, 将用户信息写入 Context
// 注意: 不检查权限, 仅验证身份 — 权限由后续的 ResilientGuard 检查
func ExtractUserInfo(rbacClient *client.RBACClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			response.Error(c, errcode.Unauthorized)
			c.Abort()
			return
		}

		// 调用 RBAC 验证 Token 有效性
		verifyResp, err := rbacClient.Verify(token)
		if err != nil || verifyResp.Code != 0 {
			response.Error(c, errcode.Unauthorized)
			c.Abort()
			return
		}

		// 注入用户信息到 Context 供后续 Handler 使用
		c.Set(ContextKeyUserID, verifyResp.Data.UserID)
		c.Set(ContextKeyUsername, verifyResp.Data.Username)
		c.Set(ContextKeyToken, token)

		c.Next()
	}
}

// extractBearerToken 从 Authorization 头部提取 Bearer Token
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// GetUserID 从 Context 中读取用户 ID
func GetUserID(c *gin.Context) uint {
	if v, ok := c.Get(ContextKeyUserID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从 Context 中读取用户名
func GetUsername(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyUsername); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetToken 从 Context 中读取原始 Token
func GetToken(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyToken); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// respondUnauthorized 返回 401 未授权响应
func respondUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":    errcode.Unauthorized,
		"message": "未授权, 请先登录",
		"data":    nil,
	})
}
