package usecase

import (
	"context"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

const batchExpireSize = 100

// closeExpiredBatch 在已开启的事务内关闭一批过期 pending 订单。
// 它只承担状态机推进与价格锁释放，cutoff 临界点与执行时间 now 由调用方决定，
// 使 OrderService.ExpireOrders 与 LifecycleService.closeExpiredOrders 共享同一套超时状态转换规则。
func closeExpiredBatch(
	txCtx context.Context,
	orders port.OrderRepository,
	priceLocks port.PriceLockRepository,
	cutoff time.Time,
	now time.Time,
) (int, error) {
	var closed int
	expired, err := orders.FindExpiredPendingForUpdate(txCtx, cutoff, batchExpireSize)
	if err != nil {
		return 0, wrap(CodeDependency, "查询过期订单失败", err)
	}
	for _, value := range expired {
		changed, transitionErr := orders.Transition(txCtx, value.ID, order.StatusPending, order.StatusClosed, now)
		if transitionErr != nil {
			return closed, wrap(CodeDependency, "关闭过期订单失败", transitionErr)
		}
		if !changed {
			return closed, fail(CodeConflict, "过期订单状态已变化，请重试")
		}
		if err := priceLocks.ReleaseByOrderID(txCtx, value.OrderID); err != nil {
			return closed, wrap(CodeDependency, "释放过期订单金额失败", err)
		}
		closed++
	}
	return closed, nil
}
