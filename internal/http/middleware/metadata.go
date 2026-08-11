package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
)

const RequestIDKey = "request_id"

func RequestMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func Recovery(logger *log.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = log.Default()
	}
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get(RequestIDKey)
				logger.Printf("请求异常 request_id=%v method=%s path=%s error=%v\n%s", requestID, c.Request.Method, c.Request.URL.Path, recovered, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, php.NewEnvelope(500, "服务器处理请求时发生错误", nil))
			}
		}()
		c.Next()
	}
}

func AccessLog(logger *log.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = log.Default()
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		requestID, _ := c.Get(RequestIDKey)
		logger.Printf(
			"HTTP request_id=%v method=%s path=%s status=%d duration=%s client=%s",
			requestID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(startedAt),
			c.ClientIP(),
		)
	}
}

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return time.Now().UTC().Format("20060102T150405.000000000")
}
