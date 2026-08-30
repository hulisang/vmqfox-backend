package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newCORSRouter 构造只挂载 CORS 中间件的最小路由，便于逐项断言响应头。
func newCORSRouter(allowedOrigin string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS(allowedOrigin))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return router
}

// TestCORSAllowsConfiguredOrigin 验证已显式配置的 Origin 会被回显。
func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	router := newCORSRouter("https://pay.example.com, https://admin.example.com")

	for _, origin := range []string{"https://pay.example.com", "https://admin.example.com"} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Origin %s 请求状态码为 %d，期望 200", origin, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("Origin %s 未被回显，实际 %q", origin, got)
		}
		if got := rec.Header().Get("Vary"); got != "Origin" {
			t.Errorf("缺少 Vary: Origin，实际 %q", got)
		}
	}
}

// TestCORSRejectsUnlistedOrigin 未列入白名单的 Origin 不得获得跨域许可。
func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	router := newCORSRouter("https://pay.example.com")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("未授权 Origin 不应获得 Allow-Origin，实际 %q", got)
	}
}

// TestCORSEmptyConfigDeniesCrossOrigin 空配置必须默认拒绝跨域，不得回退为通配符。
func TestCORSEmptyConfigDeniesCrossOrigin(t *testing.T) {
	router := newCORSRouter("")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("空配置下不应下发 Allow-Origin，实际 %q", got)
	}
}

// TestCORSWildcardConfigIsIgnored 配置写成 * 时按拒绝处理，避免误配导致全站放开。
func TestCORSWildcardConfigIsIgnored(t *testing.T) {
	router := newCORSRouter("*")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("通配配置应被忽略，实际 %q", got)
	}
}

// TestCORSSameOriginRequestUnaffected 同源请求不带 Origin，不应受跨域策略影响。
func TestCORSSameOriginRequestUnaffected(t *testing.T) {
	router := newCORSRouter("")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("同源请求状态码为 %d，期望 200", rec.Code)
	}
}

// TestCORSOptionsReturns204 预检请求统一返回 204，不进入业务处理器。
func TestCORSOptionsReturns204(t *testing.T) {
	router := newCORSRouter("https://pay.example.com")

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://pay.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS 预检请求应返回 204，实际为 %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("预检响应缺少 Access-Control-Allow-Methods")
	}
}

// TestNormalizeOriginTrimsPathAndSlash 归一化只保留 scheme://host[:port]。
func TestNormalizeOriginTrimsPathAndSlash(t *testing.T) {
	cases := map[string]string{
		"https://pay.example.com/":      "https://pay.example.com",
		"https://pay.example.com/admin": "https://pay.example.com",
		"http://127.0.0.1:8080":         "http://127.0.0.1:8080",
	}
	for input, expected := range cases {
		if got := normalizeOrigin(input); got != expected {
			t.Errorf("normalizeOrigin(%q) = %q，期望 %q", input, got, expected)
		}
	}
}
