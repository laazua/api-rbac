package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 返回一个限制请求体大小的 Gin 中间件
// 超过限制的请求返回 413 Request Entity Too Large
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":    1001,
				"message": "请求体过大",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
