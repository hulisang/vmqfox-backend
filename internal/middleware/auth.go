package middleware

import (
	"strings"

	"vmqfox-api-go/pkg/jwt"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtManager *jwt.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Missing authorization header")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		// 提取token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			response.Unauthorized(c, "Missing token")
			c.Abort()
			return
		}

		// 验证token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			response.Error(c, response.CodeTokenInvalid, "Invalid token")
			c.Abort()
			return
		}

		// 检查token类型
		if claims.Type != "access" {
			response.Error(c, response.CodeTokenInvalid, "Invalid token type")
			c.Abort()
			return
		}

		// 检查用户状态
		if claims.Status != 1 {
			response.Error(c, response.CodeUserDisabled, "User is disabled")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_role", claims.Role)
		c.Set("user_status", claims.Status)

		c.Next()
	}
}

// RequireRole 角色权限中间件
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			response.Unauthorized(c, "User not authenticated")
			c.Abort()
			return
		}

		role := userRole.(string)

		// 超级管理员拥有所有权限
		if role == "super_admin" {
			c.Next()
			return
		}

		// 检查角色权限
		for _, requiredRole := range roles {
			if role == requiredRole {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

// RequireAdmin 需要管理员权限
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin", "super_admin")
}

// RequireSuperAdmin 需要超级管理员权限
func RequireSuperAdmin() gin.HandlerFunc {
	return RequireRole("super_admin")
}

// ConditionalAuthMiddleware 条件认证中间件
// 根据请求路径和参数决定是否需要认证
func ConditionalAuthMiddleware(jwtManager *jwt.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否是公开访问的订单查询
		if isPublicOrderAccess(c) {
			// 公开访问，设置公开访问标识
			c.Set("access_type", "public")
			c.Next()
			return
		}

		// 需要认证的访问，使用标准JWT认证
		authMiddleware := AuthMiddleware(jwtManager)
		authMiddleware(c)
	}
}

// isPublicOrderAccess 判断是否是公开的订单访问
func isPublicOrderAccess(c *gin.Context) bool {
	// 检查是否有 public=true 查询参数
	if c.Query("public") == "true" {
		return true
	}

	// 检查路径是否包含 /public 段
	if strings.Contains(c.Request.URL.Path, "/public") {
		return true
	}

	// 检查是否没有Authorization头（支付页面通常不会发送认证头）
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		// 进一步检查是否是订单相关的GET请求
		if c.Request.Method == "GET" &&
			(strings.Contains(c.Request.URL.Path, "/orders/") ||
				strings.Contains(c.Request.URL.Path, "/status")) {
			return true
		}
	}

	return false
}

// GetCurrentUserID 获取当前用户ID
func GetCurrentUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uint)
	}
	return 0
}

// GetCurrentUsername 获取当前用户名
func GetCurrentUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

// GetCurrentUserRole 获取当前用户角色
func GetCurrentUserRole(c *gin.Context) string {
	if role, exists := c.Get("user_role"); exists {
		return role.(string)
	}
	return ""
}

// IsSuperAdmin 检查是否为超级管理员
func IsSuperAdmin(c *gin.Context) bool {
	return GetCurrentUserRole(c) == "super_admin"
}
