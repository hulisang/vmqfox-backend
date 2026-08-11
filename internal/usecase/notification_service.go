package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

type NotificationServiceDeps struct {
	Transactions port.TransactionManager
	Orders       port.OrderRepository
	Outbox       port.OutboxRepository
	Notifier     port.Notifier
	Clock        port.Clock
}

type NotificationService struct {
	transactions port.TransactionManager
	orders       port.OrderRepository
	outbox       port.OutboxRepository
	notifier     port.Notifier
	clock        port.Clock
}

type DeliveryStats struct {
	Claimed   int
	Delivered int
	Failed    int
}

func NewNotificationService(deps NotificationServiceDeps) (*NotificationService, error) {
	if deps.Transactions == nil || deps.Orders == nil || deps.Outbox == nil || deps.Notifier == nil || deps.Clock == nil {
		return nil, errors.New("通知用例依赖不完整")
	}
	return &NotificationService{
		transactions: deps.Transactions,
		orders:       deps.Orders,
		outbox:       deps.Outbox,
		notifier:     deps.Notifier,
		clock:        deps.Clock,
	}, nil
}

func (s *NotificationService) DeliverNext(ctx context.Context, leaseDuration time.Duration) (DeliveryStats, error) {
	if leaseDuration <= 0 {
		return DeliveryStats{}, fail(CodeConfiguration, "通知任务租约时长无效")
	}
	now := s.clock.Now()
	messages, err := s.outbox.ClaimPending(ctx, now, 1, leaseDuration)
	if err != nil {
		return DeliveryStats{}, wrap(CodeDependency, "领取通知任务失败", err)
	}

	stats := DeliveryStats{Claimed: len(messages)}
	for _, message := range messages {
		delivered, reason := s.deliver(ctx, message)
		if delivered {
			if err := s.settleDelivered(ctx, message, s.clock.Now()); err != nil {
				return stats, err
			}
			stats.Delivered++
			continue
		}
		if err := s.settleFailed(ctx, message, reason, s.clock.Now()); err != nil {
			return stats, err
		}
		stats.Failed++
	}
	return stats, nil
}

func (s *NotificationService) deliver(ctx context.Context, message port.OutboxMessage) (bool, string) {
	if message.Topic != notificationTopic {
		return false, "不支持的通知主题: " + message.Topic
	}
	var payload NotificationPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return false, "通知载荷无效: " + err.Error()
	}
	if payload.NotifyURL == "" {
		return true, ""
	}
	if payload.NewForm == "" || payload.LegacyQuery == "" {
		return false, "通知载荷缺少预签名请求参数"
	}

	newResult, newErr := s.notifier.Send(ctx, port.Notification{
		OrderID: payload.OrderID,
		URL:     payload.NotifyURL,
		Method:  http.MethodPost,
		Body:    []byte(payload.NewForm),
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
	})
	if notificationSucceeded(newResult, newErr) {
		return true, ""
	}

	legacyResult, legacyErr := s.notifier.Send(ctx, port.Notification{
		OrderID: payload.OrderID,
		URL:     appendQuery(payload.NotifyURL, payload.LegacyQuery),
		Method:  http.MethodGet,
	})
	if notificationSucceeded(legacyResult, legacyErr) {
		return true, ""
	}
	return false, notificationFailure(newResult, newErr, legacyResult, legacyErr)
}

func (s *NotificationService) settleDelivered(ctx context.Context, message port.OutboxMessage, now time.Time) error {
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		value, err := s.orders.FindByOrderIDForUpdate(txCtx, message.AggregateID)
		if err != nil {
			return wrap(CodeDependency, "查询通知订单失败", err)
		}
		if value.State == order.StatusNotifyFailed {
			changed, transitionErr := s.orders.Transition(txCtx, value.ID, order.StatusNotifyFailed, order.StatusPaid, now)
			if transitionErr != nil {
				return wrap(CodeDependency, "恢复订单通知状态失败", transitionErr)
			}
			if !changed {
				return fail(CodeConflict, "订单状态已变化，请重试")
			}
		} else if value.State != order.StatusPaid {
			return fail(CodeInvalidState, "订单不处于已支付状态")
		}
		if err := s.outbox.MarkDelivered(txCtx, message.ID, message.LeaseToken, now); err != nil {
			return wrap(CodeDependency, "确认通知任务失败", err)
		}
		return nil
	})
}

func (s *NotificationService) settleFailed(ctx context.Context, message port.OutboxMessage, reason string, now time.Time) error {
	attempts := message.Attempts + 1
	nextAttemptAt := now.Add(notificationBackoff(attempts))
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		value, err := s.orders.FindByOrderIDForUpdate(txCtx, message.AggregateID)
		if err != nil {
			return wrap(CodeDependency, "查询通知订单失败", err)
		}
		if value.State == order.StatusPaid {
			changed, transitionErr := s.orders.Transition(txCtx, value.ID, order.StatusPaid, order.StatusNotifyFailed, now)
			if transitionErr != nil {
				return wrap(CodeDependency, "标记订单通知失败异常", transitionErr)
			}
			if !changed {
				return fail(CodeConflict, "订单状态已变化，请重试")
			}
		} else if value.State != order.StatusNotifyFailed {
			return fail(CodeInvalidState, "订单不处于可通知状态")
		}
		if err := s.outbox.MarkFailed(txCtx, message.ID, message.LeaseToken, attempts, nextAttemptAt, reason); err != nil {
			return wrap(CodeDependency, "记录通知失败结果异常", err)
		}
		return nil
	})
}

func notificationSucceeded(result port.NotificationResult, err error) bool {
	// 只接受成功状态和字节级精确正文，3xx 及错误状态必须进入历史 GET 回退。
	return err == nil && result.StatusCode >= http.StatusOK && result.StatusCode < http.StatusMultipleChoices && string(result.Body) == "success"
}

func notificationFailure(newResult port.NotificationResult, newErr error, legacyResult port.NotificationResult, legacyErr error) string {
	return fmt.Sprintf(
		"新版POST(status=%d,error=%v,body=%q); 历史GET(status=%d,error=%v,body=%q)",
		newResult.StatusCode,
		newErr,
		truncateBody(newResult.Body),
		legacyResult.StatusCode,
		legacyErr,
		truncateBody(legacyResult.Body),
	)
}

func appendQuery(baseURL, query string) string {
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return baseURL + separator + query
}

func notificationBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<(attempts-1)) * time.Minute
}

func truncateBody(body []byte) string {
	const limit = 200
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit])
}
