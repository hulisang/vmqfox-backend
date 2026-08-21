package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSWildcardAllowsAnyOriginWithout403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS("*"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	origins := []string{
		"http://localhost:3006",
		"http://127.0.0.1:3006",
		"vscode-webview://example",
		"http://localhost:5173",
	}

	for _, origin := range origins {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Origin %s 请求被拦截，状态码为 %d，期望 200", origin, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("Origin %s 回显不正确，实际 %s", origin, got)
		}
	}
}

func TestCORSSpecificOriginDoesNotAbortWith403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS("http://localhost:3006"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://127.0.0.1:3006")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// 不匹配时不应直接以 403 强行阻断请求
	if rec.Code == http.StatusForbidden {
		t.Fatal("未匹配的 Origin 不应直接被 403 拒绝")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("请求状态码为 %d，期望 200", rec.Code)
	}
}

func TestCORSOptionsReturns204(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS("*"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3006")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS 预检请求应返回 204，实际为 %d", rec.Code)
	}
}