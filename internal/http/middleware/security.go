package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSPNonceKey 是本次请求 CSP nonce 在 gin 上下文中的键。
// 需要输出内联脚本的处理器（例如 isHtml=1 的支付跳转页）必须读取它，
// 而不是让 CSP 放开 'unsafe-inline'。
const CSPNonceKey = "csp_nonce"

// SecurityHeaders 统一下发安全响应头与内容安全策略。
//
// 设计取舍：
//   - script-src 只允许同源脚本与携带本次 nonce 的内联脚本，因此前端入口
//     不能再使用无 nonce 的内联 <script>。
//   - style-src 保留 'unsafe-inline'：Radix UI 与图表组件通过内联 style 定位，
//     内联样式的风险远低于内联脚本，这是当前唯一的必要放宽。
//   - frame-ancestors 'self' 与既有的 X-Frame-Options: SAMEORIGIN 语义一致，
//     不改变现有嵌入行为。
//   - HSTS 只在请求确实经由 HTTPS 到达时下发。当前部署仍允许 HTTP 监控端，
//     无条件下发 HSTS 会让浏览器强制升级并中断这些客户端。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := newCSPNonce()
		c.Set(CSPNonceKey, nonce)

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Content-Security-Policy", buildCSP(nonce))

		if requestIsHTTPS(c) {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// CSPNonce 读取当前请求的 nonce；未启用安全头中间件时返回空字符串。
func CSPNonce(c *gin.Context) string {
	value, exists := c.Get(CSPNonceKey)
	if !exists {
		return ""
	}
	nonce, ok := value.(string)
	if !ok {
		return ""
	}
	return nonce
}

func buildCSP(nonce string) string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self' 'nonce-" + nonce + "'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'self'",
	}
	return strings.Join(directives, "; ")
}

func newCSPNonce() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		// 随机源不可用时返回空 nonce：内联脚本会被 CSP 拒绝，属于安全的失败方向。
		return ""
	}
	return base64.RawStdEncoding.EncodeToString(value[:])
}

// requestIsHTTPS 判断请求真实到达协议。直连时看 TLS，网关终止 TLS 时看转发头。
// 转发头只有在 Gin 可信代理列表命中时才由 RemoteIP 之外的来源填充，
// 这里与限流共用同一份可信代理配置。
func requestIsHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	forwardedProto := c.GetHeader("X-Forwarded-Proto")
	if forwardedProto == "" {
		return false
	}
	// 多级代理会拼接为 "https, http"，只取最靠近客户端的一段。
	first := strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
	return strings.EqualFold(first, "https")
}
