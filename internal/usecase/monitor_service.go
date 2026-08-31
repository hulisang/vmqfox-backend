package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/setting"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

const notificationTopic = "order.paid.notify"

// defaultMonitorSignTTL 是挂机端签名时间戳的默认允许偏移窗口。
const defaultMonitorSignTTL = 5 * time.Minute

type HeartbeatInput struct {
	Timestamp string
	Sign      string
}

type PaymentPushInput struct {
	Timestamp string
	Type      string
	Price     string
	Sign      string
}

type PaymentPushResult struct {
	Matched   bool
	Duplicate bool
	Order     order.Order
}

type NotificationPayload struct {
	OrderID   string `json:"orderId"`
	NotifyURL string `json:"notifyUrl"`
	NewForm   string `json:"newForm"`
}

type MonitorServiceDeps struct {
	Transactions port.TransactionManager
	Orders       port.OrderRepository
	PriceLocks   port.PriceLockRepository
	Settings     port.SettingRepository
	Events       port.PaymentEventRepository
	Outbox       port.OutboxRepository
	Clock        port.Clock
	PublicTokens port.PublicTokenGenerator
	SignTTL      time.Duration
}

type MonitorService struct {
	transactions port.TransactionManager
	orders       port.OrderRepository
	priceLocks   port.PriceLockRepository
	settings     port.SettingRepository
	events       port.PaymentEventRepository
	outbox       port.OutboxRepository
	clock        port.Clock
	publicTokens port.PublicTokenGenerator
	signTTL      time.Duration
}

func NewMonitorService(deps MonitorServiceDeps) (*MonitorService, error) {
	if deps.Transactions == nil || deps.Orders == nil || deps.PriceLocks == nil || deps.Settings == nil ||
		deps.Events == nil || deps.Outbox == nil || deps.Clock == nil || deps.PublicTokens == nil {
		return nil, errors.New("监控用例依赖不完整")
	}
	signTTL := deps.SignTTL
	if signTTL <= 0 {
		// 配置缺失时回退到默认窗口，避免误配置把重放保护整体关掉。
		signTTL = defaultMonitorSignTTL
	}
	return &MonitorService{
		transactions: deps.Transactions,
		orders:       deps.Orders,
		priceLocks:   deps.PriceLocks,
		settings:     deps.Settings,
		events:       deps.Events,
		outbox:       deps.Outbox,
		clock:        deps.Clock,
		publicTokens: deps.PublicTokens,
		signTTL:      signTTL,
	}, nil
}

func (s *MonitorService) Heartbeat(ctx context.Context, input HeartbeatInput) error {
	if input.Timestamp == "" || input.Sign == "" {
		return fail(CodeInvalidArgument, "缺少必要参数")
	}
	key, err := s.settings.Get(ctx, setting.MerchantKey)
	if errors.Is(err, port.ErrNotFound) || key == "" {
		return fail(CodeConfiguration, "系统密钥未设置")
	}
	if err != nil {
		return wrap(CodeDependency, "读取系统密钥失败", err)
	}
	if !validHeartbeatSign(input, key) {
		if usesLegacyHeartbeatSign(input, key) {
			return fail(CodeDeprecatedSignAlgo, "检测到已废弃的 v1 签名，请升级挂机端 App 至 v2（HMAC-SHA-256）")
		}
		return fail(CodeInvalidSignature, "密钥错误---请检查配置数据！")
	}

	now := s.clock.Now()
	if err := s.checkTimestampFreshness(input.Timestamp, now); err != nil {
		return err
	}
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.settings.SetMany(txCtx, map[string]string{
			setting.LastHeartKey:    strconv.FormatInt(now.Unix(), 10),
			setting.MonitorStateKey: "1",
		}); err != nil {
			return wrap(CodeDependency, "更新监控心跳失败", err)
		}
		return nil
	})
}

func (s *MonitorService) Push(ctx context.Context, input PaymentPushInput) (PaymentPushResult, error) {
	if input.Timestamp == "" || input.Type == "" || input.Price == "" || input.Sign == "" {
		return PaymentPushResult{}, fail(CodeInvalidArgument, "缺少必要参数")
	}
	typeNumber, err := strconv.Atoi(input.Type)
	if err != nil || !payment.Type(typeNumber).Valid() {
		return PaymentPushResult{}, fail(CodeInvalidArgument, "支付类型错误")
	}
	paymentType := payment.Type(typeNumber)
	priceCents, err := order.ParseAmountCents(input.Price)
	if err != nil || priceCents <= 0 {
		return PaymentPushResult{}, fail(CodeInvalidArgument, "价格错误")
	}

	key, err := s.settings.Get(ctx, setting.MerchantKey)
	if errors.Is(err, port.ErrNotFound) || key == "" {
		return PaymentPushResult{}, fail(CodeConfiguration, "系统密钥未设置")
	}
	if err != nil {
		return PaymentPushResult{}, wrap(CodeDependency, "读取系统密钥失败", err)
	}
	if !validPushSign(input, key) {
		if usesLegacyPushSign(input, key) {
			return PaymentPushResult{}, fail(CodeDeprecatedSignAlgo, "检测到已废弃的 v1 签名，请升级挂机端 App 至 v2（HMAC-SHA-256）")
		}
		return PaymentPushResult{}, fail(CodeInvalidSignature, "签名校验不通过")
	}

	now := s.clock.Now()
	if err := s.checkTimestampFreshness(input.Timestamp, now); err != nil {
		return PaymentPushResult{}, err
	}
	eventKey := paymentEventKey(input.Type, input.Price, input.Timestamp)
	result := PaymentPushResult{}
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		isNew, recordErr := s.events.RecordIfNew(txCtx, port.PaymentEvent{
			EventKey:    eventKey,
			PaymentType: paymentType,
			PriceCents:  priceCents,
			EventTime:   input.Timestamp,
			CreatedAt:   now,
		})
		if recordErr != nil {
			return wrap(CodeDependency, "记录到账事件失败", recordErr)
		}
		if err := s.settings.Set(txCtx, setting.LastPayKey, strconv.FormatInt(now.Unix(), 10)); err != nil {
			return wrap(CodeDependency, "更新最后到账时间失败", err)
		}
		if !isNew {
			result.Duplicate = true
			return nil
		}

		matched, findErr := s.orders.FindPendingForUpdate(txCtx, paymentType, priceCents)
		if errors.Is(findErr, port.ErrNotFound) {
			created, createErr := s.createUnmatchedTransfer(txCtx, eventKey, paymentType, priceCents, now.Unix())
			if createErr != nil {
				return createErr
			}
			result.Order = created
			return nil
		}
		if findErr != nil {
			return wrap(CodeDependency, "查询待支付订单失败", findErr)
		}

		changed, transitionErr := s.orders.Transition(txCtx, matched.ID, order.StatusPending, order.StatusPaid, now)
		if transitionErr != nil {
			return wrap(CodeDependency, "更新订单支付状态失败", transitionErr)
		}
		if !changed {
			return fail(CodeConflict, "订单状态已变化，请重试")
		}
		if err := s.priceLocks.ReleaseByOrderID(txCtx, matched.OrderID); err != nil {
			return wrap(CodeDependency, "释放订单金额失败", err)
		}

		matched.State = order.StatusPaid
		matched.PaidAt = now
		result.Matched = true
		result.Order = matched
		if matched.NotifyURL == "" {
			return nil
		}

		payload := signedNotificationPayload(matched, key)
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return wrap(CodeDependency, "编码通知事件失败", marshalErr)
		}
		messageID := digest("notify", eventKey, matched.OrderID)
		if enqueueErr := s.outbox.Enqueue(txCtx, port.OutboxMessage{
			ID:            messageID,
			Topic:         notificationTopic,
			AggregateID:   matched.OrderID,
			Payload:       encoded,
			EventKey:      eventKey,
			CreatedAt:     now,
			NextAttemptAt: now,
		}); enqueueErr != nil && !errors.Is(enqueueErr, port.ErrAlreadyProcessed) {
			return wrap(CodeDependency, "写入通知事件失败", enqueueErr)
		}
		return nil
	})
	if err != nil {
		return PaymentPushResult{}, err
	}
	return result, nil
}

// createUnmatchedTransfer 让无订单转账也遵循公开令牌唯一性约束，并在极低概率冲突时重试。
func (s *MonitorService) createUnmatchedTransfer(ctx context.Context, eventKey string, paymentType payment.Type, priceCents, unixSecond int64) (order.Order, error) {
	for attempt := 0; attempt < publicTokenCreateRetryMax; attempt++ {
		publicToken, tokenErr := newPublicToken(s.publicTokens)
		if tokenErr != nil {
			return order.Order{}, wrap(CodeDependency, "生成无订单转账公开令牌失败", tokenErr)
		}
		created, createErr := s.orders.Create(ctx, unmatchedTransfer(eventKey, publicToken, paymentType, priceCents, unixSecond))
		if errors.Is(createErr, port.ErrPublicTokenConflict) {
			continue
		}
		if createErr != nil {
			return order.Order{}, wrap(CodeDependency, "记录无订单转账失败", createErr)
		}
		return created, nil
	}
	return order.Order{}, fail(CodeDependency, "生成唯一无订单转账公开令牌失败")
}

func signedNotificationPayload(value order.Order, merchantKey string) NotificationPayload {
	typeText := strconv.Itoa(int(value.Type))
	price := amountText(value.PriceText, value.PriceCents)
	reallyPrice := amountText(value.ReallyPriceText, value.ReallyPriceCents)

	newValues := url.Values{
		"payId":       []string{value.PayID},
		"param":       []string{value.Param},
		"type":        []string{typeText},
		"price":       []string{price},
		"reallyPrice": []string{reallyPrice},
	}
	newValues.Set("sign", payment.CallbackSignV2(
		value.PayID,
		value.Param,
		typeText,
		price,
		reallyPrice,
		merchantKey,
	))

	return NotificationPayload{
		OrderID:   value.OrderID,
		NotifyURL: value.NotifyURL,
		NewForm:   newValues.Encode(),
	}
}

func validHeartbeatSign(input HeartbeatInput, key string) bool {
	return payment.SignEqual(input.Sign, payment.HeartbeatSignV2(input.Timestamp, key))
}

func validPushSign(input PaymentPushInput, key string) bool {
	return payment.SignEqual(input.Sign, payment.PushSignV2(input.Type, input.Price, input.Timestamp, key))
}

// 下面两个 legacy 判定只用于区分「挂机端未升级」与「密钥配错」，不构成任何放行路径。
func usesLegacyHeartbeatSign(input HeartbeatInput, key string) bool {
	return payment.SignEqual(input.Sign, payment.LegacyHeartbeatSign(input.Timestamp, key))
}

func usesLegacyPushSign(input PaymentPushInput, key string) bool {
	return payment.SignEqual(input.Sign, payment.LegacyPushSign(input.Type, input.Price, input.Timestamp, key))
}

// parseMonitorTimestamp 解析挂机端时间戳；挂机端发送的是毫秒（Java new Date().getTime()）。
func parseMonitorTimestamp(raw string) (time.Time, bool) {
	millis, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || millis <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(millis), true
}

// checkTimestampFreshness 双向限制时钟偏移，使被截获的心跳或推送无法在窗口外重放。
func (s *MonitorService) checkTimestampFreshness(raw string, now time.Time) error {
	moment, ok := parseMonitorTimestamp(raw)
	if !ok {
		return fail(CodeInvalidArgument, "时间戳格式错误")
	}
	skew := now.Sub(moment)
	if skew < 0 {
		skew = -skew
	}
	if skew > s.signTTL {
		return fail(CodeStaleTimestamp, "请求时间戳超出允许窗口，请校准挂机端设备时间")
	}
	return nil
}

func paymentEventKey(paymentType, price, timestamp string) string {
	return digest("monitor-push", paymentType, price, timestamp)
}

func digest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func unmatchedTransfer(eventKey, publicToken string, paymentType payment.Type, priceCents, unixSecond int64) order.Order {
	identifier := "无订单转账-" + strconv.FormatInt(unixSecond, 10) + "-" + eventKey[:8]
	price := order.FormatCents(priceCents)
	return order.Order{
		OrderID:          identifier,
		PublicToken:      publicToken,
		PayID:            identifier,
		Type:             paymentType,
		PriceCents:       priceCents,
		ReallyPriceCents: priceCents,
		PriceText:        price,
		ReallyPriceText:  price,
		State:            order.StatusPaid,
		Param:            "无订单转账",
		CreatedAt:        unixTimeValue(unixSecond),
		PaidAt:           unixTimeValue(unixSecond),
	}
}

func amountText(value string, cents int64) string {
	if value != "" {
		return value
	}
	return order.FormatCents(cents)
}

func unixTimeValue(value int64) time.Time {
	return time.Unix(value, 0)
}
