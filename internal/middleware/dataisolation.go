package middleware

import (
	"github.com/gin-gonic/gin"
)

// DataIsolationMiddleware 数据隔离中间件
// 根据用户角色设置数据访问范围，实现多用户数据隔离
func DataIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			// 如果没有用户角色信息，跳过数据隔离
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			// 如果没有用户ID信息，跳过数据隔离
			c.Next()
			return
		}

		// 设置数据隔离上下文
		if userRole.(string) == "super_admin" {
			// 超级管理员可以访问所有数据
			c.Set("data_scope", "all")
		} else {
			// 普通管理员只能访问自己的数据
			c.Set("data_scope", "user")
			c.Set("scope_user_id", userID.(uint))
		}

		c.Next()
	}
}

// GetDataScope 获取当前用户的数据访问范围
// 返回值: "all" - 可访问所有数据, "user" - 只能访问自己的数据
func GetDataScope(c *gin.Context) string {
	if scope, exists := c.Get("data_scope"); exists {
		return scope.(string)
	}
	return "user" // 默认为用户级别，安全优先
}

// GetScopeUserID 获取数据隔离的用户ID
// 当数据范围为"user"时，返回应该过滤的用户ID
func GetScopeUserID(c *gin.Context) uint {
	if userID, exists := c.Get("scope_user_id"); exists {
		return userID.(uint)
	}
	return GetCurrentUserID(c) // 默认返回当前用户ID
}

// ShouldFilterByUser 检查是否需要按用户过滤数据
// 返回true表示需要按用户ID过滤数据
func ShouldFilterByUser(c *gin.Context) bool {
	return GetDataScope(c) == "user"
}

// CheckDataAccess 检查用户是否有权限访问指定用户的数据
// targetUserID: 目标数据的用户ID
// 返回true表示有权限访问
func CheckDataAccess(c *gin.Context, targetUserID uint) bool {
	// 超级管理员可以访问所有数据
	if GetDataScope(c) == "all" {
		return true
	}

	// 普通管理员只能访问自己的数据
	currentUserID := GetScopeUserID(c)
	return currentUserID == targetUserID
}

// ApplyUserFilter 为查询请求应用用户过滤
// 如果需要数据隔离，自动设置user_id过滤条件
func ApplyUserFilter(c *gin.Context, userIDPtr **uint) {
	if ShouldFilterByUser(c) {
		scopeUserID := GetScopeUserID(c)
		*userIDPtr = &scopeUserID
	}
	// 如果是超级管理员，不设置过滤条件，保持原有的userIDPtr值
}

// ValidateResourceAccess 验证用户是否有权限访问指定资源
// 通常用于单个资源的访问控制（如获取、更新、删除特定订单）
func ValidateResourceAccess(c *gin.Context, resourceUserID uint) bool {
	return CheckDataAccess(c, resourceUserID)
}

// GetIsolationInfo 获取当前的数据隔离信息（用于调试和日志）
func GetIsolationInfo(c *gin.Context) map[string]interface{} {
	return map[string]interface{}{
		"data_scope":    GetDataScope(c),
		"scope_user_id": GetScopeUserID(c),
		"current_user":  GetCurrentUserID(c),
		"user_role":     GetCurrentUserRole(c),
	}
}
