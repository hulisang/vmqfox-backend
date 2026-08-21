package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/config"
)

type rateBucket struct {
	started time.Time
	count   int
}

type fixedWindowLimiter struct {
	mu       sync.Mutex
	rule     config.RateLimitRule
	buckets  map[string]rateBucket
	lastTrim time.Time
}

// NewIPRateLimit 创建按客户端地址隔离的固定窗口限流器。
func NewIPRateLimit(rule config.RateLimitRule, trustedCIDRs []*net.IPNet, trustedIPs []net.IP) gin.HandlerFunc {
	resolver := NewClientIPResolver(trustedCIDRs, trustedIPs)
	return newRateLimitMiddleware(rule, func(c *gin.Context) string {
		return "ip:" + resolver.Resolve(c)
	})
}

// NewTokenRateLimit 创建按公开令牌隔离的固定窗口限流器；令牌只以摘要进入内存键。
func NewTokenRateLimit(rule config.RateLimitRule) gin.HandlerFunc {
	return newRateLimitMiddleware(rule, func(c *gin.Context) string {
		token := c.Param("token")
		if token == "" {
			// 公开路由沿用历史 :id 路径参数，但其语义已经切换为 publicToken。
			token = c.Param("id")
		}
		digest := sha256.Sum256([]byte(token))
		return "token:" + hex.EncodeToString(digest[:16])
	})
}

func newRateLimitMiddleware(rule config.RateLimitRule, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	limiter := &fixedWindowLimiter{
		rule:    rule,
		buckets: make(map[string]rateBucket),
	}
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || rule.Limit < 1 || rule.Window <= 0 {
			c.Next()
			return
		}

		allowed, retryAfter, remaining := limiter.allow(keyFunc(c), time.Now())
		c.Header("X-RateLimit-Limit", formatInt(rule.Limit))
		c.Header("X-RateLimit-Remaining", formatInt(remaining))
		if allowed {
			c.Next()
			return
		}

		c.Header("Retry-After", formatInt(retryAfter))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, php.NewEnvelope(429, "请求过于频繁，请稍后重试", nil))
	}
}

func (l *fixedWindowLimiter) allow(key string, now time.Time) (bool, int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastTrim.IsZero() || now.Sub(l.lastTrim) >= l.rule.Window {
		l.trimExpired(now)
		l.lastTrim = now
	}

	bucket, ok := l.buckets[key]
	if !ok || now.Sub(bucket.started) >= l.rule.Window {
		bucket = rateBucket{started: now}
	}
	if bucket.count >= l.rule.Limit {
		return false, retrySeconds(bucket.started.Add(l.rule.Window), now), 0
	}
	bucket.count++
	l.buckets[key] = bucket
	return true, 0, l.rule.Limit - bucket.count
}

func (l *fixedWindowLimiter) trimExpired(now time.Time) {
	for key, bucket := range l.buckets {
		if now.Sub(bucket.started) >= l.rule.Window {
			delete(l.buckets, key)
		}
	}
}

func retrySeconds(deadline, now time.Time) int {
	seconds := int(deadline.Sub(now) / time.Second)
	if deadline.After(now) && deadline.Sub(now)%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

func formatInt(value int) string {
	return strconv.Itoa(value)
}

// ClientIPResolver 只在 TCP 对端属于显式可信代理时解析转发头，阻断客户端伪造 X-Forwarded-For 绕过限流。
type ClientIPResolver struct {
	trustedCIDRs []*net.IPNet
	trustedIPs   []net.IP
}

func NewClientIPResolver(trustedCIDRs []*net.IPNet, trustedIPs []net.IP) ClientIPResolver {
	return ClientIPResolver{trustedCIDRs: trustedCIDRs, trustedIPs: trustedIPs}
}

func (r ClientIPResolver) Resolve(c *gin.Context) string {
	remote := parseHost(c.Request.RemoteAddr)
	if remote == nil {
		return "unknown"
	}
	if !r.isTrusted(remote) {
		return remote.String()
	}

	forwarded := forwardedIPs(c.GetHeader("X-Forwarded-For"))
	chain := append(forwarded, parseIP(c.GetHeader("X-Real-IP")))
	for index := len(chain) - 1; index >= 0; index-- {
		candidate := chain[index]
		if candidate == nil {
			continue
		}
		if !r.isTrusted(candidate) {
			return candidate.String()
		}
	}
	if len(forwarded) > 0 && forwarded[0] != nil {
		return forwarded[0].String()
	}
	return remote.String()
}

func (r ClientIPResolver) isTrusted(ip net.IP) bool {
	for _, allowed := range r.trustedIPs {
		if allowed != nil && allowed.Equal(ip) {
			return true
		}
	}
	for _, network := range r.trustedCIDRs {
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseHost(raw string) net.IP {
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return parseIP(host)
	}
	return parseIP(raw)
}

func parseIP(raw string) net.IP {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return net.ParseIP(value)
}

func forwardedIPs(raw string) []net.IP {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	result := make([]net.IP, 0, 4)
	for _, item := range strings.Split(raw, ",") {
		result = append(result, parseIP(item))
	}
	return result
}
