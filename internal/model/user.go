package model

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型 - 直接使用数据库字段名
type User struct {
	Id              uint    `json:"id" gorm:"primarykey"`
	Username        string  `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email           string  `json:"email" gorm:"uniqueIndex;size:100;not null"`
	Password        string  `json:"-" gorm:"size:191;not null"`
	Role            string  `json:"role" gorm:"type:enum('super_admin','admin');not null;default:'admin'"`
	Status          int     `json:"status" gorm:"type:tinyint(1);not null;default:1"`
	Last_login_time *int64  `json:"last_login_time"`
	Last_login_ip   *string `json:"last_login_ip" gorm:"size:45"`
	Created_at      int64   `json:"created_at" gorm:"not null"`
	Updated_at      int64   `json:"updated_at" gorm:"not null"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}

	now := time.Now().Unix()
	u.Created_at = now
	u.Updated_at = now

	return nil
}

// BeforeUpdate 更新前钩子
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.Updated_at = time.Now().Unix()
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// VerifyPassword 验证密码（别名方法，保持兼容性）
func (u *User) VerifyPassword(password string) bool {
	return u.CheckPassword(password)
}

// SetPassword 设置密码（加密存储）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// IsActive 检查用户是否激活
func (u *User) IsActive() bool {
	return u.Status == 1
}

// IsEnabled 检查用户是否启用（别名方法，保持兼容性）
func (u *User) IsEnabled() bool {
	return u.IsActive()
}

// IsSuperAdmin 检查是否为超级管理员
func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}

// IsAdmin 检查是否为管理员（包括超级管理员）
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleSuperAdmin
}

// CanManageUsers 检查是否可以管理用户
func (u *User) CanManageUsers() bool {
	return u.IsSuperAdmin() // 只有超级管理员可以管理用户
}

// CanViewGlobalData 检查是否可以查看全局数据
func (u *User) CanViewGlobalData() bool {
	return u.IsSuperAdmin() // 只有超级管理员可以查看全局数据
}

// CanModifySystemSettings 检查是否可以修改系统设置
func (u *User) CanModifySystemSettings() bool {
	return u.IsSuperAdmin() // 只有超级管理员可以修改系统设置
}

// ToSafeUser 转换为安全的用户信息（保持兼容性）
func (u *User) ToSafeUser() *User {
	return u // 由于我们直接使用数据库字段名，直接返回即可
}

// CreateUserRequest 创建用户请求（管理员使用）
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Role     string `json:"role" binding:"required,oneof=admin super_admin"`
}

// RegisterRequest 用户注册请求（公开注册使用）
type RegisterRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=50" example:"testuser"`
	Email           string `json:"email" binding:"required,email,max=100" example:"test@example.com"`
	Password        string `json:"password" binding:"required,min=6,max=50" example:"password123"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=6,max=50" example:"password123"`
}

// Validate 验证注册请求
func (r *RegisterRequest) Validate() error {
	// 密码确认验证
	if r.Password != r.ConfirmPassword {
		return errors.New("密码和确认密码不匹配")
	}

	// 用户名安全验证
	if err := r.validateUsername(); err != nil {
		return err
	}

	// 密码强度验证
	if err := r.validatePassword(); err != nil {
		return err
	}

	// 邮箱格式验证（额外检查）
	if err := r.validateEmail(); err != nil {
		return err
	}

	return nil
}

// validateUsername 验证用户名安全性
func (r *RegisterRequest) validateUsername() error {
	username := strings.TrimSpace(r.Username)

	// 检查长度
	if len(username) < 3 || len(username) > 50 {
		return errors.New("用户名长度必须在3-50个字符之间")
	}

	// 检查字符规则：只允许字母、数字、下划线
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", username)
	if !matched {
		return errors.New("用户名只能包含字母、数字和下划线")
	}

	// 不能以数字开头
	if unicode.IsDigit(rune(username[0])) {
		return errors.New("用户名不能以数字开头")
	}

	// 禁止的用户名
	forbiddenNames := []string{"admin", "root", "system", "test", "guest", "user", "null", "undefined"}
	for _, forbidden := range forbiddenNames {
		if strings.ToLower(username) == forbidden {
			return errors.New("该用户名不可用")
		}
	}

	return nil
}

// validatePassword 验证密码强度
func (r *RegisterRequest) validatePassword() error {
	password := r.Password

	// 检查长度
	if len(password) < 6 || len(password) > 50 {
		return errors.New("密码长度必须在6-50个字符之间")
	}

	// 检查是否包含至少一个字母和一个数字
	hasLetter := false
	hasDigit := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsDigit(char) {
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return errors.New("密码必须包含至少一个字母和一个数字")
	}

	// 检查常见弱密码
	weakPasswords := []string{"123456", "password", "123456789", "12345678", "qwerty", "abc123"}
	for _, weak := range weakPasswords {
		if strings.ToLower(password) == weak {
			return errors.New("密码过于简单，请使用更强的密码")
		}
	}

	return nil
}

// validateEmail 验证邮箱格式
func (r *RegisterRequest) validateEmail() error {
	email := strings.TrimSpace(r.Email)

	// 基本格式检查
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("邮箱格式不正确")
	}

	// 检查长度
	if len(email) > 100 {
		return errors.New("邮箱地址过长")
	}

	// 禁止临时邮箱域名
	tempDomains := []string{"10minutemail.com", "tempmail.org", "guerrillamail.com"}
	for _, domain := range tempDomains {
		if strings.HasSuffix(strings.ToLower(email), "@"+domain) {
			return errors.New("不允许使用临时邮箱")
		}
	}

	return nil
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email" binding:"omitempty,email,max=100"`
	Role     string `json:"role" binding:"omitempty,oneof=admin super_admin"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6,max=50"`
}

// UserListRequest 用户列表请求
type UserListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Username string `form:"username" binding:"omitempty,max=50"`
	Email    string `form:"email" binding:"omitempty,max=100"`
	Role     string `form:"role" binding:"omitempty,oneof=admin super_admin"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Access_token  string `json:"access_token"`
	Refresh_token string `json:"refresh_token"`
	Expires_in    int64  `json:"expires_in"`
	User          *User  `json:"user"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	Message string `json:"message" example:"注册成功"`
	User    *User  `json:"user"`
}

// RegisterConfig 注册配置
type RegisterConfig struct {
	Enabled         bool   `json:"enabled"`          // 是否开放注册
	DefaultRole     string `json:"default_role"`     // 默认角色
	RequireApproval bool   `json:"require_approval"` // 是否需要审核
	RateLimit       int    `json:"rate_limit"`       // 频率限制（每小时）
}

// SafeUser 安全的用户信息（别名，保持兼容性）
type SafeUser = User

// 用户角色常量
const (
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
)

// 用户状态常量
const (
	StatusDisabled = 0
	StatusEnabled  = 1
)
