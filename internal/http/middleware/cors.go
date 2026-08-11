package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigin string) gin.HandlerFunc {
	origin := strings.TrimSpace(allowedOrigin)
	if origin == "" {
		origin = "*"
	}

	return func(c *gin.Context) {
		requestOrigin := c.GetHeader("Origin")
		responseOrigin := origin
		if origin != "*" && requestOrigin != "" && requestOrigin != origin {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if origin != "*" && requestOrigin == origin {
			responseOrigin = requestOrigin
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Origin", responseOrigin)
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, X-Request-ID")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
