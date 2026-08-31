package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

// 本文件集中存放 usecase 包的测试替身，避免每个测试文件各自重复实现宽接口。
// 需要特定行为的用例应嵌入这里的基础替身，只覆写自己关心的方法。

// fakeClock 固定时间，使断言不受真实时钟影响。
type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

// passthroughTransactions 直接执行回调，把测试聚焦在业务逻辑而非事务语义。
type passthroughTransactions struct{ calls int }

func (t *passthroughTransactions) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.calls++
	return fn(ctx)
}

// mapSettings 用内存映射模拟系统设置读写。
type mapSettings struct{ values map[string]string }

func newMapSettings(values map[string]string) *mapSettings {
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return &mapSettings{values: copied}
}

func (s *mapSettings) Get(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", port.ErrNotFound
	}
	return value, nil
}

func (s *mapSettings) GetMany(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}

func (s *mapSettings) GetManyForUpdate(ctx context.Context, keys []string) (map[string]string, error) {
	return s.GetMany(ctx, keys)
}

func (s *mapSettings) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *mapSettings) SetMany(_ context.Context, values map[string]string) error {
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

// scriptedTokens 按脚本返回公开令牌，用于覆盖冲突重试、格式非法与随机源故障分支。
type scriptedTokens struct {
	tokens []string
	err    error
	calls  int
}

func (g *scriptedTokens) NewPublicToken() (string, error) {
	g.calls++
	if g.err != nil {
		return "", g.err
	}
	if len(g.tokens) == 0 {
		return "", errors.New("测试令牌脚本已耗尽")
	}
	token := g.tokens[0]
	g.tokens = g.tokens[1:]
	return token, nil
}

// fakeOrderIDs 生成稳定订单号，确保断言不依赖时间格式。
type fakeOrderIDs struct{ value string }

func (g fakeOrderIDs) NewOrderID(time.Time) (string, error) { return g.value, nil }

// emptyQRCodes 让创建路径回退到系统设置中的收款地址。
type emptyQRCodes struct{}

func (emptyQRCodes) FindEnabledByAmount(context.Context, payment.Type, int64) (qrcode.QRCode, error) {
	return qrcode.QRCode{}, port.ErrNotFound
}

func (emptyQRCodes) FindByID(context.Context, int64) (qrcode.QRCode, error) {
	return qrcode.QRCode{}, port.ErrNotFound
}

func (emptyQRCodes) List(context.Context, port.QRCodeFilter) (port.QRCodePage, error) {
	return port.QRCodePage{}, nil
}

func (emptyQRCodes) Create(_ context.Context, value qrcode.QRCode) (qrcode.QRCode, error) {
	return value, nil
}

func (emptyQRCodes) SetState(context.Context, int64, qrcode.State, qrcode.State) (bool, error) {
	return false, nil
}

func (emptyQRCodes) Delete(context.Context, int64) (bool, error) { return false, nil }

// grantingPriceLocks 总是成功占用金额，把测试聚焦在公开令牌上。
type grantingPriceLocks struct{}

func (grantingPriceLocks) TryAcquire(context.Context, string, payment.Type, int64) (bool, error) {
	return true, nil
}

func (grantingPriceLocks) ReleaseByOrderID(context.Context, string) error { return nil }

func (grantingPriceLocks) ReleaseBefore(context.Context, time.Time) error { return nil }

// recordingOutbox 记录入队消息，供需要断言通知副作用的用例使用。
type recordingOutbox struct{ messages []port.OutboxMessage }

func (o *recordingOutbox) Enqueue(_ context.Context, message port.OutboxMessage) error {
	o.messages = append(o.messages, message)
	return nil
}

func (o *recordingOutbox) ClaimPending(context.Context, time.Time, int, time.Duration) ([]port.OutboxMessage, error) {
	return nil, nil
}

func (o *recordingOutbox) MarkDelivered(context.Context, string, string, time.Time) error { return nil }

func (o *recordingOutbox) MarkFailed(context.Context, string, string, int, time.Time, string) error {
	return nil
}

// newEventsRepository 让每次推送都被视为新到账事件。
type newEventsRepository struct{}

func (newEventsRepository) RecordIfNew(context.Context, port.PaymentEvent) (bool, error) {
	return true, nil
}

// stubOrderRepository 为 16 方法的宽接口 port.OrderRepository 提供空实现基座。
// 测试通过嵌入它并覆写少数方法来表达意图，不必为无关方法重复写样板。
type stubOrderRepository struct{}

func (stubOrderRepository) FindByID(context.Context, int64) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindByIDForUpdate(context.Context, int64) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindByOrderID(context.Context, string) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindByOrderIDForUpdate(context.Context, string) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindByPublicToken(context.Context, string) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindByPayID(context.Context, string) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindByPayIDForUpdate(context.Context, string) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) List(context.Context, port.OrderFilter) (port.OrderPage, error) {
	return port.OrderPage{}, nil
}

func (stubOrderRepository) Statistics(context.Context, time.Time, time.Time) (port.OrderStatistics, error) {
	return port.OrderStatistics{}, nil
}

func (stubOrderRepository) Create(_ context.Context, value order.Order) (order.Order, error) {
	return value, nil
}

func (stubOrderRepository) FindPendingForUpdate(context.Context, payment.Type, int64) (order.Order, error) {
	return order.Order{}, port.ErrNotFound
}

func (stubOrderRepository) FindExpiredPendingForUpdate(context.Context, time.Time, int) ([]order.Order, error) {
	return nil, nil
}

func (stubOrderRepository) Transition(context.Context, int64, order.Status, order.Status, time.Time) (bool, error) {
	return false, nil
}

func (stubOrderRepository) Delete(context.Context, int64) (bool, error) { return false, nil }

func (stubOrderRepository) DeleteBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func (stubOrderRepository) DeleteExpiredBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

// 集中校验替身仍满足端口契约，端口新增方法时这里会第一时间编译失败。
var (
	_ port.OrderRepository        = stubOrderRepository{}
	_ port.QRCodeRepository       = emptyQRCodes{}
	_ port.PriceLockRepository    = grantingPriceLocks{}
	_ port.OutboxRepository       = (*recordingOutbox)(nil)
	_ port.PaymentEventRepository = newEventsRepository{}
	_ port.SettingRepository      = (*mapSettings)(nil)
	_ port.PublicTokenGenerator   = (*scriptedTokens)(nil)
	_ port.OrderIDGenerator       = fakeOrderIDs{}
	_ port.Clock                  = fakeClock{}
	_ port.TransactionManager     = (*passthroughTransactions)(nil)
)

// testToken 生成固定长度的小写十六进制令牌，便于断言而不依赖随机性。
func testToken(seed byte) string {
	value := make([]byte, order.PublicTokenLength)
	for index := range value {
		value[index] = "0123456789abcdef"[(int(seed)+index)%16]
	}
	return string(value)
}

func repeatedTokens(token string, count int) []string {
	result := make([]string, 0, count)
	for range count {
		result = append(result, token)
	}
	return result
}
