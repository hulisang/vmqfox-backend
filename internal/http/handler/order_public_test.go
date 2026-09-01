package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

// TestPublicOrderViewWhitelist 防止匿名收银台响应重新暴露内部订单号或回调敏感字段。
func TestPublicOrderViewWhitelist(t *testing.T) {
	view := apiOrderViewData(usecase.OrderView{
		Order: order.Order{
			OrderID:          "internal-order-id",
			PublicToken:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			PayID:            "merchant-order-id",
			Type:             payment.Wechat,
			PriceCents:       100,
			ReallyPriceCents: 99,
			PriceText:        "1.00",
			ReallyPriceText:  "0.99",
			State:            order.StatusPending,
			Param:            "private-param",
			PayURL:           "weixin://pay/example",
			NotifyURL:        "https://merchant.example/notify",
			ReturnURL:        "https://merchant.example/return",
			CreatedAt:        time.Unix(1_700_000_000, 0),
		},
		StateText:        "未支付",
		TimeoutMinutes:   5,
		RemainingSeconds: 300,
	})

	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("编码公开订单响应失败: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("解析公开订单响应失败: %v", err)
	}
	for _, key := range []string{"orderId", "order_id", "publicToken", "param", "notifyUrl", "notify_url", "returnUrl", "return_url"} {
		if _, exists := data[key]; exists {
			t.Errorf("公开订单响应不应包含 %q，实际响应: %s", key, raw)
		}
	}
	for _, key := range []string{"payId", "payType", "price", "reallyPrice", "payUrl", "state", "remainingSeconds"} {
		if _, exists := data[key]; !exists {
			t.Errorf("公开订单响应缺少支付页所需字段 %q，实际响应: %s", key, raw)
		}
	}
}

// TestPublicOrderCheckWhitelist 确保轮询接口不会提前下发回跳地址或透传参数。
func TestPublicOrderCheckWhitelist(t *testing.T) {
	raw, err := json.Marshal(apiCheckOrderData(usecase.CheckOrderResult{
		State:            order.StatusPaid,
		RemainingSeconds: 0,
	}))
	if err != nil {
		t.Fatalf("编码公开订单状态失败: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("解析公开订单状态失败: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("公开订单状态响应字段应仅为状态和剩余时间，实际: %s", raw)
	}
	for _, key := range []string{"state", "remainingSeconds"} {
		if _, exists := data[key]; !exists {
			t.Errorf("公开订单状态响应缺少 %q，实际: %s", key, raw)
		}
	}
}

func TestQueryByPayIDResponseIncludesMerchantRecoveryFields(t *testing.T) {
	view := apiQueryByPayIDData(usecase.QueryByPayIDResult{
		Status:      order.StatusPaid,
		PublicToken: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Type:        payment.Wechat,
		Price:       "10.00",
		ReallyPrice: "10.01",
		CreatedAt:   time.Unix(1_700_000_000, 0),
		PaidAt:      time.Unix(1_700_000_100, 0),
	})

	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("编码按 payId 查询响应失败: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("解析按 payId 查询响应失败: %v", err)
	}
	for _, key := range []string{"status", "publicToken", "type", "price", "reallyPrice", "createdAt", "paidAt", "closedAt"} {
		if _, exists := data[key]; !exists {
			t.Errorf("商户查询响应缺少 %q，实际: %s", key, raw)
		}
	}
	for _, key := range []string{"notifyUrl", "returnUrl", "param", "orderId", "order_id"} {
		if _, exists := data[key]; exists {
			t.Errorf("商户查询响应不应包含 %q，实际: %s", key, raw)
		}
	}
	if data["publicToken"] != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("商户查询必须返回 publicToken 以便超时恢复，实际: %s", raw)
	}
	if paidAt, ok := data["paidAt"].(float64); !ok || paidAt != 1_700_000_100 {
		t.Fatalf("paidAt 应为 unix 秒，实际: %s", raw)
	}
	if closedAt, ok := data["closedAt"].(float64); !ok || closedAt != 0 {
		t.Fatalf("未关闭订单 closedAt 应为 0，实际: %s", raw)
	}
}

func TestWriteOrderErrorConflictUsesEnvelope409OnHTTP200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/order/create", nil)
	writeOrderError(ctx, &usecase.Error{
		Code:    usecase.CodeConflict,
		Message: "商户订单号已存在且请求字段不一致",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("冲突应保持 HTTP 200，实际 %d", recorder.Code)
	}
	var envelope php.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析冲突响应失败: %v body=%s", err, recorder.Body.Bytes())
	}
	if envelope.Code != 409 || envelope.Msg != "商户订单号已存在且请求字段不一致" || envelope.Data != nil {
		t.Fatalf("冲突 envelope 应为 code=409 且 data=null，实际 %+v", envelope)
	}
}

// TestNewOrderHandlers 防止改造 DTO 时遗漏原有路由处理器装配。
func TestNewOrderHandlers(t *testing.T) {
	handlers := newOrderHandlers(nil)
	for name, routeHandler := range map[string]gin.HandlerFunc{
		"create":          handlers.CreateAPI,
		"query-by-pay-id": handlers.QueryByPayIDAPI,
		"get":             handlers.GetAPI,
		"list":            handlers.ListAPI,
		"statistics":      handlers.StatisticsAPI,
		"detail":          handlers.DetailAPI,
		"check":           handlers.CheckAPI,
		"return-url":      handlers.ReturnURLAPI,
		"close":           handlers.CloseAPI,
		"delete":          handlers.DeleteAPI,
		"delete-last":     handlers.DeleteLastAPI,
		"expired":         handlers.ExpiredAPI,
		"delete-expired":  handlers.DeleteExpiredAPI,
		"reissue":         handlers.ReissueAPI,
	} {
		if routeHandler == nil {
			t.Errorf("订单处理器 %s 未装配", name)
		}
	}
}
