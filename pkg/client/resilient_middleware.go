package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// FailMode 定义 RBAC 不可达时的故障模式
type FailMode int

const (
	FailModeDeny  FailMode = iota // 拒绝所有请求 (默认, 安全)
	FailModeCache                 // 使用本地缓存 (推荐, 高可用)
)

// 全局共享的缓存和熔断器状态 (所有 ResilientGuard 共用)
var (
	resilientCacheStore   = make(map[string]cacheEntry)
	resilientMu           sync.Mutex
	resilientFailCount    int
	resilientCircuitOpen  bool
	resilientCircuitSince time.Time
)

// ResilientGuard 返回一个带韧性能力的 Gin 权限校验中间件。
//
// 所有 ResilientGuard 实例共享同一个本地缓存和熔断器。
// 特性:
//   - RBAC 正常 → 远程校验 + 异步更新缓存
//   - RBAC 不可达 → 根据 failMode: DENY=拒绝 / CACHE=查本地缓存
//   - 熔断保护: 连续失败 5 次后走缓存, 避免超时堆积
//   - 自动恢复: 熔断 30s 后探测, 正常则自动切回
//
// 用法:
//
//	r.Use(ResilientGuard(rbacClient, FailModeCache, 300, "user", "delete"))
func ResilientGuard(client *RBACClient, failMode FailMode, cacheTTLSec int, resource, action string) gin.HandlerFunc {
	cacheTTL := time.Duration(cacheTTLSec) * time.Second

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

		// 熔断状态下直接走缓存
		if isGlobalCircuitOpen() {
			if failMode == FailModeDeny {
				c.JSON(http.StatusBadGateway, gin.H{"code": 1005, "message": "权限服务不可用(已熔断)"})
				c.Abort()
				return
			}
			if globalCheckFromCache(token, resource, action, cacheTTL) {
				c.Next()
				return
			}
			c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "无权限"})
			c.Abort()
			return
		}

		resp, err := client.CheckPermission(token, resource, action)
		if err == nil && resp.Code == 0 {
			globalOnSuccess()
			go globalPopulateCache(client, token)
			if resp.Data.Allowed {
				c.Next()
				return
			}
			c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "无权限"})
			c.Abort()
			return
		}

		// RBAC 不可达
		globalOnFailure()

		if failMode == FailModeDeny {
			c.JSON(http.StatusBadGateway, gin.H{"code": 1005, "message": "权限服务不可用"})
			c.Abort()
			return
		}

		if globalCheckFromCache(token, resource, action, cacheTTL) {
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "权限服务不可用, 缓存中无权限数据"})
		c.Abort()
	}
}

// ================================================================
// 全局缓存 + 熔断器
// ================================================================

type cacheEntry struct {
	perms     map[string][]string
	timestamp time.Time
}

func isGlobalCircuitOpen() bool {
	resilientMu.Lock()
	defer resilientMu.Unlock()
	if !resilientCircuitOpen {
		return false
	}
	if time.Since(resilientCircuitSince) > 30*time.Second {
		resilientCircuitOpen = false
		resilientFailCount = 0
		return false
	}
	return true
}

func globalOnSuccess() {
	resilientMu.Lock()
	defer resilientMu.Unlock()
	resilientFailCount = 0
	resilientCircuitOpen = false
}

func globalOnFailure() {
	resilientMu.Lock()
	defer resilientMu.Unlock()
	resilientFailCount++
	if resilientFailCount >= 5 {
		resilientCircuitOpen = true
		resilientCircuitSince = time.Now()
	}
}

// extractUserID 从 JWT payload 中提取 user_id（不验证签名）
func extractUserID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode payload: %w", err)
	}
	var claims struct {
		UserID uint `json:"user_id"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("unmarshal claims: %w", err)
	}
	if claims.UserID == 0 {
		return "", fmt.Errorf("no user_id in token")
	}
	return fmt.Sprintf("user:%d", claims.UserID), nil
}

func globalPopulateCache(client *RBACClient, token string) {
	key, err := extractUserID(token)
	if err != nil {
		return
	}
	menu, err := client.GetMenu(token)
	if err != nil || menu.Code != 0 {
		return
	}
	resilientMu.Lock()
	defer resilientMu.Unlock()
	resilientCacheStore[key] = cacheEntry{
		perms:     menu.Data.Permissions,
		timestamp: time.Now(),
	}
}

func globalCheckFromCache(token, resource, action string, ttl time.Duration) bool {
	key, err := extractUserID(token)
	if err != nil {
		return false
	}
	resilientMu.Lock()
	defer resilientMu.Unlock()

	entry, ok := resilientCacheStore[key]
	if !ok || time.Since(entry.timestamp) > ttl {
		return false
	}

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
