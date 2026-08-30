package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 只对显式配置的 Origin 回显跨域许可。
//
// 与旧实现的关键差别：配置为空时**拒绝**所有跨域请求，不再回退为通配符。
// 单服务同源部署（Go 同时提供 / 与 /api）本身不触发 CORS，
// 因此空配置不影响正常管理台使用，只会阻断真正的跨站调用。
func CORS(allowedOrigin string) gin.HandlerFunc {
	allowedList := parseAllowedOrigins(allowedOrigin)

	return func(c *gin.Context) {
		requestOrigin := c.GetHeader("Origin")

		// 无论是否命中，响应都依赖 Origin，必须声明 Vary 以避免缓存串用。
		c.Header("Vary", "Origin")

		if requestOrigin != "" && originAllowed(allowedList, requestOrigin) {
			c.Header("Access-Control-Allow-Origin", requestOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-Requested-With, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining, X-Request-ID")
			c.Header("Access-Control-Max-Age", "600")
		}

		// 预检请求不进入业务处理器：未获许可的 Origin 得不到 Allow-Origin 头，浏览器会自行阻断。
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// parseAllowedOrigins 解析逗号分隔的 Origin 列表。
// 通配符 "*" 被显式丢弃：安全默认拒绝优先于配置便利。
func parseAllowedOrigins(raw string) []string {
	var allowed []string
	for _, item := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || trimmed == "*" {
			continue
		}
		allowed = append(allowed, normalizeOrigin(trimmed))
	}
	return allowed
}

// normalizeOrigin 只保留 scheme://host[:port]，去掉路径和末尾斜杠，
// 使 "https://pay.example.com/" 与 "https://pay.example.com" 视为同一 Origin。
func normalizeOrigin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	return parsed.Scheme + "://" + parsed.Host
}

func originAllowed(allowed []string, requestOrigin string) bool {
	candidate := normalizeOrigin(requestOrigin)
	for _, item := range allowed {
		if item == candidate {
			return true
		}
	}
	return false
}
