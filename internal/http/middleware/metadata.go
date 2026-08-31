package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
)

const RequestIDKey = "request_id"

// RequestMetadata 只负责请求标识；安全响应头由 SecurityHeaders 统一下发，避免两处重复设置。
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
				logger.Printf("请求异常 request_id=%v method=%s path=%s error=%v\n%s", requestID, c.Request.Method, redactPath(c.Request.URL.Path), recovered, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, php.NewEnvelope(500, "服务器处理请求时发生错误", nil))
			}
		}()
		c.Next()
	}
}

// AccessLog 记录脱敏访问日志，并与限流器复用同一可信代理解析规则。
func AccessLog(logger *log.Logger, clientIPResolver ClientIPResolver) gin.HandlerFunc {
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
			redactPath(c.Request.URL.Path),
			c.Writer.Status(),
			time.Since(startedAt),
			clientIPResolver.Resolve(c),
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

// redactPath 隐去公开订单路径中的 bearer token，避免访问日志变成可复用的支付凭据。
func redactPath(path string) string {
	parts := strings.Split(path, "/")
	for index := 0; index+1 < len(parts); index++ {
		switch parts[index] {
		case "get", "check", "return-url":
			if index > 1 && parts[index-1] == "order" {
				parts[index+1] = "[redacted-token]"
			}
		}
	}
	return strings.Join(parts, "/")
}
