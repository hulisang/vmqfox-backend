package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/adapter/httpclient"
	"github.com/hulisang/vmqfox-backend/internal/adapter/mysql"
	"github.com/hulisang/vmqfox-backend/internal/adapter/qrcodeimage"
	"github.com/hulisang/vmqfox-backend/internal/adapter/system"
	"github.com/hulisang/vmqfox-backend/internal/auth"
	"github.com/hulisang/vmqfox-backend/internal/config"
	transport "github.com/hulisang/vmqfox-backend/internal/http"
	"github.com/hulisang/vmqfox-backend/internal/http/handler"
	"github.com/hulisang/vmqfox-backend/internal/http/middleware"
	"github.com/hulisang/vmqfox-backend/internal/job"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
	"gorm.io/gorm"
)

type backgroundJob interface {
	Run(context.Context)
}

type App struct {
	server     *http.Server
	db         *gorm.DB
	jobs       []backgroundJob
	jobsCancel context.CancelFunc
	jobsWG     sync.WaitGroup
	start      sync.Once
	close      sync.Once
	closeErr   error
}

func New(cfg config.Config, db *gorm.DB, tokenParser *auth.TokenService, logger *log.Logger) (*App, error) {
	if db == nil {
		return nil, errors.New("数据库连接不能为空")
	}
	if tokenParser == nil {
		return nil, errors.New("Token 服务不能为空")
	}
	if logger == nil {
		logger = log.Default()
	}

	if cfg.Server.Mode != "" {
		gin.SetMode(cfg.Server.Mode)
	}
	status := newStatusProvider(db, cfg)
	transactions := mysql.NewTransactionManager(db)
	credentials := mysql.NewAdminCredentialRepository(db)
	settings := mysql.NewSettingRepository(db)
	orders := mysql.NewOrderRepository(db)
	priceLocks := mysql.NewPriceLockRepository(db)
	qrcodes := mysql.NewQRCodeRepository(db)
	paymentEvents := mysql.NewPaymentEventRepository(db)
	outbox := mysql.NewOutboxRepository(db)
	clock := system.Clock{}
	publicTokens := system.PublicTokenGenerator{}
	passwords := auth.NewBcryptPasswordHasher()
	qrImageCodec := qrcodeimage.New()

	authService, err := usecase.NewAuthService(usecase.AuthServiceDeps{
		Credentials: credentials,
		Passwords:   passwords,
		Tokens:      tokenParser,
	})
	if err != nil {
		return nil, err
	}
	settingsService, err := usecase.NewSettingsService(usecase.SettingsServiceDeps{
		Transactions: transactions,
		Credentials:  credentials,
		Passwords:    passwords,
		Settings:     settings,
		Clock:        clock,
	})
	if err != nil {
		return nil, err
	}
	qrImageService, err := usecase.NewQRCodeImageService(usecase.QRCodeImageServiceDeps{
		Codec: qrImageCodec,
	})
	if err != nil {
		return nil, err
	}
	qrCodeService, err := usecase.NewQRCodeService(usecase.QRCodeServiceDeps{
		QRCodes: qrcodes,
	})
	if err != nil {
		return nil, err
	}
	orderService, err := usecase.NewOrderService(usecase.OrderServiceDeps{
		Transactions: transactions,
		Orders:       orders,
		PriceLocks:   priceLocks,
		QRCodes:      qrcodes,
		Settings:     settings,
		Outbox:       outbox,
		Clock:        clock,
		OrderIDs:     system.OrderIDGenerator{},
		PublicTokens: publicTokens,
		FrontendURL:  cfg.Server.FrontendURL,
	})
	if err != nil {
		return nil, err
	}
	monitorService, err := usecase.NewMonitorService(usecase.MonitorServiceDeps{
		Transactions: transactions,
		Orders:       orders,
		PriceLocks:   priceLocks,
		Settings:     settings,
		Events:       paymentEvents,
		Outbox:       outbox,
		Clock:        clock,
		PublicTokens: publicTokens,
		SignTTL:      cfg.Jobs.MonitorSignTTL,
	})
	if err != nil {
		return nil, err
	}

	backgroundJobs := make([]backgroundJob, 0, 2)
	if cfg.Runtime.CanWrite() {
		notificationService, serviceErr := usecase.NewNotificationService(usecase.NotificationServiceDeps{
			Transactions: transactions,
			Orders:       orders,
			Outbox:       outbox,
			Notifier:     httpclient.NewNotifier(nil, cfg.Outbound),
			Clock:        clock,
		})
		if serviceErr != nil {
			return nil, serviceErr
		}
		notificationWorker, workerErr := job.NewNotificationWorker(notificationService, job.NotificationWorkerConfig{
			PollInterval:   cfg.Jobs.NotificationPollInterval,
			AttemptTimeout: cfg.Jobs.NotificationAttemptTimeout,
			LeaseDuration:  cfg.Jobs.NotificationLeaseDuration,
			BatchSize:      cfg.Jobs.NotificationBatchSize,
		}, logger)
		if workerErr != nil {
			return nil, workerErr
		}

		lifecycleService, serviceErr := usecase.NewLifecycleService(usecase.LifecycleServiceDeps{
			Transactions: transactions,
			Orders:       orders,
			PriceLocks:   priceLocks,
			Settings:     settings,
			Clock:        clock,
		})
		if serviceErr != nil {
			return nil, serviceErr
		}
		lifecycleWorker, workerErr := job.NewLifecycleWorker(lifecycleService, job.LifecycleWorkerConfig{
			PollInterval:     cfg.Jobs.LifecyclePollInterval,
			AttemptTimeout:   cfg.Jobs.LifecycleAttemptTimeout,
			HeartbeatTimeout: cfg.Jobs.MonitorHeartbeatTimeout,
			BatchSize:        cfg.Jobs.LifecycleBatchSize,
		}, logger)
		if workerErr != nil {
			return nil, workerErr
		}
		backgroundJobs = append(backgroundJobs, notificationWorker, lifecycleWorker)
	}

	handlers := handler.New(handler.Dependencies{
		Auth:         authService,
		Settings:     settingsService,
		Orders:       orderService,
		Monitor:      monitorService,
		QRCodes:      qrCodeService,
		QRCodeImages: qrImageService,
		TokenParser:  tokenParser,
		Status:       status,
	})
	engine := transport.NewRouter(transport.RouterDeps{
		Handlers:    handlers,
		TokenParser: tokenParser,
		Origin:      cfg.Server.AllowedOrigin,
		RuntimeMode: middleware.RuntimeMode(cfg.Runtime.Mode),
		RateLimit:   cfg.RateLimit,
		Logger:      logger,
	})

	return &App{
		db:   db,
		jobs: backgroundJobs,
		server: &http.Server{
			Addr:              cfg.Server.Address(),
			Handler:           engine,
			ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
			ReadTimeout:       cfg.Server.ReadTimeout,
			WriteTimeout:      cfg.Server.WriteTimeout,
			IdleTimeout:       cfg.Server.IdleTimeout,
		},
	}, nil
}

func (a *App) Start(ctx context.Context) {
	a.start.Do(func() {
		if ctx.Err() != nil || len(a.jobs) == 0 {
			return
		}
		jobContext, cancel := context.WithCancel(ctx)
		a.jobsCancel = cancel
		for _, job := range a.jobs {
			a.jobsWG.Add(1)
			go func(j backgroundJob) {
				defer a.jobsWG.Done()
				j.Run(jobContext)
			}(job)
		}
	})
}

func (a *App) Server() *http.Server { return a.server }

func (a *App) Close(ctx context.Context) error {
	a.close.Do(func() {
		if a.jobsCancel != nil {
			a.jobsCancel()
		}
		if a.server != nil {
			if shutdownErr := a.server.Shutdown(ctx); shutdownErr != nil && a.closeErr == nil {
				a.closeErr = shutdownErr
			}
		}
		jobsDone := make(chan struct{})
		go func() {
			a.jobsWG.Wait()
			close(jobsDone)
		}()
		select {
		case <-jobsDone:
		case <-ctx.Done():
			if a.closeErr == nil {
				a.closeErr = ctx.Err()
			}
		}
		if a.db != nil {
			sqlDB, dbErr := a.db.DB()
			if dbErr != nil && a.closeErr == nil {
				a.closeErr = dbErr
			}
			if sqlDB != nil {
				if dbErr = sqlDB.Close(); dbErr != nil && a.closeErr == nil {
					a.closeErr = dbErr
				}
			}
		}
	})
	return a.closeErr
}
