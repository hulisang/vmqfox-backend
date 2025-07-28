package service

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"time"

	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// UserService 用户服务接口
type UserService interface {
	GetUsers(page, limit int, search string) ([]*model.User, int64, error)
	GetUserByID(id uint) (*model.User, error)
	CreateUser(req *model.CreateUserRequest) (*model.User, error)
	UpdateUser(id uint, req *model.UpdateUserRequest) (*model.User, error)
	DeleteUser(id uint) error
	ResetPassword(id uint, password string) error
	SyncSettingToUser(userID uint, settingKey, settingValue string) error
}

// userService 用户服务实现
type userService struct {
	userRepo    repository.UserRepository
	settingRepo repository.SettingRepository
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository, settingRepo repository.SettingRepository) UserService {
	return &userService{
		userRepo:    userRepo,
		settingRepo: settingRepo,
	}
}

// GetUsers 获取用户列表
func (s *userService) GetUsers(page, limit int, search string) ([]*model.User, int64, error) {
	return s.userRepo.GetUsers(page, limit, search)
}

// GetUserByID 根据ID获取用户
func (s *userService) GetUserByID(id uint) (*model.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// CreateUser 创建用户
func (s *userService) CreateUser(req *model.CreateUserRequest) (*model.User, error) {
	// 检查用户名是否已存在
	exists, err := s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	// 检查邮箱是否已存在
	exists, err = s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	// 创建用户对象
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // 密码会在BeforeCreate钩子中自动加密
		Role:     req.Role,
	}

	// 设置默认角色
	if user.Role == "" {
		user.Role = model.RoleAdmin
	}

	// 设置默认状态为启用（保持数据库兼容性）
	user.Status = model.StatusEnabled

	// 保存用户
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// 同步到setting表
	if err := s.syncUserToSetting(user); err != nil {
		// 记录错误但不影响用户创建
		log.Printf("Warning: Failed to sync user to setting table: %v", err)
	}

	return user, nil
}

// UpdateUser 更新用户
func (s *userService) UpdateUser(id uint, req *model.UpdateUserRequest) (*model.User, error) {
	// 获取现有用户
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 检查用户名是否已被其他用户使用
	if req.Username != "" && req.Username != user.Username {
		exists, err := s.userRepo.ExistsByUsernameExcludeID(req.Username, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserExists
		}
		user.Username = req.Username
	}

	// 检查邮箱是否已被其他用户使用
	if req.Email != "" && req.Email != user.Email {
		exists, err := s.userRepo.ExistsByEmailExcludeID(req.Email, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserExists
		}
		user.Email = req.Email
	}

	// 更新角色
	if req.Role != "" {
		user.Role = req.Role
	}

	// 注意：不再支持更新用户状态，因为用户状态逻辑已移除

	// 保存更新
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	// 同步到setting表
	if err := s.syncUserToSetting(user); err != nil {
		// 记录错误但不影响用户更新
		log.Printf("Warning: Failed to sync user to setting table: %v", err)
	}

	return user, nil
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(id uint) error {
	// 检查用户是否存在
	_, err := s.userRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 先删除用户的setting表数据
	if err := s.deleteUserSettings(id); err != nil {
		log.Printf("Warning: Failed to delete user settings for user %d: %v", id, err)
		// 不因为setting删除失败而中断用户删除
	}

	// 删除用户
	return s.userRepo.Delete(id)
}

// ResetPassword 重置密码
func (s *userService) ResetPassword(id uint, password string) error {
	// 获取用户
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 直接更新密码，避免BeforeUpdate钩子的重复加密
	if err := s.userRepo.UpdatePasswordDirect(id, string(hashedPassword)); err != nil {
		return err
	}

	// 更新user对象的密码字段，以便正确同步到setting表
	user.Password = string(hashedPassword)

	// 同步到setting表
	if err := s.syncUserToSetting(user); err != nil {
		// 记录错误但不影响密码重置
		log.Printf("Warning: Failed to sync user to setting table: %v", err)
	}

	return nil
}

// syncUserToSetting 同步用户信息到setting表
func (s *userService) syncUserToSetting(user *model.User) error {
	// 检查是否是新用户（没有setting记录）
	existingSettings, err := s.settingRepo.GetAllSettings(user.Id)
	if err != nil || len(existingSettings) == 0 {
		// 新用户，创建完整的setting表数据
		return s.createUserSettings(user)
	}

	// 现有用户，只更新需要同步的字段
	return s.updateUserSettings(user)
}

// updateUserSettings 更新现有用户的setting表数据
func (s *userService) updateUserSettings(user *model.User) error {
	// 需要同步的字段映射
	syncFields := map[string]string{
		"user": user.Username,
		"pass": user.Password, // 同步加密后的密码
	}

	// 批量更新需要同步的字段
	for key, value := range syncFields {
		if err := s.settingRepo.UpdateSetting(user.Id, key, value); err != nil {
			log.Printf("Failed to update setting %s for user %d: %v", key, user.Id, err)
			// 继续更新其他字段，不因为单个字段失败而中断
		}
	}

	log.Printf("Successfully updated settings for user %s (ID: %d)", user.Username, user.Id)
	return nil
}

// createUserSettings 为新用户创建完整的setting表数据
func (s *userService) createUserSettings(user *model.User) error {
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

	log.Printf("Successfully created settings for user %s (ID: %d)", user.Username, user.Id)
	return nil
}

// deleteUserSettings 删除用户的所有setting表数据
func (s *userService) deleteUserSettings(userID uint) error {
	// 直接删除用户的所有设置
	if err := s.settingRepo.DeleteAllUserSettings(userID); err != nil {
		log.Printf("Failed to delete settings for user %d: %v", userID, err)
		return err
	}

	log.Printf("Successfully deleted all settings for user %d", userID)
	return nil
}

// SyncSettingToUser 从setting表同步数据到users表
func (s *userService) SyncSettingToUser(userID uint, settingKey, settingValue string) error {
	// 获取用户
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	// 根据setting的key来决定更新users表的哪个字段
	updated := false
	switch settingKey {
	case "user":
		if user.Username != settingValue {
			user.Username = settingValue
			updated = true
		}
	case "pass":
		// setting表中的密码现在已经是加密的，可以直接同步
		// 但是为了安全起见，我们跳过BeforeUpdate钩子直接更新
		if user.Password != settingValue {
			// 直接设置加密后的密码，不触发BeforeUpdate的重复加密
			if err := s.userRepo.UpdatePasswordDirect(userID, settingValue); err != nil {
				log.Printf("Failed to sync password to user %d: %v", userID, err)
				return err
			}
			log.Printf("Successfully synced password to user %d", userID)
			return nil // 密码已经单独处理，直接返回
		}
	default:
		// 其他setting字段不需要同步到users表
		return nil
	}

	// 如果有更新，保存到数据库
	if updated {
		if err := s.userRepo.Update(user); err != nil {
			log.Printf("Failed to sync setting %s to user %d: %v", settingKey, userID, err)
			return err
		}
		log.Printf("Successfully synced setting %s to user %d", settingKey, userID)
	}

	return nil
}
