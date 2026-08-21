package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigin string) gin.HandlerFunc {
	rawOrigin := strings.TrimSpace(allowedOrigin)
	if rawOrigin == "" {
		rawOrigin = "*"
	}

	// 支持逗号分隔的多 Origin 配置
	var allowedList []string
	isWildcard := false
	if rawOrigin == "*" {
		isWildcard = true
	} else {
		for _, item := range strings.Split(rawOrigin, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed == "*" {
				isWildcard = true
				break
			}
			if trimmed != "" {
				allowedList = append(allowedList, trimmed)
			}
		}
	}

	return func(c *gin.Context) {
		requestOrigin := c.GetHeader("Origin")
		responseOrigin := "*"

		if isWildcard {
			if requestOrigin != "" {
				responseOrigin = requestOrigin
				c.Header("Vary", "Origin")
			}
		} else if requestOrigin != "" {
			matched := false
			for _, allowed := range allowedList {
				if allowed == requestOrigin {
					responseOrigin = requestOrigin
					c.Header("Vary", "Origin")
					matched = true
					break
				}
			}
			if !matched && len(allowedList) > 0 {
				responseOrigin = allowedList[0]
			}
		}

		c.Header("Access-Control-Allow-Origin", responseOrigin)
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining, X-Request-ID")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

