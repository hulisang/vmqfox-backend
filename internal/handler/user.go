package handler

import (
	"strconv"

	"vmqfox-api-go/internal/middleware"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表，支持分页和搜索
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param search query string false "搜索关键词"
// @Success 200 {object} response.PagedResponse
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v2/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 获取用户列表
	users, total, err := h.userService.GetUsers(page, limit, search)
	if err != nil {
		response.InternalError(c, "Failed to get users")
		return
	}

	// 计算总页数
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// 转换为安全用户信息
	safeUsers := make([]*model.SafeUser, len(users))
	for i, user := range users {
		safeUsers[i] = user.ToSafeUser()
	}

	// 返回分页响应
	response.SuccessPaged(c, safeUsers, response.PageMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// GetUser 获取单个用户
// @Summary 获取单个用户
// @Description 根据ID获取用户详情
// @Tags users
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=model.SafeUser}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ValidationFailed(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		if err == service.ErrUserNotFound {
			response.Error(c, response.CodeNotFound, "User not found")
			return
		}
		response.InternalError(c, "Failed to get user")
		return
	}

	response.Success(c, user.ToSafeUser())
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.CreateUserRequest true "用户信息"
// @Success 200 {object} response.Response{data=model.SafeUser}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v2/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 创建用户
	user, err := h.userService.CreateUser(&req)
	if err != nil {
		if err == service.ErrUserExists {
			response.Error(c, response.CodeUserExists)
			return
		}
		response.InternalError(c, "Failed to create user")
		return
	}

	response.SuccessWithMessage(c, "User created successfully", user.ToSafeUser())
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param user body model.UpdateUserRequest true "用户信息"
// @Success 200 {object} response.Response{data=model.SafeUser}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// 获取用户ID
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 更新用户
	user, err := h.userService.UpdateUser(uint(userID), &req)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		if err == service.ErrUserExists {
			response.Error(c, response.CodeUserExists)
			return
		}
		response.InternalError(c, "Failed to update user")
		return
	}

	response.SuccessWithMessage(c, "User updated successfully", user.ToSafeUser())
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 获取用户ID
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// 不能删除自己
	currentUserID := middleware.GetCurrentUserID(c)
	if uint(userID) == currentUserID {
		response.BadRequest(c, "Cannot delete yourself")
		return
	}

	// 删除用户
	err = h.userService.DeleteUser(uint(userID))
	if err != nil {
		if err == service.ErrUserNotFound {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		response.InternalError(c, "Failed to delete user")
		return
	}

	response.SuccessWithMessage(c, "User deleted successfully", nil)
}

// ResetPassword 重置密码
// @Summary 重置用户密码
// @Description 重置指定用户的密码
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param password body model.ResetPasswordRequest true "新密码"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v2/users/{id}/password [patch]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	// 获取用户ID
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req model.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 重置密码
	err = h.userService.ResetPassword(uint(userID), req.Password)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.Error(c, response.CodeUserNotFound)
			return
		}
		response.InternalError(c, "Failed to reset password")
		return
	}

	response.SuccessWithMessage(c, "Password reset successfully", nil)
}
