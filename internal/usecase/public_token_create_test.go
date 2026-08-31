package usecase

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/setting"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

// tokenConflictOrders 只覆写创建与令牌查询，用于模拟公开令牌唯一索引冲突；
// 其余仓储方法沿用 stubOrderRepository 的空实现。
type tokenConflictOrders struct {
	stubOrderRepository
	conflicts int
	created   []order.Order
	nextID    int64
}

func (r *tokenConflictOrders) Create(_ context.Context, value order.Order) (order.Order, error) {
	if r.conflicts > 0 {
		r.conflicts--
		return order.Order{}, port.ErrPublicTokenConflict
	}
	r.nextID++
	value.ID = r.nextID
	r.created = append(r.created, value)
	return value, nil
}

func (r *tokenConflictOrders) FindByPublicToken(_ context.Context, token string) (order.Order, error) {
	for _, value := range r.created {
		if value.PublicToken == token {
			return value, nil
		}
	}
	return order.Order{}, port.ErrNotFound
}

var _ port.OrderRepository = (*tokenConflictOrders)(nil)

func createOrderInput(key string) CreateOrderInput {
	input := CreateOrderInput{
		PayID:     "merchant-order-1",
		Param:     "param-1",
		Type:      strconv.Itoa(int(payment.Alipay)),
		Price:     "1.00",
		NotifyURL: "https://merchant.test/notify",
		ReturnURL: "https://merchant.test/return",
	}
	input.Sign = payment.CreateSignV2(input.PayID, input.Param, input.Type, input.Price, input.NotifyURL, input.ReturnURL, key)
	return input
}

func newCreateOrderService(t *testing.T, tokens *scriptedTokens, orders *tokenConflictOrders) *OrderService {
	t.Helper()
	service, err := NewOrderService(OrderServiceDeps{
		Transactions: &passthroughTransactions{},
		Orders:       orders,
		PriceLocks:   grantingPriceLocks{},
		QRCodes:      emptyQRCodes{},
		Settings: newMapSettings(map[string]string{
			setting.MerchantKey:     testMerchantKey,
			setting.MonitorStateKey: "1",
			setting.AlipayPayURLKey: "https://pay.test/alipay",
			setting.CloseMinutesKey: "5",
		}),
		Outbox:       &recordingOutbox{},
		Clock:        fakeClock{now: time.Unix(1_700_000_000, 0)},
		OrderIDs:     fakeOrderIDs{value: "2026082500000000001"},
		PublicTokens: tokens,
		FrontendURL:  "https://pay.example.test/",
	})
	if err != nil {
		t.Fatalf("构造订单用例失败: %v", err)
	}
	return service
}

// TestCreateOrderIssuesPublicTokenAndTokenOnlyRedirect 锁定建单路径必须写入合法令牌，且回跳链接只携带令牌。
func TestCreateOrderIssuesPublicTokenAndTokenOnlyRedirect(t *testing.T) {
	token := testToken(1)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)

	result, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	if err != nil {
		t.Fatalf("建单应成功，实际 err=%v", err)
	}
	if result.Order.PublicToken != token {
		t.Fatalf("订单应写入生成的令牌，期望 %q 实际 %q", token, result.Order.PublicToken)
	}
	if !order.IsValidPublicToken(result.Order.PublicToken) {
		t.Fatalf("订单令牌格式无效: %q", result.Order.PublicToken)
	}
	if len(orders.created) != 1 || orders.created[0].PublicToken != token {
		t.Fatalf("持久化订单应携带令牌，实际 %+v", orders.created)
	}
	expectedRedirect := "https://pay.example.test/#/payment/" + token
	if result.RedirectURL != expectedRedirect {
		t.Fatalf("回跳链接应只携带令牌，期望 %q 实际 %q", expectedRedirect, result.RedirectURL)
	}
	if orders.created[0].OrderID == "" {
		t.Fatal("内部订单号应保留给管理端使用")
	}
}

// TestCreateOrderRetriesOnPublicTokenConflict 覆盖唯一索引冲突后的重试路径。
func TestCreateOrderRetriesOnPublicTokenConflict(t *testing.T) {
	conflicting := testToken(2)
	accepted := testToken(3)
	tokens := &scriptedTokens{tokens: []string{conflicting, conflicting, accepted}}
	orders := &tokenConflictOrders{conflicts: 2}
	service := newCreateOrderService(t, tokens, orders)

	result, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	if err != nil {
		t.Fatalf("冲突重试后建单应成功，实际 err=%v", err)
	}
	if tokens.calls != 3 {
		t.Fatalf("每次冲突都应重新生成令牌，期望 3 次实际 %d 次", tokens.calls)
	}
	if result.Order.PublicToken != accepted {
		t.Fatalf("最终应使用重试后的令牌，期望 %q 实际 %q", accepted, result.Order.PublicToken)
	}
}

// TestCreateOrderFailsWhenPublicTokenAlwaysConflicts 锁定重试耗尽时不落库、不返回空令牌订单。
func TestCreateOrderFailsWhenPublicTokenAlwaysConflicts(t *testing.T) {
	tokens := &scriptedTokens{tokens: repeatedTokens(testToken(4), publicTokenCreateRetryMax)}
	orders := &tokenConflictOrders{conflicts: publicTokenCreateRetryMax}
	service := newCreateOrderService(t, tokens, orders)

	_, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	code, ok := ErrorCodeOf(err)
	if !ok || code != CodeDependency {
		t.Fatalf("重试耗尽应返回依赖错误，实际 err=%v code=%v", err, code)
	}
	if tokens.calls != publicTokenCreateRetryMax {
		t.Fatalf("重试次数应等于上限 %d，实际 %d", publicTokenCreateRetryMax, tokens.calls)
	}
	if len(orders.created) != 0 {
		t.Fatalf("重试耗尽后不应留下订单，实际 %d 条", len(orders.created))
	}
}

// TestCreateOrderRejectsInvalidGeneratedToken 锁定生成器返回非法格式时拒绝建单，避免写入弱公开凭据。
func TestCreateOrderRejectsInvalidGeneratedToken(t *testing.T) {
	tokens := &scriptedTokens{tokens: []string{"SHORT-TOKEN"}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)

	_, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	code, ok := ErrorCodeOf(err)
	if !ok || code != CodeDependency {
		t.Fatalf("非法令牌应返回依赖错误，实际 err=%v code=%v", err, code)
	}
	if len(orders.created) != 0 {
		t.Fatalf("非法令牌不应落库，实际 %d 条", len(orders.created))
	}
}

func newMonitorService(t *testing.T, tokens *scriptedTokens, orders *tokenConflictOrders, now time.Time) *MonitorService {
	t.Helper()
	service, err := NewMonitorService(MonitorServiceDeps{
		Transactions: &passthroughTransactions{},
		Orders:       orders,
		PriceLocks:   grantingPriceLocks{},
		Settings:     newMapSettings(map[string]string{setting.MerchantKey: testMerchantKey}),
		Events:       newEventsRepository{},
		Outbox:       &recordingOutbox{},
		Clock:        fakeClock{now: now},
		PublicTokens: tokens,
		SignTTL:      5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("构造监控用例失败: %v", err)
	}
	return service
}

func pushInput(now time.Time, key string) PaymentPushInput {
	input := PaymentPushInput{
		Timestamp: strconv.FormatInt(now.UnixMilli(), 10),
		Type:      strconv.Itoa(int(payment.Alipay)),
		Price:     "3.21",
	}
	input.Sign = payment.PushSignV2(input.Type, input.Price, input.Timestamp, key)
	return input
}

// TestUnmatchedTransferIssuesPublicToken 锁定无订单转账同样生成合法令牌，避免留下无令牌订单破坏非空约束。
func TestUnmatchedTransferIssuesPublicToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := testToken(5)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newMonitorService(t, tokens, orders, now)

	result, err := service.Push(context.Background(), pushInput(now, testMerchantKey))
	if err != nil {
		t.Fatalf("无订单转账推送应成功，实际 err=%v", err)
	}
	if result.Matched || result.Duplicate {
		t.Fatalf("未匹配订单时不应标记匹配或重复，实际 %+v", result)
	}
	if result.Order.PublicToken != token {
		t.Fatalf("无订单转账应写入令牌，期望 %q 实际 %q", token, result.Order.PublicToken)
	}
	if !order.IsValidPublicToken(result.Order.PublicToken) {
		t.Fatalf("无订单转账令牌格式无效: %q", result.Order.PublicToken)
	}
	if len(orders.created) != 1 || orders.created[0].PublicToken != token {
		t.Fatalf("持久化的无订单转账应携带令牌，实际 %+v", orders.created)
	}
	if orders.created[0].State != order.StatusPaid {
		t.Fatalf("无订单转账应记为已支付，实际状态 %d", orders.created[0].State)
	}
}

// TestUnmatchedTransferRetriesOnPublicTokenConflict 覆盖无订单转账在令牌冲突时的重试与重试耗尽行为。
func TestUnmatchedTransferRetriesOnPublicTokenConflict(t *testing.T) {
	now := time.Unix(1_700_000_100, 0)
	accepted := testToken(6)
	tokens := &scriptedTokens{tokens: []string{testToken(7), accepted}}
	orders := &tokenConflictOrders{conflicts: 1}
	service := newMonitorService(t, tokens, orders, now)

	result, err := service.Push(context.Background(), pushInput(now, testMerchantKey))
	if err != nil {
		t.Fatalf("冲突重试后应成功，实际 err=%v", err)
	}
	if tokens.calls != 2 {
		t.Fatalf("冲突应触发一次重试，期望 2 次调用实际 %d 次", tokens.calls)
	}
	if result.Order.PublicToken != accepted {
		t.Fatalf("应使用重试后的令牌，期望 %q 实际 %q", accepted, result.Order.PublicToken)
	}

	exhaustedTokens := &scriptedTokens{tokens: repeatedTokens(testToken(8), publicTokenCreateRetryMax)}
	exhaustedOrders := &tokenConflictOrders{conflicts: publicTokenCreateRetryMax}
	exhausted := newMonitorService(t, exhaustedTokens, exhaustedOrders, now)
	_, err = exhausted.Push(context.Background(), pushInput(now, testMerchantKey))
	code, ok := ErrorCodeOf(err)
	if !ok || code != CodeDependency {
		t.Fatalf("重试耗尽应返回依赖错误，实际 err=%v code=%v", err, code)
	}
	if len(exhaustedOrders.created) != 0 {
		t.Fatalf("重试耗尽后不应留下订单，实际 %d 条", len(exhaustedOrders.created))
	}
}
