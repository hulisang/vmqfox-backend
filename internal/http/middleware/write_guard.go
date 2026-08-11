package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
)

type RuntimeMode string

const (
	RuntimeShadow RuntimeMode = "shadow"
	RuntimeWriter RuntimeMode = "writer"
)

// WriteGuard 按 HTTP 方法拒绝只读模式下的写请求。
func WriteGuard(mode RuntimeMode) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mode == RuntimeWriter || !isMutating(c.Request.Method) {
			c.Next()
			return
		}
		rejectShadowWrite(c)
	}
}

// WriteGuardAlways 用于 GET 也会改变状态的历史协议，例如监控心跳和推送。
func WriteGuardAlways(mode RuntimeMode) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mode == RuntimeWriter {
			c.Next()
			return
		}
		rejectShadowWrite(c)
	}
}

func rejectShadowWrite(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusConflict, php.NewEnvelope(409, "Go 服务处于只读模式，暂不接受写入", nil))
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
