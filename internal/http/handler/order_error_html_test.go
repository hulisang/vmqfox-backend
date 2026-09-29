package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestWritePaymentErrorHTML 锁定 isHtml=1 下单失败时输出可读 HTML 错误页，且错误信息经过转义。
func TestWritePaymentErrorHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	writePaymentErrorHTML(c, `暂无可用支付二维码<script>alert(1)</script>`)

	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("错误页应为 HTML，实际 Content-Type=%q", contentType)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "暂无可用支付二维码") {
		t.Fatalf("错误页应展示错误信息，实际: %s", body)
	}
	if strings.Contains(body, "<script>") {
		t.Fatalf("错误信息必须转义，实际: %s", body)
	}
}
