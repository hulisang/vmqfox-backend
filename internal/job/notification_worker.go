package job

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

type NotificationDelivery interface {
	DeliverNext(context.Context, time.Duration) (usecase.DeliveryStats, error)
}

type NotificationWorkerConfig struct {
	PollInterval   time.Duration
	AttemptTimeout time.Duration
	LeaseDuration  time.Duration
	BatchSize      int
}

type NotificationWorker struct {
	delivery NotificationDelivery
	config   NotificationWorkerConfig
	logger   *log.Logger
}

func NewNotificationWorker(delivery NotificationDelivery, config NotificationWorkerConfig, logger *log.Logger) (*NotificationWorker, error) {
	if delivery == nil {
		return nil, errors.New("通知 worker 用例不能为空")
	}
	if config.PollInterval <= 0 || config.AttemptTimeout <= 0 || config.LeaseDuration <= config.AttemptTimeout || config.BatchSize < 1 {
		return nil, errors.New("通知 worker 配置无效")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &NotificationWorker{delivery: delivery, config: config, logger: logger}, nil
}

// Run 串行排空每批任务，避免同一进程内并发领取造成租约和结算顺序失控。
func (w *NotificationWorker) Run(ctx context.Context) {
	w.runBatch(ctx)
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runBatch(ctx)
		}
	}
}

func (w *NotificationWorker) runBatch(ctx context.Context) {
	for index := 0; index < w.config.BatchSize; index++ {
		if ctx.Err() != nil {
			return
		}
		attemptCtx, cancel := context.WithTimeout(ctx, w.config.AttemptTimeout)
		stats, err := w.delivery.DeliverNext(attemptCtx, w.config.LeaseDuration)
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				w.logger.Printf("通知任务处理失败: %v", err)
			}
			return
		}
		if stats.Claimed == 0 {
			return
		}
		if stats.Failed > 0 {
			w.logger.Printf("通知任务本次投递失败，已安排重试")
		}
	}
}
