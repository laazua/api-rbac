package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 基于内存的 IP 速率限制器
// 使用滑动窗口计数器算法，适合单实例部署
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // 时间窗口内允许的请求数
	interval time.Duration // 时间窗口大小
}

type visitor struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter 创建速率限制器
// rate: 每个时间窗口允许的最大请求数
// interval: 时间窗口大小
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		interval: interval,
	}
	go rl.cleanup()
	return rl
}

// Limit 返回一个 Gin 中间件，对超过速率限制的请求返回 429
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		now := time.Now()

		if !exists {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: now}
			rl.mu.Unlock()
			c.Next()
			return
		}

		// 时间窗口已过，重置计数
		if now.Sub(v.lastSeen) > rl.interval {
			v.count = 1
			v.lastSeen = now
			rl.mu.Unlock()
			c.Next()
			return
		}

		v.count++
		v.lastSeen = now

		if v.count > rl.rate {
			rl.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    1001,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		rl.mu.Unlock()
		c.Next()
	}
}

// cleanup 定期清理过期的访问记录
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.interval*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
