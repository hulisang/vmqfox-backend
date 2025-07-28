package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vmqfox-api-go/internal/config"
	"vmqfox-api-go/internal/handler"
	"vmqfox-api-go/internal/middleware"
	"vmqfox-api-go/internal/repository"
	"vmqfox-api-go/internal/scheduler"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/jwt"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	if err := config.LoadConfig("."); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置Gin模式
	gin.SetMode(config.AppConfig.Server.Mode)

	// 初始化数据库
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 跳过自动迁移，因为表已经存在且有外键约束
	// 如果需要迁移，请手动执行SQL
	log.Println("Skipping auto migration - using existing database schema")

	// 初始化JWT管理器
	jwtManager := jwt.NewJWTManager(&config.AppConfig.JWT)

	// 初始化仓库层
	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	qrcodeRepo := repository.NewQrcodeRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	merchantRepo := repository.NewMerchantRepository(db)

	// 初始化服务层
	userService := service.NewUserService(userRepo, settingRepo)
	authService := service.NewAuthService(userRepo, settingRepo, merchantRepo, jwtManager)
	orderService := service.NewOrderService(orderRepo, userRepo, settingRepo)
	qrcodeService := service.NewQrcodeService(qrcodeRepo)
	settingService := service.NewSettingService(settingRepo, orderRepo, merchantRepo)
	paymentService := service.NewPaymentService(orderRepo, settingRepo, qrcodeService)

	// 设置双向同步：setting服务需要能够同步到user服务
	settingService.SetUserSyncService(userService)

	// 初始化处理器
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	orderHandler := handler.NewOrderHandler(orderService, settingService)
	qrcodeHandler := handler.NewQrcodeHandler(qrcodeService)
	settingHandler := handler.NewSettingHandler(settingService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	publicOrderHandler := handler.NewPublicOrderHandler(orderService, settingService, qrcodeService, db)
	menuHandler := handler.NewMenuHandler()

	// 初始化定时任务调度器
	taskScheduler := scheduler.NewScheduler(orderService, settingService)
	taskScheduler.Start()

	// 初始化路由
	router := setupRouter(jwtManager, userHandler, authHandler, orderHandler, qrcodeHandler, settingHandler, paymentHandler, publicOrderHandler, menuHandler)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         ":" + config.AppConfig.Server.Port,
		Handler:      router,
		ReadTimeout:  config.AppConfig.Server.ReadTimeout,
		WriteTimeout: config.AppConfig.Server.WriteTimeout,
	}

	// 设置优雅关闭
	go func() {
		log.Printf("Server starting on port %s", config.AppConfig.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 停止定时任务调度器
	taskScheduler.Stop()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

// initDatabase 初始化数据库连接
func initDatabase() (*gorm.DB, error) {
	cfg := &config.AppConfig.Database

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}

// setupRouter 设置路由
func setupRouter(jwtManager *jwt.JWTManager, userHandler *handler.UserHandler, authHandler *handler.AuthHandler, orderHandler *handler.OrderHandler, qrcodeHandler *handler.QrcodeHandler, settingHandler *handler.SettingHandler, paymentHandler *handler.PaymentHandler, publicOrderHandler *handler.PublicOrderHandler, menuHandler *handler.MenuHandler) *gin.Engine {
	router := gin.New()

	// 中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// 公开API路由组（供第三方商户使用）
	publicAPI := router.Group("/api/public")
	{
		publicAPI.POST("/order", publicOrderHandler.CreateOrder)                      // 创建订单
		publicAPI.GET("/order/:order_id", publicOrderHandler.GetOrder)                // 获取订单详情
		publicAPI.GET("/order/:order_id/status", publicOrderHandler.CheckOrderStatus) // 检查订单状态
	}

	// API路由组
	v2 := router.Group("/api/v2")
	{
		// 认证路由（无需认证）
		auth := v2.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// 统一的订单路由（支持公开和认证访问）
		orders := v2.Group("/orders")
		{
			// 需要认证的订单管理操作（放在前面，避免路由冲突）
			authenticated := orders.Group("")
			authenticated.Use(middleware.AuthMiddleware(jwtManager))
			authenticated.Use(middleware.RequireAdmin())
			authenticated.Use(middleware.DataIsolationMiddleware())
			{
				authenticated.GET("", orderHandler.GetOrders)                           // 获取订单列表
				authenticated.POST("", orderHandler.CreateOrder)                        // 创建订单
				authenticated.POST("/close-expired", orderHandler.CloseExpiredOrders)   // 关闭过期订单
				authenticated.POST("/delete-expired", orderHandler.DeleteExpiredOrders) // 删除过期订单
			}

			// 使用条件认证中间件，支持公开和认证访问（放在后面）
			orders.Use(middleware.ConditionalAuthMiddleware(jwtManager))
			orders.GET("/:order_id", orderHandler.GetOrderUnified)              // 统一的订单查询
			orders.GET("/:order_id/status", orderHandler.GetOrderStatusUnified) // 统一的订单状态查询
			orders.PUT("/:order_id", orderHandler.UpdateOrder)                  // 更新订单（需要认证）
			orders.DELETE("/:order_id", orderHandler.DeleteOrder)               // 删除订单（需要认证）
			orders.PUT("/:order_id/close", orderHandler.CloseOrder)             // 关闭订单（需要认证）
			orders.GET("/:order_id/return-url", orderHandler.GenerateReturnURL) // 生成返回URL（需要认证）
		}

		// 需要认证的路由
		authenticated := v2.Group("")
		authenticated.Use(middleware.AuthMiddleware(jwtManager))
		{
			// 用户管理路由（需要超级管理员权限）
			users := authenticated.Group("/users")
			users.Use(middleware.RequireSuperAdmin())
			{
				users.GET("", userHandler.GetUsers)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser) // 获取单个用户
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.PATCH("/:id/password", userHandler.ResetPassword) // 优雅的重置密码路由
			}

			// 收款码管理路由（需要管理员权限 + 数据隔离）
			qrcodes := authenticated.Group("/qrcodes")
			qrcodes.Use(middleware.RequireAdmin())
			qrcodes.Use(middleware.DataIsolationMiddleware()) // 添加数据隔离中间件
			{
				qrcodes.GET("", qrcodeHandler.GetQrcodes)                    // 获取收款码列表
				qrcodes.POST("", qrcodeHandler.CreateQrcode)                 // 创建收款码
				qrcodes.DELETE("/:id", qrcodeHandler.DeleteQrcode)           // 删除收款码
				qrcodes.PUT("/:id/status", qrcodeHandler.UpdateQrcodeStatus) // 更新收款码状态
				qrcodes.POST("/parse", qrcodeHandler.ParseQrcode)            // 解析收款码
			}

			// 二维码生成路由（无需认证，直接返回图片）
			v2.GET("/qrcode/generate", qrcodeHandler.GenerateQrcode) // 生成二维码

			// 系统设置路由（普通admin和超级管理员都可以访问，但只能操作自己的数据）
			settings := authenticated.Group("/settings")
			{
				settings.GET("", settingHandler.GetSystemConfig)             // 获取系统配置
				settings.POST("", settingHandler.UpdateSystemConfig)         // 更新系统配置
				settings.GET("/monitor", settingHandler.GetMonitorConfig)    // 获取监控配置
				settings.PUT("/monitor", settingHandler.UpdateMonitorConfig) // 更新监控配置
			}

			// 系统信息路由（需要管理员权限）
			system := authenticated.Group("/system")
			system.Use(middleware.RequireAdmin())
			{
				system.GET("/status", settingHandler.GetSystemStatus) // 获取系统状态
				system.GET("/info", settingHandler.GetSystemInfo)     // 获取系统信息
				system.GET("/update", settingHandler.CheckUpdate)     // 检查更新
				system.GET("/ip", settingHandler.GetIPInfo)           // 获取IP信息
			}

			// 全局系统状态路由（只有超级管理员可以访问）
			globalSystem := authenticated.Group("/system")
			globalSystem.Use(middleware.RequireSuperAdmin())
			{
				globalSystem.GET("/global-status", settingHandler.GetGlobalSystemStatus) // 获取全局系统状态
			}

			// 仪表板路由（需要管理员权限）
			authenticated.GET("/dashboard", settingHandler.GetDashboard)

			// 菜单路由（需要管理员权限）
			authenticated.GET("/menu", menuHandler.GetMenu)

			// 当前用户信息
			authenticated.GET("/me", authHandler.GetCurrentUser)
			authenticated.POST("/logout", authHandler.Logout)
		}
	}

	// 监控端API路由（无需认证，通过签名验证）
	monitor := v2.Group("/monitor")
	{
		monitor.POST("/heart", settingHandler.MonitorHeart) // 监控心跳（POST）
		monitor.GET("/heart", settingHandler.MonitorHeart)  // 监控心跳（GET，兼容Android客户端）
		monitor.POST("/push", settingHandler.MonitorPush)   // 监控推送
	}

	// 支付页面API路由（无需认证）
	public := router.Group("/api/public")
	{
		public.GET("/orders/:order_id", paymentHandler.GetPaymentOrder)              // 获取支付订单信息
		public.GET("/orders/:order_id/status", paymentHandler.CheckPaymentStatus)    // 检查支付状态
		public.GET("/orders/:order_id/return-url", paymentHandler.GenerateReturnURL) // 生成返回URL
		// 二维码生成已移至 /api/v2/qrcode/generate
	}

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "Route not found")
	})

	return router
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
