package service

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// 系统设置相关错误
var (
	ErrSettingNotFound = errors.New("setting not found")
	ErrInvalidSign     = errors.New("invalid signature")
)

// UserSyncService 用户同步服务接口（避免循环依赖）
type UserSyncService interface {
	SyncSettingToUser(userID uint, settingKey, settingValue string) error
}

// SettingService 系统设置服务接口
type SettingService interface {
	GetSystemConfig(userID uint) (*model.SystemConfigResponse, error)
	UpdateSystemConfig(userID uint, req *model.SystemConfigRequest) error
	GetSystemStatus(userID uint) (*model.SystemStatusResponse, error)
	GetGlobalSystemStatus() (*model.SystemStatusResponse, error)
	GetDashboard(userID uint) (*model.DashboardResponse, error)
	GetMonitorConfig(userID uint) (*model.MonitorConfigResponse, error)
	UpdateMonitorConfig(userID uint, req *model.MonitorConfigRequest) error
	ProcessMonitorHeart(req *model.MonitorHeartRequest) error
	ProcessMonitorPush(req *model.MonitorPushRequest) error
	GetSystemInfo() (*model.SystemInfoResponse, error)
	CheckUpdate(req *model.UpdateSystemRequest) (*model.UpdateSystemResponse, error)
	GetIPInfo() (*model.IPInfoResponse, error)
	SetUserSyncService(userSyncService UserSyncService)
	GetSettingValue(key string) (string, error) // 新增方法，用于公开API
	GetUserSettingValue(userID uint, key string) (string, error)
	// 商户映射相关方法
	GetUserIDByAppID(appId string) (uint, error)
	SetMerchantMapping(userID uint, appId string) error
	// 监控端状态检查
	CheckAndUpdateMonitorStatus() error
}

// settingService 系统设置服务实现
type settingService struct {
	settingRepo     repository.SettingRepository
	orderRepo       repository.OrderRepository
	merchantRepo    repository.MerchantRepository
	userSyncService UserSyncService
	startTime       time.Time
}

// NewSettingService 创建系统设置服务
func NewSettingService(settingRepo repository.SettingRepository, orderRepo repository.OrderRepository, merchantRepo repository.MerchantRepository) SettingService {
	return &settingService{
		settingRepo:  settingRepo,
		orderRepo:    orderRepo,
		merchantRepo: merchantRepo,
		startTime:    time.Now(),
	}
}

// SetUserSyncService 设置用户同步服务
func (s *settingService) SetUserSyncService(userSyncService UserSyncService) {
	s.userSyncService = userSyncService
}

// GetSettingValue 获取设置值（用于公开API，默认使用用户ID=1）
func (s *settingService) GetSettingValue(key string) (string, error) {
	setting, err := s.settingRepo.GetSetting(1, key) // 默认使用用户ID=1
	if err != nil {
		return "", err
	}
	return setting.Vvalue, nil
}

// GetSystemConfig 获取系统配置
func (s *settingService) GetSystemConfig(userID uint) (*model.SystemConfigResponse, error) {
	settingsMap, err := s.settingRepo.GetSettingsMap(userID)
	if err != nil {
		return nil, err
	}

	// 如果key为空，生成一个新的
	if settingsMap["key"] == "" {
		newKey := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().Unix()))))
		err = s.settingRepo.UpdateSetting(userID, "key", newKey)
		if err != nil {
			return nil, err
		}
		settingsMap["key"] = newKey
	}

	return &model.SystemConfigResponse{
		User:      settingsMap["user"],
		Pass:      settingsMap["pass"],
		NotifyUrl: settingsMap["notifyUrl"],
		ReturnUrl: settingsMap["returnUrl"],
		Key:       settingsMap["key"],
		AppId:     settingsMap["appId"], // 从setting表获取AppID
		Lastheart: settingsMap["lastheart"],
		Lastpay:   settingsMap["lastpay"],
		Jkstate:   settingsMap["jkstate"],
		Close:     settingsMap["close"],
		PayQf:     settingsMap["payQf"],
		Wxpay:     settingsMap["wxpay"],
		Zfbpay:    settingsMap["zfbpay"],
		// 注册配置字段
		RegisterEnabled:         settingsMap["register_enabled"],
		RegisterDefaultRole:     settingsMap["register_default_role"],
		RegisterRequireApproval: settingsMap["register_require_approval"],
		RegisterRateLimit:       settingsMap["register_rate_limit"],
	}, nil
}

// UpdateSystemConfig 更新系统配置
func (s *settingService) UpdateSystemConfig(userID uint, req *model.SystemConfigRequest) error {
	// 处理密码加密
	hashedPassword := req.Pass
	if req.Pass != "" {
		// 检查是否已经是哈希值（bcrypt哈希值以$2a$、$2b$、$2x$、$2y$开头）
		if len(req.Pass) < 60 || (!strings.HasPrefix(req.Pass, "$2a$") &&
			!strings.HasPrefix(req.Pass, "$2b$") &&
			!strings.HasPrefix(req.Pass, "$2x$") &&
			!strings.HasPrefix(req.Pass, "$2y$")) {
			// 明文密码，需要加密
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Pass), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Failed to hash password: %v", err)
				return err
			}
			hashedPassword = string(hashedBytes)
		}
	}

	// 允许修改的配置项
	allowedKeys := map[string]string{
		"user":      req.User,
		"pass":      hashedPassword, // 使用加密后的密码
		"notifyUrl": req.NotifyUrl,
		"returnUrl": req.ReturnUrl,
		"close":     req.Close,
		"payQf":     req.PayQf,
		"wxpay":     req.Wxpay,
		"zfbpay":    req.Zfbpay,
		"appId":     req.AppId, // 新增AppId字段
		// 注册配置字段
		"register_enabled":          req.RegisterEnabled,
		"register_default_role":     req.RegisterDefaultRole,
		"register_require_approval": req.RegisterRequireApproval,
		"register_rate_limit":       req.RegisterRateLimit,
	}

	// 如果提供了key，也允许更新
	if req.Key != "" {
		allowedKeys["key"] = req.Key
	}

	// 处理AppId的商户映射
	if req.AppId != "" {
		// 检查AppId是否已被其他用户使用
		exists, err := s.merchantRepo.CheckAppIdExists(req.AppId, userID)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("商户ID已被使用")
		}

		// 更新商户映射
		err = s.merchantRepo.UpsertMapping(req.AppId, userID)
		if err != nil {
			return err
		}
	}

	// 更新setting表
	if err := s.settingRepo.BatchUpdateSettings(userID, allowedKeys); err != nil {
		return err
	}

	// 触发反向同步：将需要同步的字段同步到users表
	if s.userSyncService != nil {
		syncFields := []string{"user", "pass"}
		for _, key := range syncFields {
			if value, exists := allowedKeys[key]; exists && value != "" {
				if err := s.userSyncService.SyncSettingToUser(userID, key, value); err != nil {
					// 记录错误但不中断操作
					log.Printf("Warning: Failed to sync setting %s to user %d: %v", key, userID, err)
				}
			}
		}
	}

	return nil
}

// GetSystemStatus 获取系统状态
func (s *settingService) GetSystemStatus(userID uint) (*model.SystemStatusResponse, error) {
	// 获取今日统计
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	todayStats, err := s.orderRepo.GetOrderStatsByDateRange(userID, today.Unix(), tomorrow.Unix())
	if err != nil {
		return nil, err
	}

	// 获取总统计
	totalStats, err := s.orderRepo.GetOrderStatsByDateRange(userID, 0, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	// 获取监控状态
	settingsMap, err := s.settingRepo.GetSettingsMap(userID)
	if err != nil {
		return nil, err
	}

	// 计算监控状态
	monitorStatus := s.calculateMonitorStatus(settingsMap["lastheart"])
	lastHeartTime := s.formatTimestamp(settingsMap["lastheart"])
	lastPayTime := s.formatTimestamp(settingsMap["lastpay"])

	response := &model.SystemStatusResponse{
		TodayOrder:        todayStats.TotalOrders,
		TodaySuccessOrder: todayStats.SuccessOrders,
		TodayCloseOrder:   todayStats.ClosedOrders,
		TodayMoney:        todayStats.TotalAmount,
		CountOrder:        totalStats.TotalOrders,
		CountMoney:        totalStats.TotalAmount,
		Lastheart:         settingsMap["lastheart"],
		Lastpay:           settingsMap["lastpay"],
		Jkstate:           settingsMap["jkstate"],
		MonitorStatus:     monitorStatus,
		LastHeartTime:     lastHeartTime,
		LastPayTime:       lastPayTime,
	}

	log.Printf("系统状态统计 - 用户ID: %d, 今日总订单: %d, 今日成功: %d, 今日失败: %d, 今日收入: %.2f",
		userID, response.TodayOrder, response.TodaySuccessOrder, response.TodayCloseOrder, response.TodayMoney)

	return response, nil
}

// GetGlobalSystemStatus 获取全局系统状态（所有用户的汇总数据）
// 只有超级管理员可以调用此接口
func (s *settingService) GetGlobalSystemStatus() (*model.SystemStatusResponse, error) {
	// 获取今日统计（所有用户）
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	todayStats, err := s.orderRepo.GetOrderStatsByDateRange(0, today.Unix(), tomorrow.Unix()) // userID=0表示所有用户
	if err != nil {
		return nil, err
	}

	// 获取总统计（所有用户）
	totalStats, err := s.orderRepo.GetOrderStatsByDateRange(0, 0, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	// 获取全局监控状态（使用用户ID=1的设置作为全局设置）
	settingsMap, err := s.settingRepo.GetSettingsMap(1)
	if err != nil {
		return nil, err
	}

	// 计算监控状态
	monitorStatus := s.calculateMonitorStatus(settingsMap["lastheart"])
	lastHeartTime := s.formatTimestamp(settingsMap["lastheart"])
	lastPayTime := s.formatTimestamp(settingsMap["lastpay"])

	response := &model.SystemStatusResponse{
		TodayOrder:        todayStats.TotalOrders,
		TodaySuccessOrder: todayStats.SuccessOrders,
		TodayCloseOrder:   todayStats.ClosedOrders,
		TodayMoney:        todayStats.TotalAmount,
		CountOrder:        totalStats.TotalOrders,
		CountMoney:        totalStats.TotalAmount,
		Lastheart:         settingsMap["lastheart"],
		Lastpay:           settingsMap["lastpay"],
		Jkstate:           settingsMap["jkstate"],
		MonitorStatus:     monitorStatus,
		LastHeartTime:     lastHeartTime,
		LastPayTime:       lastPayTime,
	}

	log.Printf("全局系统状态统计 - 今日总订单: %d, 今日成功: %d, 今日失败: %d, 今日收入: %.2f",
		response.TodayOrder, response.TodaySuccessOrder, response.TodayCloseOrder, response.TodayMoney)

	return response, nil
}

// GetDashboard 获取仪表板数据
func (s *settingService) GetDashboard(userID uint) (*model.DashboardResponse, error) {
	// 获取系统状态
	status, err := s.GetSystemStatus(userID)
	if err != nil {
		return nil, err
	}

	// 获取系统信息
	sysInfo, err := s.GetSystemInfo()
	if err != nil {
		return nil, err
	}

	return &model.DashboardResponse{
		TodayOrder:        status.TodayOrder,
		TodaySuccessOrder: status.TodaySuccessOrder,
		TodayCloseOrder:   status.TodayCloseOrder,
		TodayMoney:        status.TodayMoney,
		CountOrder:        status.CountOrder,
		CountMoney:        status.CountMoney,
		PHPVersion:        "N/A (Go Version)",
		PHPOS:             runtime.GOOS,
		Server:            "Go/Gin Server",
		MySQL:             "MySQL 8.0+",
		Thinkphp:          "N/A (Go Version)",
		RunTime:           s.getUptime(),
		Ver:               sysInfo.AppVersion,
		GD:                "N/A (Go Version)",
	}, nil
}

// GetMonitorConfig 获取监控配置
func (s *settingService) GetMonitorConfig(userID uint) (*model.MonitorConfigResponse, error) {
	settingsMap, err := s.settingRepo.GetSettingsMap(userID)
	if err != nil {
		return nil, err
	}

	return &model.MonitorConfigResponse{
		Jkstate:   settingsMap["jkstate"],
		Lastheart: settingsMap["lastheart"],
		Lastpay:   settingsMap["lastpay"],
	}, nil
}

// UpdateMonitorConfig 更新监控配置
func (s *settingService) UpdateMonitorConfig(userID uint, req *model.MonitorConfigRequest) error {
	return s.settingRepo.UpdateSetting(userID, "jkstate", req.Jk)
}

// ProcessMonitorHeart 处理监控心跳
func (s *settingService) ProcessMonitorHeart(req *model.MonitorHeartRequest) error {
	// 确定用户ID
	userID := uint(1) // 默认用户ID
	if req.AppID != "" {
		log.Printf("心跳请求包含AppID: %s，尝试查找对应用户", req.AppID)
		// 通过AppID查找用户ID
		foundUserID, err := s.GetUserIDByAppID(req.AppID)
		if err != nil {
			log.Printf("AppID查找失败: %s, 错误: %v", req.AppID, err)
			return fmt.Errorf("invalid appid: %s", req.AppID)
		}
		userID = foundUserID
		log.Printf("AppID %s 对应用户ID: %d", req.AppID, userID)
	} else {
		log.Printf("心跳请求未包含AppID，使用默认用户ID: %d", userID)
	}

	// 获取密钥
	key, err := s.settingRepo.GetSetting(userID, "key")
	if err != nil {
		return err
	}

	// 验证签名 - 适配Android端格式：md5(timestamp + key)
	expectedSign := fmt.Sprintf("%x", md5.Sum([]byte(req.T+key.Vvalue)))
	if req.Sign != expectedSign {
		return ErrInvalidSign
	}

	// 更新心跳时间和监控状态
	now := strconv.FormatInt(time.Now().Unix(), 10)
	err = s.settingRepo.UpdateSetting(userID, "lastheart", now)
	if err != nil {
		return err
	}

	return s.settingRepo.UpdateSetting(userID, "jkstate", "1")
}

// ProcessMonitorPush 处理监控推送
func (s *settingService) ProcessMonitorPush(req *model.MonitorPushRequest) error {
	// 确定用户ID
	userID := uint(1) // 默认用户ID
	if req.AppID != "" {
		// 通过AppID查找用户ID
		foundUserID, err := s.GetUserIDByAppID(req.AppID)
		if err != nil {
			return fmt.Errorf("invalid appid: %s", req.AppID)
		}
		userID = foundUserID
	}

	// 获取密钥
	key, err := s.settingRepo.GetSetting(userID, "key")
	if err != nil {
		return err
	}

	// 验证签名 - 适配Android端格式：md5(type + price + timestamp + key)
	signStr := req.Type + req.Price + req.T + key.Vvalue
	expectedSign := fmt.Sprintf("%x", md5.Sum([]byte(signStr)))
	if req.Sign != expectedSign {
		return ErrInvalidSign
	}

	// 根据价格和类型查找对应的待支付订单
	price, err := strconv.ParseFloat(req.Price, 64)
	if err != nil {
		return fmt.Errorf("invalid price: %s", req.Price)
	}

	orderType, err := strconv.Atoi(req.Type)
	if err != nil {
		return fmt.Errorf("invalid type: %s", req.Type)
	}

	// 查找该用户最近创建的匹配订单
	order, err := s.orderRepo.GetRecentPendingOrderByPriceAndType(userID, price, orderType)
	if err != nil {
		log.Printf("未找到匹配的订单: 用户ID=%d, 价格=%f, 类型=%d, 错误=%v", userID, price, orderType, err)
		// 即使没找到订单，也更新lastpay时间
	} else {
		// 更新订单状态为已支付
		order.State = model.OrderStatusPaid
		order.Pay_date = time.Now().Unix()

		err = s.orderRepo.Update(order)
		if err != nil {
			log.Printf("更新订单状态失败: 订单ID=%s, 错误=%v", order.Order_id, err)
		} else {
			log.Printf("订单支付成功: 订单ID=%s, 用户ID=%d, 价格=%f", order.Order_id, userID, price)
		}
	}

	// 更新最后支付时间
	now := strconv.FormatInt(time.Now().Unix(), 10)
	return s.settingRepo.UpdateSetting(userID, "lastpay", now)
}

// GetSystemInfo 获取系统信息
func (s *settingService) GetSystemInfo() (*model.SystemInfoResponse, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &model.SystemInfoResponse{
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		Server:       "VMQFox Go API Server",
		MySQLVersion: "MySQL 8.0+",
		AppVersion:   "v2.0.0",
		RunTime:      s.getUptime(),
		StartTime:    s.startTime,
		MemoryUsage:  fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024),
		GoroutineNum: runtime.NumGoroutine(),
	}, nil
}

// CheckUpdate 检查更新
func (s *settingService) CheckUpdate(req *model.UpdateSystemRequest) (*model.UpdateSystemResponse, error) {
	// 模拟检查更新逻辑
	return &model.UpdateSystemResponse{
		HasUpdate:      false,
		CurrentVersion: "v2.0.0",
		LatestVersion:  "v2.0.0",
		UpdateUrl:      "",
		UpdateLog:      "当前已是最新版本",
	}, nil
}

// GetIPInfo 获取IP信息
func (s *settingService) GetIPInfo() (*model.IPInfoResponse, error) {
	// 模拟IP信息
	return &model.IPInfoResponse{
		IP:       "127.0.0.1",
		Country:  "中国",
		Region:   "本地",
		City:     "本地",
		ISP:      "本地网络",
		Location: "本地服务器",
	}, nil
}

// getUptime 获取运行时间
func (s *settingService) getUptime() string {
	uptime := time.Since(s.startTime)
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	} else {
		return fmt.Sprintf("%d分钟", minutes)
	}
}

// GetUserSettingValue 获取指定用户的设置值
func (s *settingService) GetUserSettingValue(userID uint, key string) (string, error) {
	setting, err := s.settingRepo.GetSetting(userID, key)
	if err != nil {
		return "", err
	}
	return setting.Vvalue, nil
}

// GetUserIDByAppID 通过AppID获取用户ID
func (s *settingService) GetUserIDByAppID(appId string) (uint, error) {
	return s.merchantRepo.GetUserIDByAppID(appId)
}

// SetMerchantMapping 设置商户映射
func (s *settingService) SetMerchantMapping(userID uint, appId string) error {
	// 检查AppId是否已被其他用户使用
	exists, err := s.merchantRepo.CheckAppIdExists(appId, userID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("商户ID已被使用")
	}

	// 更新商户映射
	return s.merchantRepo.UpsertMapping(appId, userID)
}

// calculateMonitorStatus 计算监控状态
// 返回值：0-未知 1-正常 2-异常
func (s *settingService) calculateMonitorStatus(lastHeartStr string) int {
	if lastHeartStr == "" {
		return 0 // 未知状态
	}

	heartTime, err := strconv.ParseInt(lastHeartStr, 10, 64)
	if err != nil || heartTime <= 0 {
		return 0 // 未知状态
	}

	// 心跳超时时间：180秒（3分钟）
	const heartbeatTimeout = 180
	currentTime := time.Now().Unix()

	if currentTime-heartTime < heartbeatTimeout {
		return 1 // 正常
	} else {
		return 2 // 异常
	}
}

// formatTimestamp 格式化时间戳
func (s *settingService) formatTimestamp(timestampStr string) string {
	if timestampStr == "" {
		return ""
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil || timestamp <= 0 {
		return ""
	}

	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

// CheckAndUpdateMonitorStatus 检查并更新所有用户的监控端状态
func (s *settingService) CheckAndUpdateMonitorStatus() error {
	// 心跳超时时间：180秒（3分钟）
	const heartbeatTimeout = 180
	currentTime := time.Now().Unix()

	// 获取所有有lastheart设置的用户
	settings, err := s.settingRepo.GetAllSettingsByKey("lastheart")
	if err != nil {
		return err
	}

	for _, setting := range settings {
		userID := setting.User_id
		lastHeartStr := setting.Vvalue

		if lastHeartStr == "" {
			// 没有心跳记录，设置为掉线状态
			s.settingRepo.UpdateSetting(userID, "jkstate", "0")
			continue
		}

		heartTime, err := strconv.ParseInt(lastHeartStr, 10, 64)
		if err != nil || heartTime <= 0 {
			// 心跳时间格式错误，设置为掉线状态
			s.settingRepo.UpdateSetting(userID, "jkstate", "0")
			continue
		}

		// 检查心跳是否超时
		if currentTime-heartTime >= heartbeatTimeout {
			// 心跳超时，设置为掉线状态
			s.settingRepo.UpdateSetting(userID, "jkstate", "0")
		}
		// 如果心跳正常，不需要更新，因为心跳接口会自动设置为1
	}

	return nil
}
