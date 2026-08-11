package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/auth"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
)

type TokenParser interface {
	ParseAuthorization(value string) (auth.Identity, error)
}

func TokenAuth(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		if parser == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "认证服务不可用", nil))
			return
		}

		identity, err := parser.ParseAuthorization(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, php.NewEnvelope(401, "令牌无效，请重新登录", nil))
			return
		}

		c.Request = c.Request.WithContext(auth.WithIdentity(c.Request.Context(), identity))
		c.Next()
	}
}
