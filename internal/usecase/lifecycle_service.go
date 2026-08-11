package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/setting"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

type LifecycleServiceDeps struct {
	Transactions port.TransactionManager
	Orders       port.OrderRepository
	PriceLocks   port.PriceLockRepository
	Settings     port.SettingRepository
	Clock        port.Clock
}

type LifecycleStats struct {
	ClosedOrders int
	MonitorState string
}

// LifecycleService 集中处理时间驱动的状态变化，避免查询请求产生写库副作用。
type LifecycleService struct {
	transactions port.TransactionManager
	orders       port.OrderRepository
	priceLocks   port.PriceLockRepository
	settings     port.SettingRepository
	clock        port.Clock
}

func NewLifecycleService(deps LifecycleServiceDeps) (*LifecycleService, error) {
	if deps.Transactions == nil || deps.Orders == nil || deps.PriceLocks == nil || deps.Settings == nil || deps.Clock == nil {
		return nil, errors.New("生命周期用例依赖不完整")
	}
	return &LifecycleService{
		transactions: deps.Transactions,
		orders:       deps.Orders,
		priceLocks:   deps.PriceLocks,
		settings:     deps.Settings,
		clock:        deps.Clock,
	}, nil
}

func (s *LifecycleService) Maintain(ctx context.Context, heartbeatTimeout time.Duration, batchSize int) (LifecycleStats, error) {
	if heartbeatTimeout <= 0 || batchSize < 1 {
		return LifecycleStats{}, fail(CodeConfiguration, "生命周期任务配置无效")
	}

	now := s.clock.Now()
	monitorState, monitorErr := s.refreshMonitorState(ctx, now, heartbeatTimeout)
	closedOrders, expiryErr := s.closeExpiredOrders(ctx, now, batchSize)
	return LifecycleStats{
		ClosedOrders: closedOrders,
		MonitorState: monitorState,
	}, errors.Join(monitorErr, expiryErr)
}

func (s *LifecycleService) refreshMonitorState(ctx context.Context, now time.Time, heartbeatTimeout time.Duration) (string, error) {
	state := ""
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		values, err := s.settings.GetManyForUpdate(txCtx, []string{
			setting.LastHeartKey,
			setting.MonitorStateKey,
		})
		if err != nil {
			return wrap(CodeDependency, "读取监控状态失败", err)
		}

		state = expectedMonitorState(values[setting.LastHeartKey], now, heartbeatTimeout)
		if values[setting.MonitorStateKey] == state {
			return nil
		}
		if err := s.settings.Set(txCtx, setting.MonitorStateKey, state); err != nil {
			return wrap(CodeDependency, "更新监控状态失败", err)
		}
		return nil
	})
	return state, err
}

func (s *LifecycleService) closeExpiredOrders(ctx context.Context, now time.Time, batchSize int) (int, error) {
	closed := 0
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		rawCloseMinutes, err := s.settings.Get(txCtx, setting.CloseMinutesKey)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return wrap(CodeDependency, "读取订单超时设置失败", err)
		}
		cutoff := now.Add(-time.Duration(closeMinutes(rawCloseMinutes)) * time.Minute)

		expired, err := s.orders.FindExpiredPendingForUpdate(txCtx, cutoff, batchSize)
		if err != nil {
			return wrap(CodeDependency, "查询过期订单失败", err)
		}
		for _, value := range expired {
			changed, transitionErr := s.orders.Transition(txCtx, value.ID, order.StatusPending, order.StatusClosed, now)
			if transitionErr != nil {
				return wrap(CodeDependency, "关闭过期订单失败", transitionErr)
			}
			if !changed {
				return fail(CodeConflict, "过期订单状态已变化，请重试")
			}
			if err := s.priceLocks.ReleaseByOrderID(txCtx, value.OrderID); err != nil {
				return wrap(CodeDependency, "释放过期订单金额失败", err)
			}
			closed++
		}
		return nil
	})
	return closed, err
}

func expectedMonitorState(rawLastHeart string, now time.Time, heartbeatTimeout time.Duration) string {
	lastHeart, err := strconv.ParseInt(strings.TrimSpace(rawLastHeart), 10, 64)
	if err != nil || lastHeart <= 0 {
		return "-1"
	}
	if time.Unix(lastHeart, 0).Add(heartbeatTimeout).After(now) {
		return "1"
	}
	return "0"
}
