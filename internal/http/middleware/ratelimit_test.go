package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/config"
)

func TestFixedWindowLimiterIsolatesAndTrimsBuckets(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	limiter := &fixedWindowLimiter{
		rule: config.RateLimitRule{Limit: 2, Window: time.Minute},
		buckets: map[string]rateBucket{
			"expired": {started: now.Add(-time.Minute)},
		},
	}

	for request := 0; request < 2; request++ {
		allowed, _, remaining := limiter.allow("client-a", now)
		if !allowed || remaining != 1-request {
			t.Fatalf("client-a 第 %d 次请求的额度计算错误: allowed=%t remaining=%d", request+1, allowed, remaining)
		}
	}
	if allowed, retryAfter, remaining := limiter.allow("client-a", now); allowed || retryAfter != 60 || remaining != 0 {
		t.Fatalf("额度耗尽后应拒绝且返回 60 秒退避，实际 allowed=%t retryAfter=%d remaining=%d", allowed, retryAfter, remaining)
	}
	if allowed, _, remaining := limiter.allow("client-b", now); !allowed || remaining != 1 {
		t.Fatalf("不同客户端应使用独立桶，实际 allowed=%t remaining=%d", allowed, remaining)
	}
	if _, exists := limiter.buckets["expired"]; exists {
		t.Fatal("过期桶应在惰性清理时移除")
	}
}

func TestTokenRateLimitReturns429AndSkipsOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	called := 0
	router.Any("/api/order/check/:token",
		NewTokenRateLimit(config.RateLimitRule{Limit: 1, Window: time.Minute}),
		func(c *gin.Context) {
			called++
			c.Status(http.StatusNoContent)
		},
	)

	request := func(method, token string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, "/api/order/check/"+token, nil))
		return recorder
	}

	if recorder := request(http.MethodOptions, "token-a"); recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS 应放行且不消耗额度，实际状态 %d", recorder.Code)
	}
	if recorder := request(http.MethodGet, "token-a"); recorder.Code != http.StatusNoContent {
		t.Fatalf("令牌首次请求应放行，实际状态 %d", recorder.Code)
	}
	limited := request(http.MethodGet, "token-a")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("令牌超过额度应返回 429，实际状态 %d", limited.Code)
	}
	if got := limited.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("429 应携带 Retry-After=60，实际 %q", got)
	}
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(limited.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析 429 响应失败: %v", err)
	}
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("429 应保留兼容 envelope code，实际 %d", response.Code)
	}
	if recorder := request(http.MethodGet, "token-b"); recorder.Code != http.StatusNoContent {
		t.Fatalf("不同令牌应使用独立额度，实际状态 %d", recorder.Code)
	}
	if called != 3 {
		t.Fatalf("被限流请求不应继续执行 handler，实际调用次数 %d", called)
	}
}

func TestClientIPResolverTrustsForwardedHeadersOnlyFromConfiguredProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	request.RemoteAddr = "198.51.100.9:12345"
	request.Header.Set("X-Forwarded-For", "203.0.113.9")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	if actual := NewClientIPResolver(nil, nil).Resolve(context); actual != "198.51.100.9" {
		t.Fatalf("未配置可信代理时不应信任 X-Forwarded-For，实际 %s", actual)
	}

	_, trustedNetwork, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("构造可信代理网段失败: %v", err)
	}
	request.RemoteAddr = "10.0.0.5:443"
	request.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.6")
	if actual := NewClientIPResolver([]*net.IPNet{trustedNetwork}, nil).Resolve(context); actual != "203.0.113.9" {
		t.Fatalf("可信代理转发链应解析首个非可信客户端地址，实际 %s", actual)
	}
}
