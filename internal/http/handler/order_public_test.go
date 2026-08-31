package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

// TestNewOrderHandlers 防止改造 DTO 时遗漏原有路由处理器装配。
func TestNewOrderHandlers(t *testing.T) {
	handlers := newOrderHandlers(nil)
	for name, routeHandler := range map[string]gin.HandlerFunc{
		"create":         handlers.CreateAPI,
		"get":            handlers.GetAPI,
		"list":           handlers.ListAPI,
		"statistics":     handlers.StatisticsAPI,
		"detail":         handlers.DetailAPI,
		"check":          handlers.CheckAPI,
		"return-url":     handlers.ReturnURLAPI,
		"close":          handlers.CloseAPI,
		"delete":         handlers.DeleteAPI,
		"delete-last":    handlers.DeleteLastAPI,
		"expired":        handlers.ExpiredAPI,
		"delete-expired": handlers.DeleteExpiredAPI,
		"reissue":        handlers.ReissueAPI,
	} {
		if routeHandler == nil {
			t.Errorf("订单处理器 %s 未装配", name)
		}
	}
}
