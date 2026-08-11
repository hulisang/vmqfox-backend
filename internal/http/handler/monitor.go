package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

type MonitorManager interface {
	Heartbeat(context.Context, usecase.HeartbeatInput) error
	Push(context.Context, usecase.PaymentPushInput) (usecase.PaymentPushResult, error)
}

type MonitorHandlers struct {
	HeartbeatAPI    gin.HandlerFunc
	PushAPI         gin.HandlerFunc
	HeartbeatLegacy gin.HandlerFunc
	PushLegacy      gin.HandlerFunc
}

func newMonitorHandlers(service MonitorManager) MonitorHandlers {
	return MonitorHandlers{
		HeartbeatAPI:    heartbeatHandler(service, ProtocolAPI, false),
		PushAPI:         paymentPushHandler(service, ProtocolAPI, false),
		HeartbeatLegacy: heartbeatHandler(service, ProtocolLegacy, true),
		PushLegacy:      paymentPushHandler(service, ProtocolLegacy, true),
	}
}

func heartbeatHandler(service MonitorManager, style ProtocolStyle, legacyCompatible bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeMonitorUnavailable(c, style)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeMonitorBindingError(c, style)
			return
		}
		if err := service.Heartbeat(c.Request.Context(), usecase.HeartbeatInput{
			Timestamp:        params["t"],
			Sign:             params["sign"],
			LegacyCompatible: legacyCompatible,
		}); err != nil {
			writeMonitorError(c, err, style)
			return
		}

		if style == ProtocolLegacy {
			c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "成功"})
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "心跳更新成功", nil))
	}
}

func paymentPushHandler(service MonitorManager, style ProtocolStyle, legacyCompatible bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeMonitorUnavailable(c, style)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeMonitorBindingError(c, style)
			return
		}
		result, err := service.Push(c.Request.Context(), usecase.PaymentPushInput{
			Timestamp:        params["t"],
			Type:             params["type"],
			Price:            params["price"],
			Sign:             params["sign"],
			LegacyCompatible: legacyCompatible,
		})
		if err != nil {
			writeMonitorError(c, err, style)
			return
		}

		if style == ProtocolLegacy {
			c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "成功"})
			return
		}
		message := "成功"
		if result.Matched && !result.Duplicate {
			message = "订单支付成功"
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, message, nil))
	}
}

func writeMonitorUnavailable(c *gin.Context, style ProtocolStyle) {
	if style == ProtocolLegacy {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": -1, "msg": "监控服务不可用"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "监控服务不可用", nil))
}

func writeMonitorBindingError(c *gin.Context, style ProtocolStyle) {
	if style == ProtocolLegacy {
		c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "请求参数格式错误"})
		return
	}
	c.JSON(http.StatusOK, php.NewEnvelope(400, "请求参数格式错误", nil))
}

func writeMonitorError(c *gin.Context, err error, style ProtocolStyle) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		if style == ProtocolLegacy {
			c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "订单处理失败"})
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(500, "订单处理失败", nil))
		return
	}

	if style == ProtocolLegacy {
		c.JSON(http.StatusOK, gin.H{"code": -1, "msg": appError.Message})
		return
	}
	code := 400
	if appError.Code == usecase.CodeDependency {
		code = 500
	}
	c.JSON(http.StatusOK, php.NewEnvelope(code, appError.Message, nil))
}
