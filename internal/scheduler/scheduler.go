package scheduler

import (
	"log"
	"time"

	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	orderService   service.OrderService
	settingService service.SettingService
	stopChan       chan bool
	running        bool
}

// NewScheduler 创建新的调度器
func NewScheduler(orderService service.OrderService, settingService service.SettingService) *Scheduler {
	return &Scheduler{
		orderService:   orderService,
		settingService: settingService,
		stopChan:       make(chan bool),
		running:        false,
	}
}

// Start 启动定时任务
func (s *Scheduler) Start() {
	if s.running {
		return
	}

	s.running = true
	log.Println("定时任务调度器启动")

	// 启动关闭过期订单的定时任务
	go s.startCloseExpiredOrdersTask()

	// 启动监控端状态检查的定时任务
	go s.startMonitorStatusCheckTask()
}

// Stop 停止定时任务
func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.running = false
	s.stopChan <- true
	log.Println("定时任务调度器停止")
}

// startCloseExpiredOrdersTask 启动关闭过期订单的定时任务
func (s *Scheduler) startCloseExpiredOrdersTask() {
	// 每分钟执行一次
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// 立即执行一次
	s.closeExpiredOrders()

	for {
		select {
		case <-ticker.C:
			s.closeExpiredOrders()
		case <-s.stopChan:
			log.Println("关闭过期订单定时任务停止")
			return
		}
	}
}

// closeExpiredOrders 关闭过期订单
func (s *Scheduler) closeExpiredOrders() {
	startTime := time.Now()
	log.Printf("定时任务：开始检查过期订单 - %s", startTime.Format("2006-01-02 15:04:05"))

	// 分别处理每个用户的过期订单，使用各自的超时配置
	// 这样更符合PHP版本的逻辑，每个用户可以有不同的超时时间
	totalClosedCount := int64(0)

	// 先处理所有用户的过期订单，使用各自的配置
	// 这里我们使用nil来处理所有用户，但在service层会读取每个用户的配置
	req := &model.CloseExpiredOrdersRequest{
		User_id: nil, // nil表示处理所有用户
		Limit:   100, // 限制每次处理100条，避免一次性处理太多
	}

	closedCount, err := s.orderService.CloseExpiredOrders(req)
	if err != nil {
		log.Printf("定时任务：关闭过期订单失败: %v", err)
		return
	}

	totalClosedCount += closedCount

	duration := time.Since(startTime)
	if totalClosedCount > 0 {
		log.Printf("定时任务：成功关闭 %d 条过期订单，耗时 %.2fms", totalClosedCount, float64(duration.Nanoseconds())/1e6)
	} else {
		log.Printf("定时任务：没有找到需要关闭的过期订单，耗时 %.2fms", float64(duration.Nanoseconds())/1e6)
	}
}

// startMonitorStatusCheckTask 启动监控端状态检查的定时任务
func (s *Scheduler) startMonitorStatusCheckTask() {
	log.Println("监控端状态检查定时任务启动")

	// 每分钟执行一次
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// 立即执行一次
	s.checkMonitorStatus()

	for {
		select {
		case <-ticker.C:
			s.checkMonitorStatus()
		case <-s.stopChan:
			log.Println("监控端状态检查定时任务停止")
			return
		}
	}
}

// checkMonitorStatus 检查监控端状态
func (s *Scheduler) checkMonitorStatus() {
	log.Printf("定时任务：开始检查监控端状态 - %s", time.Now().Format("2006-01-02 15:04:05"))

	// 获取所有用户的监控状态并检查心跳超时
	err := s.settingService.CheckAndUpdateMonitorStatus()
	if err != nil {
		log.Printf("定时任务：检查监控端状态失败: %v", err)
	} else {
		log.Printf("定时任务：监控端状态检查完成")
	}
}
