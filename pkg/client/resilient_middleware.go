package client

import (
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

// ResilientGuard 返回一个带韧性能力的 Gin 权限校验中间件。
//
// 特性:
//   - RBAC 正常 → 远程校验
//   - RBAC 不可达 → 根据 failMode: DENY=拒绝 / CACHE=查本地缓存
//   - 熔断保护: 连续失败超阈值后直接走缓存，避免超时堆积
//   - 自动恢复: 熔断后定期探测，恢复正常自动切回
//
// 用法:
//
//	r.Use(ResilientGuard(rbacClient, FailModeCache, 300, "user", "delete"))
func ResilientGuard(client *RBACClient, failMode FailMode, cacheTTLSec int, resource, action string) gin.HandlerFunc {
	rc := &resilientCache{
		client:     client,
		failMode:   failMode,
		cacheTTL:   time.Duration(cacheTTLSec) * time.Second,
		permCache:  make(map[string]cacheEntry),
		cbMaxFails: 5,
		cbRecovery: 30 * time.Second,
	}

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

		// 熔断状态下直接走缓存, 避免每次请求超时等待
		if rc.isCircuitOpen() {
			if failMode == FailModeDeny {
				c.JSON(http.StatusBadGateway, gin.H{"code": 1005, "message": "权限服务不可用(已熔断)"})
				c.Abort()
				return
			}
			// FailModeCache: 用本地缓存
			if rc.checkFromCache(token, resource, action) {
				c.Next()
				return
			}
			c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "无权限"})
			c.Abort()
			return
		}

		resp, err := client.CheckPermission(token, resource, action)
		if err == nil && resp.Code == 0 {
			rc.onSuccess()
			// 异步加载用户完整权限到本地缓存
			go rc.tryPopulateCache(token)
			if resp.Data.Allowed {
				c.Next()
				return
			}
			c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "无权限"})
			c.Abort()
			return
		}

		// RBAC 不可达
		rc.onFailure()

		if failMode == FailModeDeny {
			c.JSON(http.StatusBadGateway, gin.H{"code": 1005, "message": "权限服务不可用"})
			c.Abort()
			return
		}

		// FailModeCache: 降级到本地缓存
		if rc.checkFromCache(token, resource, action) {
			c.Next()
			return
		}
		// 缓存中也无 → 安全拒绝
		c.JSON(http.StatusForbidden, gin.H{"code": 1003, "message": "权限服务不可用, 缓存中无权限数据"})
		c.Abort()
	}
}

// ================================================================
// 韧性缓存实现
// ================================================================

type cacheEntry struct {
	perms     map[string][]string // resource → [action...]
	timestamp time.Time
}

type resilientCache struct {
	client     *RBACClient
	failMode   FailMode
	cacheTTL   time.Duration
	cbMaxFails int
	cbRecovery time.Duration

	mu            sync.Mutex
	permCache     map[string]cacheEntry // token → cacheEntry
	failureCount  int
	circuitOpen   bool
	circuitOpened time.Time
}

func (rc *resilientCache) isCircuitOpen() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !rc.circuitOpen {
		return false
	}
	if time.Since(rc.circuitOpened) > rc.cbRecovery {
		rc.circuitOpen = false
		rc.failureCount = 0
		return false
	}
	return true
}

func (rc *resilientCache) onSuccess() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.failureCount = 0
	rc.circuitOpen = false
}

func (rc *resilientCache) onFailure() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.failureCount++
	if rc.failureCount >= rc.cbMaxFails {
		rc.circuitOpen = true
		rc.circuitOpened = time.Now()
	}
}

// tryPopulateCache 异步加载用户完整权限到本地缓存
func (rc *resilientCache) tryPopulateCache(token string) {
	menu, err := rc.client.GetMenu(token)
	if err != nil || menu.Code != 0 {
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.permCache[token] = cacheEntry{
		perms:     menu.Data.Permissions,
		timestamp: time.Now(),
	}
}

func (rc *resilientCache) checkFromCache(token, resource, action string) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	entry, ok := rc.permCache[token]
	if !ok || time.Since(entry.timestamp) > rc.cacheTTL {
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
