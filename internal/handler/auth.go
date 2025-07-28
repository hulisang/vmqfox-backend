package handler

import (
	"vmqfox-api-go/internal/middleware"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取访问令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param login body model.LoginRequest true "登录信息"
// @Success 200 {object} response.Response{data=model.LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v2/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 执行登录
	loginResp, err := h.authService.Login(&req)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			response.Error(c, response.CodeInvalidPassword, "Invalid username or password")
			return
		case service.ErrUserDisabled:
			response.Error(c, response.CodeUserDisabled)
			return
		default:
			response.InternalError(c, "Login failed")
			return
		}
	}

	response.SuccessWithMessage(c, "Login successful", loginResp)
}

// Register 用户注册
// @Summary 用户注册
// @Description 用户注册创建新账户
// @Tags auth
// @Accept json
// @Produce json
// @Param register body model.RegisterRequest true "注册信息"
// @Success 200 {object} response.Response{data=model.RegisterResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v2/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 获取客户端IP
	clientIP := c.ClientIP()

	// 调用注册服务
	registerResp, err := h.authService.Register(&req, clientIP)
	if err != nil {
		// 根据错误类型返回不同的响应
		switch err.Error() {
		case "用户名已存在":
			response.Error(c, response.CodeUserExists, "Username already exists")
			return
		case "邮箱已存在":
			response.Error(c, response.CodeUserExists, "Email already exists")
			return
		case "密码和确认密码不匹配":
			response.ValidationFailed(c, "Password and confirm password do not match")
			return
		case "用户注册功能已关闭":
			response.Error(c, response.CodeForbidden, "User registration is disabled")
			return
		case "注册频率过高，请稍后再试":
			response.Error(c, response.CodeBadRequest, "Registration rate limit exceeded")
			return
		default:
			response.InternalError(c, "Registration failed")
			return
		}
	}

	response.SuccessWithMessage(c, "Registration successful", registerResp)
}

// RefreshToken 刷新令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body object{refresh_token=string} true "刷新令牌"
// @Success 200 {object} response.Response{data=model.LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v2/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 刷新令牌
	loginResp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			response.Error(c, response.CodeTokenInvalid)
			return
		case service.ErrUserDisabled:
			response.Error(c, response.CodeUserDisabled)
			return
		case service.ErrInvalidCredentials:
			response.Error(c, response.CodeUserNotFound, "User not found")
			return
		default:
			response.InternalError(c, "Token refresh failed")
			return
		}
	}

	response.SuccessWithMessage(c, "Token refreshed successfully", loginResp)
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=model.SafeUser}
// @Failure 401 {object} response.Response
// @Router /api/v2/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.authService.GetCurrentUser(userID)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		response.InternalError(c, "Failed to get user info")
		return
	}

	response.Success(c, user)
}

// Logout 用户注销
// @Summary 用户注销
// @Description 注销当前用户登录状态
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v2/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := h.authService.Logout(userID)
	if err != nil {
		response.InternalError(c, "Logout failed")
		return
	}

	response.SuccessWithMessage(c, "Logout successful", nil)
}
