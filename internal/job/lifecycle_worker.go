package job

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

type LifecycleMaintenance interface {
	Maintain(context.Context, time.Duration, int) (usecase.LifecycleStats, error)
}

type LifecycleWorkerConfig struct {
	PollInterval     time.Duration
	AttemptTimeout   time.Duration
	HeartbeatTimeout time.Duration
	BatchSize        int
}

type LifecycleWorker struct {
	maintenance LifecycleMaintenance
	config      LifecycleWorkerConfig
	logger      *log.Logger
}

func NewLifecycleWorker(maintenance LifecycleMaintenance, config LifecycleWorkerConfig, logger *log.Logger) (*LifecycleWorker, error) {
	if maintenance == nil {
		return nil, errors.New("生命周期 worker 用例不能为空")
	}
	if config.PollInterval <= 0 || config.AttemptTimeout <= 0 || config.HeartbeatTimeout <= 0 || config.BatchSize < 1 {
		return nil, errors.New("生命周期 worker 配置无效")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &LifecycleWorker{maintenance: maintenance, config: config, logger: logger}, nil
}

func (w *LifecycleWorker) Run(ctx context.Context) {
	w.runOnce(ctx)
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *LifecycleWorker) runOnce(ctx context.Context) {
	closedOrders := 0
	for ctx.Err() == nil {
		attemptCtx, cancel := context.WithTimeout(ctx, w.config.AttemptTimeout)
		stats, err := w.maintenance.Maintain(attemptCtx, w.config.HeartbeatTimeout, w.config.BatchSize)
		cancel()
		closedOrders += stats.ClosedOrders
		if err != nil {
			if ctx.Err() == nil {
				w.logger.Printf("生命周期任务处理失败: %v", err)
			}
			return
		}
		if stats.ClosedOrders < w.config.BatchSize {
			break
		}
	}
	if closedOrders > 0 {
		w.logger.Printf("生命周期任务关闭过期订单: %d", closedOrders)
	}
}
