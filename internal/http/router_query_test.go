package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQueryByPayIDRouteIsPOSTOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterDeps{})

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, "/api/order/query-by-pay-id?payId=x&t=1&sign=y", nil))
	if getRecorder.Code != http.StatusNotFound {
		t.Fatalf("query-by-pay-id 不得接受 GET，实际状态 %d body=%s", getRecorder.Code, getRecorder.Body.Bytes())
	}

	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, httptest.NewRequest(http.MethodPost, "/api/order/query-by-pay-id", nil))
	if postRecorder.Code == http.StatusNotFound {
		t.Fatal("query-by-pay-id 应注册 POST")
	}
}
