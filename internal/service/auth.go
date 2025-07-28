package service

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/repository"
	"vmqfox-api-go/pkg/jwt"

	"gorm.io/gorm"
)

// 认证相关错误
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrInvalidToken       = errors.New("invalid token")
)

// 注册频率限制
type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

var (
	rateLimitMap = make(map[string]*rateLimitEntry)
	rateLimitMux = sync.RWMutex{}
)

// AuthService 认证服务接口
type AuthService interface {
	Login(req *model.LoginRequest) (*model.LoginResponse, error)
	Register(req *model.RegisterRequest, clientIP string) (*model.RegisterResponse, error)
	RefreshToken(refreshToken string) (*model.LoginResponse, error)
	GetCurrentUser(userID uint) (*model.SafeUser, error)
	Logout(userID uint) error
}

// authService 认证服务实现
type authService struct {
	userRepo     repository.UserRepository
	settingRepo  repository.SettingRepository
	merchantRepo repository.MerchantRepository
	jwtManager   *jwt.JWTManager
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo repository.UserRepository, settingRepo repository.SettingRepository, merchantRepo repository.MerchantRepository, jwtManager *jwt.JWTManager) AuthService {
	return &authService{
		userRepo:     userRepo,
		settingRepo:  settingRepo,
		merchantRepo: merchantRepo,
		jwtManager:   jwtManager,
	}
}

// Login 用户登录
func (s *authService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	// 查找用户
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 验证密码
	if !user.VerifyPassword(req.Password) {
		return nil, ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsEnabled() {
		return nil, ErrUserDisabled
	}

	// 生成JWT令牌
	accessToken, refreshToken, err := s.jwtManager.GenerateTokens(
		user.Id,
		user.Username,
		user.Role,
		user.Status,
	)
	if err != nil {
		return nil, err
	}

	// 返回登录响应
	return &model.LoginResponse{
		User:          user,
		Access_token:  accessToken,
		Refresh_token: refreshToken,
		Expires_in:    7200, // 2小时，应该从配置读取
	}, nil
}

// getRegisterConfig 获取注册配置
func (s *authService) getRegisterConfig() (*model.RegisterConfig, error) {
	// 获取配置（使用用户ID=1作为全局配置）
	settingsMap, err := s.settingRepo.GetSettingsMap(1)
	if err != nil {
		return nil, err
	}

	config := &model.RegisterConfig{
		Enabled:         settingsMap["register_enabled"] == "1",
		DefaultRole:     settingsMap["register_default_role"],
		RequireApproval: settingsMap["register_require_approval"] == "1",
		RateLimit:       10, // 默认值
	}

	// 解析频率限制
	if rateLimitStr, exists := settingsMap["register_rate_limit"]; exists {
		if rateLimit, err := strconv.Atoi(rateLimitStr); err == nil {
			config.RateLimit = rateLimit
		}
	}

	// 设置默认值
	if config.DefaultRole == "" {
		config.DefaultRole = model.RoleAdmin
	}

	// 验证默认角色是否有效
	if !isValidRole(config.DefaultRole) {
		config.DefaultRole = model.RoleAdmin
	}

	return config, nil
}

// isValidRole 验证角色是否有效
func isValidRole(role string) bool {
	return role == model.RoleAdmin || role == model.RoleSuperAdmin
}

// checkRateLimit 检查注册频率限制
func (s *authService) checkRateLimit(clientIP string, limit int) error {
	rateLimitMux.Lock()
	defer rateLimitMux.Unlock()

	now := time.Now()
	entry, exists := rateLimitMap[clientIP]

	if !exists {
		// 首次注册
		rateLimitMap[clientIP] = &rateLimitEntry{
			count:     1,
			resetTime: now.Add(time.Hour),
		}
		return nil
	}

	// 检查是否需要重置计数器
	if now.After(entry.resetTime) {
		entry.count = 1
		entry.resetTime = now.Add(time.Hour)
		return nil
	}

	// 检查是否超过限制
	if entry.count >= limit {
		return errors.New("注册频率过高，请稍后再试")
	}

	// 增加计数
	entry.count++
	return nil
}

// Register 用户注册
func (s *authService) Register(req *model.RegisterRequest, clientIP string) (*model.RegisterResponse, error) {
	// 获取注册配置
	config, err := s.getRegisterConfig()
	if err != nil {
		return nil, err
	}

	// 检查是否开放注册
	if !config.Enabled {
		return nil, errors.New("用户注册功能已关闭")
	}

	// 检查注册频率限制
	if err := s.checkRateLimit(clientIP, config.RateLimit); err != nil {
		return nil, err
	}

	// 验证请求数据
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 检查用户名是否已存在
	exists, err := s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	exists, err = s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("邮箱已存在")
	}

	// 创建用户对象
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,        // 密码会在BeforeCreate钩子中自动加密
		Role:     config.DefaultRole,  // 使用配置中的默认角色
		Status:   model.StatusEnabled, // 默认启用
	}

	// 保存用户
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// 为新用户创建默认设置
	if err := s.createUserSettings(user); err != nil {
		// 记录错误并返回错误，因为这是关键功能
		log.Printf("Error: Failed to create default settings for user %d: %v", user.Id, err)
		return nil, fmt.Errorf("failed to create default settings: %v", err)
	}

	// 返回注册响应（不包含敏感信息）
	return &model.RegisterResponse{
		Message: "注册成功",
		User:    user,
	}, nil
}

// RefreshToken 刷新令牌
func (s *authService) RefreshToken(refreshToken string) (*model.LoginResponse, error) {
	// 验证刷新令牌
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 检查令牌类型
	if claims.Type != "refresh" {
		return nil, ErrInvalidToken
	}

	// 获取用户信息（确保用户仍然存在且启用）
	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// 检查用户状态
	if !user.IsEnabled() {
		return nil, ErrUserDisabled
	}

	// 生成新的令牌对
	accessToken, newRefreshToken, err := s.jwtManager.GenerateTokens(
		user.Id,
		user.Username,
		user.Role,
		user.Status,
	)
	if err != nil {
		return nil, err
	}

	// 返回新的令牌
	return &model.LoginResponse{
		User:          user,
		Access_token:  accessToken,
		Refresh_token: newRefreshToken,
		Expires_in:    7200, // 2小时，应该从配置读取
	}, nil
}

// GetCurrentUser 获取当前用户信息
func (s *authService) GetCurrentUser(userID uint) (*model.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// Logout 用户注销
func (s *authService) Logout(userID uint) error {
	// 在实际应用中，这里可能需要：
	// 1. 将令牌加入黑名单
	// 2. 记录注销日志
	// 3. 清理相关缓存

	// 目前只是一个占位符实现
	// 由于JWT是无状态的，客户端删除令牌即可实现注销
	return nil
}

// createUserSettings 为新用户创建完整的setting表数据
func (s *authService) createUserSettings(user *model.User) error {
	// 生成key，与原版ThinkPHP5.1保持一致
	keyValue := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().Unix()))))

	// 定义默认设置
	defaultSettings := map[string]string{
		"user":      user.Username,
		"pass":      user.Password,                // 使用加密后的密码
		"close":     "5",                          // 订单过期时间（小时）
		"jkstate":   "0",                          // 监控状态
		"key":       keyValue,                     // API密钥
		"lastheart": "",                           // 最后心跳时间
		"lastpay":   "",                           // 最后支付时间
		"notifyUrl": "https://example.com/notify", // 通知URL
		"returnUrl": "https://example.com/return", // 返回URL
		"payQf":     "1",                          // 支付配置
		"wxpay":     "",                           // 微信支付二维码
		"zfbpay":    "",                           // 支付宝支付二维码
	}

	// 批量创建设置
	for key, value := range defaultSettings {
		setting := &model.Setting{
			Vkey:    key,
			User_id: user.Id,
			Vvalue:  value,
		}

		if err := s.settingRepo.CreateSetting(setting); err != nil {
			log.Printf("Failed to create setting %s for user %d: %v", key, user.Id, err)
			// 继续创建其他设置，不因为单个设置失败而中断
		}
	}

	// 自动生成AppID并保存到setting表
	appID, err := s.generateAppID()
	if err != nil {
		log.Printf("Failed to generate AppID for user %d: %v", user.Id, err)
		// 不因为AppID生成失败而中断注册流程，但记录错误
	} else {
		// 保存AppID到setting表
		if err := s.settingRepo.UpdateSetting(user.Id, "appId", appID); err != nil {
			log.Printf("Failed to save AppID to settings for user %d with AppID %s: %v", user.Id, appID, err)
			// 不因为保存失败而中断注册流程，但记录错误
		} else {
			// 同步到merchant_mapping表
			if err := s.merchantRepo.UpsertMapping(appID, user.Id); err != nil {
				log.Printf("Failed to sync AppID to merchant mapping for user %d with AppID %s: %v", user.Id, appID, err)
				// 不因为同步失败而中断注册流程，但记录错误
			} else {
				log.Printf("Successfully created and synced AppID %s for user %s (ID: %d)", appID, user.Username, user.Id)
			}
		}
	}

	log.Printf("Successfully created settings for user %s (ID: %d)", user.Username, user.Id)
	return nil
}

// generateAppID 生成唯一的AppID
func (s *authService) generateAppID() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// 生成12位随机字符串
		randomBytes := make([]byte, 12)
		for i := range randomBytes {
			randomBytes[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		}

		appID := "VMQ_" + string(randomBytes)

		// 检查AppID是否已存在
		exists, err := s.merchantRepo.CheckAppIdExists(appID, 0)
		if err != nil {
			return "", fmt.Errorf("failed to check AppID existence: %v", err)
		}

		if !exists {
			return appID, nil
		}

		// 如果存在，稍微延迟后重试
		time.Sleep(time.Millisecond)
	}

	return "", errors.New("failed to generate unique AppID after multiple attempts")
}
