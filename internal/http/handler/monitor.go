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
	HeartbeatAPI gin.HandlerFunc
	PushAPI      gin.HandlerFunc
}

func newMonitorHandlers(service MonitorManager) MonitorHandlers {
	return MonitorHandlers{
		HeartbeatAPI: heartbeatHandler(service),
		PushAPI:      paymentPushHandler(service),
	}
}

func heartbeatHandler(service MonitorManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeMonitorUnavailable(c)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeMonitorBindingError(c)
			return
		}
		if err := service.Heartbeat(c.Request.Context(), usecase.HeartbeatInput{
			Timestamp: params["t"],
			Sign:      params["sign"],
		}); err != nil {
			writeMonitorError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "心跳更新成功", nil))
	}
}

func paymentPushHandler(service MonitorManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			writeMonitorUnavailable(c)
			return
		}
		params, err := scalarRequestParams(c)
		if err != nil {
			writeMonitorBindingError(c)
			return
		}
		result, err := service.Push(c.Request.Context(), usecase.PaymentPushInput{
			Timestamp: params["t"],
			Type:      params["type"],
			Price:     params["price"],
			Sign:      params["sign"],
		})
		if err != nil {
			writeMonitorError(c, err)
			return
		}
		message := "成功"
		if result.Matched && !result.Duplicate {
			message = "订单支付成功"
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, message, nil))
	}
}

func writeMonitorUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "监控服务不可用", nil))
}

func writeMonitorBindingError(c *gin.Context) {
	c.JSON(http.StatusOK, php.NewEnvelope(400, "请求参数格式错误", nil))
}

func writeMonitorError(c *gin.Context, err error) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		c.JSON(http.StatusOK, php.NewEnvelope(500, "订单处理失败", nil))
		return
	}

	code := 400
	if appError.Code == usecase.CodeDependency {
		code = 500
	}
	c.JSON(http.StatusOK, php.NewEnvelope(code, appError.Message, nil))
}
