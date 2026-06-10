package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"api-rbac/config"
)

func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowAll := false
	for _, origin := range cfg.AllowOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}

	return func(c *gin.Context) {
		reqOrigin := c.Request.Header.Get("Origin")

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if reqOrigin != "" {
			for _, allowed := range cfg.AllowOrigins {
				if strings.EqualFold(reqOrigin, allowed) {
					c.Header("Access-Control-Allow-Origin", reqOrigin)
					c.Header("Vary", "Origin")
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
