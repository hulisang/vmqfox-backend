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

func signCreateInput(input CreateOrderInput, key string) CreateOrderInput {
	input.Sign = payment.CreateSignV2(input.PayID, input.Param, input.Type, input.Price, input.NotifyURL, input.ReturnURL, key)
	return input
}

func queryByPayIDInput(payID string, now time.Time, key string) QueryByPayIDInput {
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	return QueryByPayIDInput{
		PayID:     payID,
		Timestamp: timestamp,
		Sign:      payment.QueryByPayIDSignV2(payID, timestamp, key),
	}
}

func TestCreateRequestDigestIsStableAndFieldSensitive(t *testing.T) {
	base := createRequestDigest("2", "1.00", "param-1", "https://merchant.test/notify", "https://merchant.test/return")
	same := createRequestDigest("2", "1.00", "param-1", "https://merchant.test/notify", "https://merchant.test/return")
	if base != same || len(base) != 64 {
		t.Fatalf("相同建单字段应得到稳定 64 位摘要，实际 %q / %q", base, same)
	}

	changed := map[string]string{
		"type":      createRequestDigest("1", "1.00", "param-1", "https://merchant.test/notify", "https://merchant.test/return"),
		"price":     createRequestDigest("2", "1.01", "param-1", "https://merchant.test/notify", "https://merchant.test/return"),
		"param":     createRequestDigest("2", "1.00", "param-2", "https://merchant.test/notify", "https://merchant.test/return"),
		"notifyUrl": createRequestDigest("2", "1.00", "param-1", "https://other.test/notify", "https://merchant.test/return"),
		"returnUrl": createRequestDigest("2", "1.00", "param-1", "https://merchant.test/notify", "https://other.test/return"),
	}
	for field, digest := range changed {
		if digest == base {
			t.Errorf("改写 %s 后摘要未变化", field)
		}
	}
}

func TestCreateOrderReplaysIdenticalPayID(t *testing.T) {
	token := testToken(9)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)
	input := createOrderInput(testMerchantKey)

	first, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("首次建单应成功，实际 err=%v", err)
	}
	second, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("相同字段重复建单应重放原订单，实际 err=%v", err)
	}
	if len(orders.created) != 1 {
		t.Fatalf("重复建单不应再插入订单，实际 %d 条", len(orders.created))
	}
	if tokens.calls != 1 {
		t.Fatalf("重放路径不应再消耗公开令牌，实际调用 %d 次", tokens.calls)
	}
	if second.Order.PublicToken != first.Order.PublicToken || second.Order.PublicToken != token {
		t.Fatalf("重放应返回原 publicToken，首次=%q 重放=%q", first.Order.PublicToken, second.Order.PublicToken)
	}
	if second.ReallyPrice != first.ReallyPrice {
		t.Fatalf("重放应返回原 reallyPrice，首次=%q 重放=%q", first.ReallyPrice, second.ReallyPrice)
	}
	if second.RedirectURL != first.RedirectURL || second.RedirectURL == "" {
		t.Fatalf("重放应返回原 redirectUrl，首次=%q 重放=%q", first.RedirectURL, second.RedirectURL)
	}
}

func TestCreateOrderConflictsWhenPayIDFieldsDiffer(t *testing.T) {
	token := testToken(10)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)
	original := createOrderInput(testMerchantKey)
	if _, err := service.Create(context.Background(), original); err != nil {
		t.Fatalf("首次建单应成功，实际 err=%v", err)
	}

	cases := map[string]func(CreateOrderInput) CreateOrderInput{
		"type": func(in CreateOrderInput) CreateOrderInput {
			in.Type = strconv.Itoa(int(payment.Wechat))
			return in
		},
		"price": func(in CreateOrderInput) CreateOrderInput {
			in.Price = "2.00"
			return in
		},
		"param": func(in CreateOrderInput) CreateOrderInput {
			in.Param = "param-changed"
			return in
		},
		"notifyUrl": func(in CreateOrderInput) CreateOrderInput {
			in.NotifyURL = "https://merchant.test/notify-other"
			return in
		},
		"returnUrl": func(in CreateOrderInput) CreateOrderInput {
			in.ReturnURL = "https://merchant.test/return-other"
			return in
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := service.Create(context.Background(), signCreateInput(mutate(original), testMerchantKey))
			code, ok := ErrorCodeOf(err)
			if !ok || code != CodeConflict {
				t.Fatalf("字段冲突应返回 %s，实际 err=%v code=%v", CodeConflict, err, code)
			}
			if len(orders.created) != 1 {
				t.Fatalf("冲突请求不得改写原订单，实际 %d 条", len(orders.created))
			}
			if orders.created[0].PublicToken != token {
				t.Fatalf("冲突后原 publicToken 被改写: %q", orders.created[0].PublicToken)
			}
			if orders.created[0].PriceText != original.Price || orders.created[0].Param != original.Param {
				t.Fatalf("冲突后原订单字段被改写: %+v", orders.created[0])
			}
		})
	}
}

func TestQueryByPayIDSuccess(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := testToken(11)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)
	created, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	if err != nil {
		t.Fatalf("建单应成功，实际 err=%v", err)
	}

	result, err := service.QueryByPayID(context.Background(), queryByPayIDInput(created.Order.PayID, now, testMerchantKey))
	if err != nil {
		t.Fatalf("按 payId 查询应成功，实际 err=%v", err)
	}
	if result.PublicToken != token || result.Status != order.StatusPending || result.Type != payment.Alipay {
		t.Fatalf("查询结果与原订单不符: %+v", result)
	}
	if result.Price != "1.00" || result.ReallyPrice != created.ReallyPrice {
		t.Fatalf("查询金额与原订单不符: price=%q reallyPrice=%q", result.Price, result.ReallyPrice)
	}
	if result.CreatedAt.Unix() != now.Unix() || !result.PaidAt.IsZero() || !result.ClosedAt.IsZero() {
		t.Fatalf("查询时间戳与原订单不符: %+v", result)
	}
}

func TestQueryByPayIDRejectsExpiredTimestamp(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	service := newCreateOrderService(t, &scriptedTokens{tokens: []string{testToken(12)}}, &tokenConflictOrders{})
	if _, err := service.Create(context.Background(), createOrderInput(testMerchantKey)); err != nil {
		t.Fatalf("建单应成功，实际 err=%v", err)
	}

	stale := now.Add(-6 * time.Minute)
	_, err := service.QueryByPayID(context.Background(), queryByPayIDInput("merchant-order-1", stale, testMerchantKey))
	code, ok := ErrorCodeOf(err)
	if !ok || code != CodeStaleTimestamp {
		t.Fatalf("过期时间戳应返回 %s，实际 err=%v code=%v", CodeStaleTimestamp, err, code)
	}
}

func TestQueryByPayIDRejectsBadSignature(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	service := newCreateOrderService(t, &scriptedTokens{tokens: []string{testToken(13)}}, &tokenConflictOrders{})
	if _, err := service.Create(context.Background(), createOrderInput(testMerchantKey)); err != nil {
		t.Fatalf("建单应成功，实际 err=%v", err)
	}

	input := queryByPayIDInput("merchant-order-1", now, testMerchantKey)
	input.Sign = "00" + input.Sign[2:]
	_, err := service.QueryByPayID(context.Background(), input)
	code, ok := ErrorCodeOf(err)
	if !ok || code != CodeInvalidSignature {
		t.Fatalf("错误签名应返回 %s，实际 err=%v code=%v", CodeInvalidSignature, err, code)
	}
}

// payIDRaceOrders 模拟「首次 Find 未命中、Create 撞上已提交的 pay_id、随后 Find 命中」的超时重试竞态。
type payIDRaceOrders struct {
	stubOrderRepository
	winner          order.Order
	findMisses      int
	createConflicts int
	created         []order.Order
	findCalls       int
	createCalls     int
}

func (r *payIDRaceOrders) FindByPayID(_ context.Context, payID string) (order.Order, error) {
	if r.winner.PayID == payID {
		return r.winner, nil
	}
	return order.Order{}, port.ErrNotFound
}

func (r *payIDRaceOrders) FindByPayIDForUpdate(_ context.Context, payID string) (order.Order, error) {
	r.findCalls++
	if r.findMisses > 0 {
		r.findMisses--
		return order.Order{}, port.ErrNotFound
	}
	if r.winner.PayID == payID {
		return r.winner, nil
	}
	return order.Order{}, port.ErrNotFound
}

func (r *payIDRaceOrders) Create(_ context.Context, value order.Order) (order.Order, error) {
	r.createCalls++
	if r.createConflicts > 0 {
		r.createConflicts--
		return order.Order{}, port.ErrConflict
	}
	r.created = append(r.created, value)
	return value, nil
}

var _ port.OrderRepository = (*payIDRaceOrders)(nil)

type recordingPriceLocks struct {
	held     map[string]struct{}
	acquired []string
	released []string
}

func (l *recordingPriceLocks) TryAcquire(_ context.Context, orderID string, _ payment.Type, _ int64) (bool, error) {
	if l.held == nil {
		l.held = make(map[string]struct{})
	}
	l.held[orderID] = struct{}{}
	l.acquired = append(l.acquired, orderID)
	return true, nil
}

func (l *recordingPriceLocks) ReleaseByOrderID(_ context.Context, orderID string) error {
	delete(l.held, orderID)
	l.released = append(l.released, orderID)
	return nil
}

func (l *recordingPriceLocks) ReleaseBefore(context.Context, time.Time) error { return nil }

func (l *recordingPriceLocks) liveCount() int { return len(l.held) }

var _ port.PriceLockRepository = (*recordingPriceLocks)(nil)

func raceWinnerOrder(token string, mutate func(*order.Order)) order.Order {
	value := order.Order{
		ID:               1,
		OrderID:          "winner-order",
		PublicToken:      token,
		PayID:            "merchant-order-1",
		Type:             payment.Alipay,
		PriceCents:       100,
		ReallyPriceCents: 101,
		PriceText:        "1.00",
		ReallyPriceText:  "1.01",
		State:            order.StatusPending,
		Param:            "param-1",
		NotifyURL:        "https://merchant.test/notify",
		ReturnURL:        "https://merchant.test/return",
		CreatedAt:        time.Unix(1_700_000_000, 0),
	}
	if mutate != nil {
		mutate(&value)
	}
	return value
}

func TestCreateOrderPayIDRaceReplaysWithoutOrphanLock(t *testing.T) {
	token := testToken(16)
	orders := &payIDRaceOrders{
		winner:          raceWinnerOrder(token, nil),
		findMisses:      1,
		createConflicts: 1,
	}
	locks := &recordingPriceLocks{}
	service := newCreateOrderServiceWith(t, &scriptedTokens{tokens: []string{testToken(17)}}, orders, locks)

	result, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	if err != nil {
		t.Fatalf("pay_id 竞态后相同字段应重放，实际 err=%v", err)
	}
	if result.Order.PublicToken != token || result.ReallyPrice != "1.01" {
		t.Fatalf("应返回胜者订单，实际 token=%q reallyPrice=%q", result.Order.PublicToken, result.ReallyPrice)
	}
	if result.RedirectURL != "https://pay.example.test/#/payment/"+token {
		t.Fatalf("重放 redirectUrl 错误: %q", result.RedirectURL)
	}
	if len(orders.created) != 0 {
		t.Fatalf("竞态重放不得再插入订单，实际 %d 条", len(orders.created))
	}
	if orders.createCalls != 1 || orders.findCalls != 2 {
		t.Fatalf("应先 miss 再 conflict 再 hit，实际 find=%d create=%d", orders.findCalls, orders.createCalls)
	}
	if locks.liveCount() != 0 {
		t.Fatalf("回滚后不应残留价格锁，held=%v acquired=%v released=%v", locks.held, locks.acquired, locks.released)
	}
	if len(locks.acquired) != 1 || len(locks.released) != 1 {
		t.Fatalf("失败插入占用的锁必须被释放，acquired=%v released=%v", locks.acquired, locks.released)
	}
}

func TestCreateOrderPayIDRaceConflictsWithoutSecondInsert(t *testing.T) {
	token := testToken(18)
	orders := &payIDRaceOrders{
		winner: raceWinnerOrder(token, func(value *order.Order) {
			value.Param = "param-other"
			value.NotifyURL = "https://merchant.test/notify-other"
		}),
		findMisses:      1,
		createConflicts: 1,
	}
	locks := &recordingPriceLocks{}
	service := newCreateOrderServiceWith(t, &scriptedTokens{tokens: []string{testToken(19)}}, orders, locks)

	_, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	code, ok := ErrorCodeOf(err)
	if !ok || code != CodeConflict {
		t.Fatalf("字段冲突应返回 %s，实际 err=%v code=%v", CodeConflict, err, code)
	}
	if len(orders.created) != 0 {
		t.Fatalf("冲突不得插入第二条订单，实际 %d 条", len(orders.created))
	}
	if orders.winner.Param != "param-other" || orders.winner.PublicToken != token {
		t.Fatalf("冲突不得改写胜者订单: %+v", orders.winner)
	}
	if locks.liveCount() != 0 {
		t.Fatalf("冲突路径也不应残留价格锁，held=%v", locks.held)
	}
}

func TestCreateTimeoutRecoveredByQueryByPayID(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := testToken(14)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)

	created, err := service.Create(context.Background(), createOrderInput(testMerchantKey))
	if err != nil {
		t.Fatalf("服务端已接受建单，实际 err=%v", err)
	}

	// 客户端超时后不再持有建单响应，只能凭 payId 恢复 publicToken。
	recovered, err := service.QueryByPayID(context.Background(), queryByPayIDInput(created.Order.PayID, now, testMerchantKey))
	if err != nil {
		t.Fatalf("超时恢复查询应成功，实际 err=%v", err)
	}
	if recovered.PublicToken != created.Order.PublicToken {
		t.Fatalf("恢复到的 publicToken 不一致，建单=%q 查询=%q", created.Order.PublicToken, recovered.PublicToken)
	}
	if recovered.Price != created.Price || recovered.ReallyPrice != created.ReallyPrice {
		t.Fatalf("恢复到的金额不一致，建单 price=%q reallyPrice=%q 查询 price=%q reallyPrice=%q",
			created.Price, created.ReallyPrice, recovered.Price, recovered.ReallyPrice)
	}
	if recovered.Status != order.StatusPending {
		t.Fatalf("超时恢复时订单仍应为待支付，实际状态 %d", recovered.Status)
	}
}

func TestCreateOrderReplayWorksWhenMonitorGoesOffline(t *testing.T) {
	token := testToken(15)
	tokens := &scriptedTokens{tokens: []string{token}}
	orders := &tokenConflictOrders{}
	service := newCreateOrderService(t, tokens, orders)
	input := createOrderInput(testMerchantKey)
	if _, err := service.Create(context.Background(), input); err != nil {
		t.Fatalf("首次建单应成功，实际 err=%v", err)
	}

	settings := service.settings.(*mapSettings)
	settings.values[setting.MonitorStateKey] = "-1"

	replayed, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("监控离线时相同请求仍应重放，实际 err=%v", err)
	}
	if replayed.Order.PublicToken != token {
		t.Fatalf("离线重放应返回原令牌，实际 %q", replayed.Order.PublicToken)
	}
}
