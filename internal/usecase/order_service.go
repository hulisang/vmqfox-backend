package usecase

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
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

const (
	defaultCloseMinutes = 5
	priceLockAttempts   = 10
)

type CreateSignatureMode uint8

const (
	CreateSignatureNew CreateSignatureMode = iota
	CreateSignatureLegacy
)

type CreateOrderInput struct {
	PayID         string
	Param         string
	Type          string
	Price         string
	Sign          string
	NotifyURL     string
	ReturnURL     string
	SignatureMode CreateSignatureMode
}

type CreateOrderResult struct {
	Order          order.Order
	Price          string
	ReallyPrice    string
	TimeoutMinutes int
	RedirectURL    string
}

type OrderView struct {
	Order            order.Order
	StateText        string
	TimeoutMinutes   int
	RemainingSeconds int64
}

type ListOrdersInput struct {
	State *order.Status
	Page  int
	Limit int
}

type OrderPageView struct {
	Items []order.Order
	Total int64
}

type OrderStatisticsView struct {
	TodayOrders       int64
	TodayPaidOrders   int64
	TodayClosedOrders int64
	TodayPaidCents    int64
	TotalOrders       int64
	TotalPaidCents    int64
}

type CheckOrderResult struct {
	State            order.Status
	RedirectURL      string
	RemainingSeconds int64
	ReturnURL        string
	Param            string
	Message          string
}

type ReturnURLResult struct {
	ReturnURL       string
	ReturnURLNew    string
	ReturnURLLegacy string
	Sign            string
	SignLegacy      string
}

type OrderServiceDeps struct {
	Transactions port.TransactionManager
	Orders       port.OrderRepository
	PriceLocks   port.PriceLockRepository
	QRCodes      port.QRCodeRepository
	Settings     port.SettingRepository
	Outbox       port.OutboxRepository
	Clock        port.Clock
	OrderIDs     port.OrderIDGenerator
	FrontendURL  string
}

type OrderService struct {
	transactions port.TransactionManager
	orders       port.OrderRepository
	priceLocks   port.PriceLockRepository
	qrcodes      port.QRCodeRepository
	settings     port.SettingRepository
	outbox       port.OutboxRepository
	clock        port.Clock
	orderIDs     port.OrderIDGenerator
	frontendURL  string
}

func NewOrderService(deps OrderServiceDeps) (*OrderService, error) {
	if deps.Transactions == nil || deps.Orders == nil || deps.PriceLocks == nil || deps.QRCodes == nil ||
		deps.Settings == nil || deps.Outbox == nil || deps.Clock == nil || deps.OrderIDs == nil {
		return nil, errors.New("订单用例依赖不完整")
	}
	return &OrderService{
		transactions: deps.Transactions,
		orders:       deps.Orders,
		priceLocks:   deps.PriceLocks,
		qrcodes:      deps.QRCodes,
		settings:     deps.Settings,
		outbox:       deps.Outbox,
		clock:        deps.Clock,
		orderIDs:     deps.OrderIDs,
		frontendURL:  strings.TrimRight(deps.FrontendURL, "/"),
	}, nil
}

func (s *OrderService) Create(ctx context.Context, input CreateOrderInput) (CreateOrderResult, error) {
	if input.PayID == "" || input.Type == "" || input.Price == "" || input.Sign == "" {
		return CreateOrderResult{}, fail(CodeInvalidArgument, "参数不完整")
	}

	typeNumber, err := strconv.Atoi(input.Type)
	if err != nil || !payment.Type(typeNumber).Valid() {
		return CreateOrderResult{}, fail(CodeInvalidArgument, "支付类型错误")
	}
	paymentType := payment.Type(typeNumber)
	priceCents, err := order.ParseAmountCents(input.Price)
	if err != nil || priceCents <= 0 {
		return CreateOrderResult{}, fail(CodeInvalidArgument, "价格错误")
	}

	settings, err := s.settings.GetMany(ctx, []string{
		setting.MerchantKey,
		setting.MonitorStateKey,
		setting.PriceAdjustKey,
		setting.NotifyURLKey,
		setting.ReturnURLKey,
		setting.WechatPayURLKey,
		setting.AlipayPayURLKey,
		setting.CloseMinutesKey,
	})
	if err != nil {
		return CreateOrderResult{}, wrap(CodeDependency, "读取系统设置失败", err)
	}
	merchantKey := settings[setting.MerchantKey]
	if merchantKey == "" {
		return CreateOrderResult{}, fail(CodeConfiguration, "系统未配置密钥")
	}
	if !validCreateSign(input, merchantKey) {
		return CreateOrderResult{}, fail(CodeInvalidSignature, "签名错误")
	}
	if settings[setting.MonitorStateKey] != "1" {
		return CreateOrderResult{}, fail(CodeMonitorOffline, "监控端状态异常，请检查")
	}

	now := s.clock.Now()
	orderID, err := s.orderIDs.NewOrderID(now)
	if err != nil {
		return CreateOrderResult{}, wrap(CodeDependency, "生成订单号失败", err)
	}

	var created order.Order
	err = s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if _, findErr := s.orders.FindByPayIDForUpdate(txCtx, input.PayID); findErr == nil {
			return fail(CodeDuplicateOrder, "商户订单号已存在，请勿重复提交")
		} else if !errors.Is(findErr, port.ErrNotFound) {
			return wrap(CodeDependency, "检查商户订单号失败", findErr)
		}

		reallyPriceCents, lockErr := s.acquirePrice(txCtx, orderID, paymentType, priceCents, settings[setting.PriceAdjustKey])
		if lockErr != nil {
			return lockErr
		}

		payURL, isAuto, payURLErr := s.resolvePayURL(txCtx, paymentType, reallyPriceCents, settings)
		if payURLErr != nil {
			return payURLErr
		}

		notifyURL := input.NotifyURL
		if notifyURL == "" {
			notifyURL = settings[setting.NotifyURLKey]
		}
		returnURL := input.ReturnURL
		if returnURL == "" {
			returnURL = settings[setting.ReturnURLKey]
		}

		value := order.Order{
			OrderID:          orderID,
			PayID:            input.PayID,
			Type:             paymentType,
			PriceCents:       priceCents,
			ReallyPriceCents: reallyPriceCents,
			PriceText:        input.Price,
			ReallyPriceText:  order.FormatCents(reallyPriceCents),
			State:            order.StatusPending,
			Param:            input.Param,
			PayURL:           payURL,
			IsAuto:           isAuto,
			NotifyURL:        notifyURL,
			ReturnURL:        returnURL,
			CreatedAt:        now,
		}
		created, err = s.orders.Create(txCtx, value)
		if errors.Is(err, port.ErrConflict) {
			return fail(CodeConflict, "创建订单冲突")
		}
		if err != nil {
			return wrap(CodeDependency, "创建订单失败", err)
		}
		return nil
	})
	if err != nil {
		return CreateOrderResult{}, err
	}

	redirectURL := ""
	if s.frontendURL != "" {
		redirectURL = s.frontendURL + "/#/payment/" + created.OrderID
	}
	return CreateOrderResult{
		Order:          created,
		Price:          input.Price,
		ReallyPrice:    order.FormatCents(created.ReallyPriceCents),
		TimeoutMinutes: closeMinutes(settings[setting.CloseMinutesKey]),
		RedirectURL:    redirectURL,
	}, nil
}

func (s *OrderService) Get(ctx context.Context, orderID string) (OrderView, error) {
	if orderID == "" {
		return OrderView{}, fail(CodeInvalidArgument, "订单号不能为空")
	}
	value, err := s.orders.FindByOrderID(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return OrderView{}, fail(CodeNotFound, "订单不存在")
	}
	if err != nil {
		return OrderView{}, wrap(CodeDependency, "查询订单失败", err)
	}
	minutes, err := s.readCloseMinutes(ctx)
	if err != nil {
		return OrderView{}, err
	}
	return orderView(value, minutes, s.clock.Now()), nil
}

func (s *OrderService) List(ctx context.Context, input ListOrdersInput) (OrderPageView, error) {
	if input.State != nil && !input.State.Valid() {
		return OrderPageView{}, fail(CodeInvalidArgument, "订单状态无效")
	}
	page, err := s.orders.List(ctx, port.OrderFilter{
		State: input.State,
		Page:  input.Page,
		Limit: input.Limit,
	})
	if err != nil {
		return OrderPageView{}, wrap(CodeDependency, "查询订单列表失败", err)
	}
	return OrderPageView{Items: page.Items, Total: page.Total}, nil
}

func (s *OrderService) Statistics(ctx context.Context) (OrderStatisticsView, error) {
	now := s.clock.Now()
	year, month, day := now.Date()
	dayStart := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	statistics, err := s.orders.Statistics(ctx, dayStart, dayStart.AddDate(0, 0, 1))
	if err != nil {
		return OrderStatisticsView{}, wrap(CodeDependency, "查询订单统计失败", err)
	}
	return OrderStatisticsView{
		TodayOrders:       statistics.TodayOrders,
		TodayPaidOrders:   statistics.TodayPaidOrders,
		TodayClosedOrders: statistics.TodayClosedOrders,
		TodayPaidCents:    statistics.TodayPaidCents,
		TotalOrders:       statistics.TotalOrders,
		TotalPaidCents:    statistics.TotalPaidCents,
	}, nil
}

func (s *OrderService) Detail(ctx context.Context, id int64) (order.Order, error) {
	if id <= 0 {
		return order.Order{}, fail(CodeInvalidArgument, "订单ID无效")
	}
	value, err := s.orders.FindByID(ctx, id)
	if errors.Is(err, port.ErrNotFound) {
		return order.Order{}, fail(CodeNotFound, "订单不存在")
	}
	if err != nil {
		return order.Order{}, wrap(CodeDependency, "查询订单失败", err)
	}
	return value, nil
}

// Check 只计算对外可见状态，持久化过期状态由生命周期任务统一处理。
func (s *OrderService) Check(ctx context.Context, orderID string) (CheckOrderResult, error) {
	if orderID == "" {
		return CheckOrderResult{}, fail(CodeInvalidArgument, "订单号不能为空")
	}
	minutes, err := s.readCloseMinutes(ctx)
	if err != nil {
		return CheckOrderResult{}, err
	}
	value, err := s.orders.FindByOrderID(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return CheckOrderResult{}, fail(CodeNotFound, "订单不存在")
	}
	if err != nil {
		return CheckOrderResult{}, wrap(CodeDependency, "检查订单状态失败", err)
	}
	return checkResult(value, minutes, s.clock.Now()), nil
}

func (s *OrderService) ReturnURL(ctx context.Context, orderID string) (ReturnURLResult, error) {
	if orderID == "" {
		return ReturnURLResult{}, fail(CodeInvalidArgument, "订单号不能为空")
	}
	value, err := s.orders.FindByOrderID(ctx, orderID)
	if errors.Is(err, port.ErrNotFound) {
		return ReturnURLResult{}, fail(CodeNotFound, "订单不存在")
	}
	if err != nil {
		return ReturnURLResult{}, wrap(CodeDependency, "查询订单失败", err)
	}
	if value.ReturnURL == "" {
		return ReturnURLResult{}, fail(CodeConfiguration, "订单没有配置返回URL")
	}
	merchantKey, err := s.settings.Get(ctx, setting.MerchantKey)
	if errors.Is(err, port.ErrNotFound) || merchantKey == "" {
		return ReturnURLResult{}, fail(CodeConfiguration, "系统未配置密钥")
	}
	if err != nil {
		return ReturnURLResult{}, wrap(CodeDependency, "读取系统密钥失败", err)
	}

	typeText := strconv.Itoa(int(value.Type))
	price := amountText(value.PriceText, value.PriceCents)
	reallyPrice := amountText(value.ReallyPriceText, value.ReallyPriceCents)
	signNew := payment.CallbackSignNew(value.PayID, value.Param, typeText, price, reallyPrice, merchantKey)
	signLegacy := payment.CallbackSignLegacy(value.PayID, value.Param, typeText, price, reallyPrice, merchantKey)
	baseQuery := "payId=" + url.QueryEscape(value.PayID) +
		"&param=" + url.QueryEscape(value.Param) +
		"&type=" + url.QueryEscape(typeText) +
		"&price=" + url.QueryEscape(price) +
		"&reallyPrice=" + url.QueryEscape(reallyPrice)
	newURL := appendReturnQuery(value.ReturnURL, baseQuery+"&sign="+url.QueryEscape(signNew))
	legacyURL := appendReturnQuery(value.ReturnURL, baseQuery+"&sign="+url.QueryEscape(signLegacy))

	return ReturnURLResult{
		ReturnURL:       newURL,
		ReturnURLNew:    newURL,
		ReturnURLLegacy: legacyURL,
		Sign:            signNew,
		SignLegacy:      signLegacy,
	}, nil
}

func appendReturnQuery(baseURL, query string) string {
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return baseURL + separator + query
}

func (s *OrderService) CloseByID(ctx context.Context, id int64) error {
	if id <= 0 {
		return fail(CodeInvalidArgument, "订单ID无效")
	}
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		value, err := s.orders.FindByIDForUpdate(txCtx, id)
		if errors.Is(err, port.ErrNotFound) {
			return fail(CodeNotFound, "订单不存在")
		}
		if err != nil {
			return wrap(CodeDependency, "查询订单失败", err)
		}
		if value.State != order.StatusPending {
			return fail(CodeInvalidState, "只能关闭未支付的订单")
		}
		changed, err := s.orders.Transition(txCtx, value.ID, order.StatusPending, order.StatusClosed, s.clock.Now())
		if err != nil {
			return wrap(CodeDependency, "关闭订单失败", err)
		}
		if !changed {
			return fail(CodeConflict, "订单状态已变化，请重试")
		}
		if err := s.priceLocks.ReleaseByOrderID(txCtx, value.OrderID); err != nil {
			return wrap(CodeDependency, "释放订单金额失败", err)
		}
		return nil
	})
}

func (s *OrderService) acquirePrice(ctx context.Context, orderID string, paymentType payment.Type, baseCents int64, rawAdjustment string) (int64, error) {
	adjustment := order.NoAdjustment
	switch rawAdjustment {
	case "1":
		adjustment = order.Increase
	case "2":
		adjustment = order.Decrease
	}
	for _, candidate := range order.CandidateAmounts(baseCents, adjustment, priceLockAttempts) {
		acquired, err := s.priceLocks.TryAcquire(ctx, orderID, paymentType, candidate)
		if err != nil {
			return 0, wrap(CodeDependency, "占用订单金额失败", err)
		}
		if acquired {
			return candidate, nil
		}
	}
	return 0, fail(CodeOverloaded, "订单超出负荷，请稍后重试")
}

func (s *OrderService) resolvePayURL(ctx context.Context, paymentType payment.Type, amountCents int64, settings map[string]string) (string, bool, error) {
	fixed, err := s.qrcodes.FindEnabledByAmount(ctx, paymentType, amountCents)
	if err == nil && fixed.PayURL != "" {
		return fixed.PayURL, false, nil
	}
	if err != nil && !errors.Is(err, port.ErrNotFound) {
		return "", false, wrap(CodeDependency, "查询支付二维码失败", err)
	}

	key := setting.AlipayPayURLKey
	method := "支付宝"
	if paymentType == payment.Wechat {
		key = setting.WechatPayURLKey
		method = "微信"
	}
	if settings[key] == "" {
		return "", false, fail(CodeConfiguration, "暂无可用支付二维码，请在后台【系统设置】或【"+method+"二维码】中配置")
	}
	return settings[key], true, nil
}

func (s *OrderService) readCloseMinutes(ctx context.Context) (int, error) {
	value, err := s.settings.Get(ctx, setting.CloseMinutesKey)
	if errors.Is(err, port.ErrNotFound) {
		return defaultCloseMinutes, nil
	}
	if err != nil {
		return 0, wrap(CodeDependency, "读取订单超时设置失败", err)
	}
	return closeMinutes(value), nil
}

func validCreateSign(input CreateOrderInput, key string) bool {
	expected := payment.CreateSignNew(input.PayID, input.Param, input.Type, input.Price, key)
	if input.SignatureMode == CreateSignatureLegacy {
		expected = payment.CreateSignLegacy(input.PayID, input.Param, input.Type, input.Price, key)
	}
	return len(expected) == len(input.Sign) && subtle.ConstantTimeCompare([]byte(expected), []byte(input.Sign)) == 1
}

func closeMinutes(value string) int {
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes <= 0 {
		return defaultCloseMinutes
	}
	return minutes
}

func orderView(value order.Order, minutes int, now time.Time) OrderView {
	return OrderView{
		Order:            value,
		StateText:        stateText(value.State),
		TimeoutMinutes:   minutes,
		RemainingSeconds: remainingSeconds(value.CreatedAt, minutes, now),
	}
}

func checkResult(value order.Order, minutes int, now time.Time) CheckOrderResult {
	result := CheckOrderResult{
		State:     value.State,
		ReturnURL: value.ReturnURL,
		Param:     value.Param,
	}
	switch value.State {
	case order.StatusPaid, order.StatusNotifyFailed:
		result.RedirectURL = value.ReturnURL
		result.Message = "支付成功"
	case order.StatusClosed:
		result.RemainingSeconds = 0
		result.Message = "订单已过期"
	default:
		result.RemainingSeconds = remainingSeconds(value.CreatedAt, minutes, now)
		if order.IsExpired(value.State, value.CreatedAt, time.Duration(minutes)*time.Minute, now) {
			result.State = order.StatusClosed
			result.RemainingSeconds = 0
			result.Message = "订单已过期"
		} else {
			result.Message = "订单未支付"
		}
	}
	return result
}

func remainingSeconds(createdAt time.Time, minutes int, now time.Time) int64 {
	remaining := createdAt.Add(time.Duration(minutes)*time.Minute).Unix() - now.Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func stateText(status order.Status) string {
	switch status {
	case order.StatusClosed:
		return "已关闭"
	case order.StatusPending:
		return "未支付"
	case order.StatusPaid:
		return "已支付"
	case order.StatusNotifyFailed:
		return "通知失败"
	default:
		return "未知状态"
	}
}

func (s *OrderService) DeleteByID(ctx context.Context, id int64) error {
	if id <= 0 {
		return fail(CodeInvalidArgument, "订单ID无效")
	}
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		value, err := s.orders.FindByIDForUpdate(txCtx, id)
		if errors.Is(err, port.ErrNotFound) {
			return fail(CodeNotFound, "订单不存在")
		}
		if err != nil {
			return wrap(CodeDependency, "查询订单失败", err)
		}

		if value.State == order.StatusPending {
			if err := s.priceLocks.ReleaseByOrderID(txCtx, value.OrderID); err != nil {
				return wrap(CodeDependency, "释放订单价格锁失败", err)
			}
		}

		_, err = s.orders.Delete(txCtx, id)
		if err != nil {
			return wrap(CodeDependency, "删除订单失败", err)
		}
		return nil
	})
}

func (s *OrderService) DeleteLast(ctx context.Context, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		return 0, fail(CodeInvalidArgument, "清理时间范围无效")
	}
	now := s.clock.Now()
	cutoff := now.Add(-olderThan)

	var count int64
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.priceLocks.ReleaseBefore(txCtx, cutoff); err != nil {
			return wrap(CodeDependency, "批量清理价格锁失败", err)
		}

		c, err := s.orders.DeleteBefore(txCtx, cutoff)
		if err != nil {
			return wrap(CodeDependency, "批量删除订单失败", err)
		}
		count = c
		return nil
	})
	return count, err
}

func (s *OrderService) DeleteExpired(ctx context.Context) (int64, error) {
	now := s.clock.Now()
	var count int64
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		rawCloseMinutes, err := s.settings.Get(txCtx, setting.CloseMinutesKey)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return wrap(CodeDependency, "读取订单超时设置失败", err)
		}
		cutoff := now.Add(-time.Duration(closeMinutes(rawCloseMinutes)) * time.Minute)

		expiredPending, err := s.orders.FindExpiredPendingForUpdate(txCtx, cutoff, 1000)
		if err != nil {
			return wrap(CodeDependency, "查询过期订单失败", err)
		}
		for _, o := range expiredPending {
			if err := s.priceLocks.ReleaseByOrderID(txCtx, o.OrderID); err != nil {
				return wrap(CodeDependency, "级联释放过期订单锁失败", err)
			}
		}

		c, err := s.orders.DeleteExpiredBefore(txCtx, cutoff)
		if err != nil {
			return wrap(CodeDependency, "删除过期订单失败", err)
		}
		count = c
		return nil
	})
	return count, err
}

func (s *OrderService) ExpireOrders(ctx context.Context) (int, error) {
	now := s.clock.Now()
	closed := 0
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		rawCloseMinutes, err := s.settings.Get(txCtx, setting.CloseMinutesKey)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return wrap(CodeDependency, "读取订单超时设置失败", err)
		}
		cutoff := now.Add(-time.Duration(closeMinutes(rawCloseMinutes)) * time.Minute)
		c, err := closeExpiredBatch(txCtx, s.orders, s.priceLocks, cutoff, now)
		closed = c
		return err
	})
	return closed, err
}

func (s *OrderService) ReissueByID(ctx context.Context, id int64) error {
	if id <= 0 {
		return fail(CodeInvalidArgument, "订单ID无效")
	}
	now := s.clock.Now()
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		value, err := s.orders.FindByIDForUpdate(txCtx, id)
		if errors.Is(err, port.ErrNotFound) {
			return fail(CodeNotFound, "订单不存在")
		}
		if err != nil {
			return wrap(CodeDependency, "查询订单失败", err)
		}

		if value.State == order.StatusPaid || value.State == order.StatusNotifyFailed {
			return fail(CodeInvalidState, "订单已付款，请勿重复补单")
		}

		originalState := value.State
		changed, transitionErr := s.orders.Transition(txCtx, value.ID, originalState, order.StatusPaid, now)
		if transitionErr != nil {
			return wrap(CodeDependency, "更新订单支付状态失败", transitionErr)
		}
		if !changed {
			return fail(CodeConflict, "订单状态已变化，请重试")
		}

		if originalState == order.StatusPending {
			if err := s.priceLocks.ReleaseByOrderID(txCtx, value.OrderID); err != nil {
				return wrap(CodeDependency, "释放订单价格锁失败", err)
			}
		}

		value.State = order.StatusPaid
		value.PaidAt = now

		if value.NotifyURL == "" {
			return nil
		}

		key, err := s.settings.Get(txCtx, setting.MerchantKey)
		if errors.Is(err, port.ErrNotFound) || key == "" {
			return fail(CodeConfiguration, "系统密钥未设置")
		}
		if err != nil {
			return wrap(CodeDependency, "读取系统密钥失败", err)
		}

		payload := reissueNotificationPayload(value, key)
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return wrap(CodeDependency, "编码通知事件失败", marshalErr)
		}

		eventKey := reissueEventKey(value.OrderID, now)
		messageID := reissueDigest("notify", eventKey, value.OrderID)

		if enqueueErr := s.outbox.Enqueue(txCtx, port.OutboxMessage{
			ID:            messageID,
			Topic:         notificationTopic,
			AggregateID:   value.OrderID,
			Payload:       encoded,
			EventKey:      eventKey,
			CreatedAt:     now,
			NextAttemptAt: now,
		}); enqueueErr != nil && !errors.Is(enqueueErr, port.ErrAlreadyProcessed) {
			return wrap(CodeDependency, "写入通知事件失败", enqueueErr)
		}
		return nil
	})
}

func reissueNotificationPayload(value order.Order, merchantKey string) NotificationPayload {
	typeText := strconv.Itoa(int(value.Type))
	price := amountText(value.PriceText, value.PriceCents)
	reallyPrice := amountText(value.ReallyPriceText, value.ReallyPriceCents)

	newValues := url.Values{
		"payId":       []string{value.OrderID},
		"param":       []string{value.Param},
		"type":        []string{typeText},
		"price":       []string{price},
		"reallyPrice": []string{reallyPrice},
	}
	newValues.Set("sign", payment.CallbackSignNew(
		value.OrderID,
		value.Param,
		typeText,
		price,
		reallyPrice,
		merchantKey,
	))

	legacyValues := url.Values{
		"payId":       []string{value.PayID},
		"param":       []string{value.Param},
		"type":        []string{typeText},
		"price":       []string{price},
		"reallyPrice": []string{reallyPrice},
	}
	legacyValues.Set("sign", payment.CallbackSignLegacy(
		value.PayID,
		value.Param,
		typeText,
		price,
		reallyPrice,
		merchantKey,
	))

	return NotificationPayload{
		OrderID:     value.OrderID,
		NotifyURL:   value.NotifyURL,
		NewForm:     newValues.Encode(),
		LegacyQuery: legacyValues.Encode(),
	}
}

func reissueEventKey(orderID string, now time.Time) string {
	return reissueDigest("reissue", orderID, now.String())
}

func reissueDigest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
