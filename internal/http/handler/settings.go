package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/domain/setting"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

type SettingsManager interface {
	Get(ctx context.Context) (usecase.SettingsView, error)
	Update(ctx context.Context, input usecase.UpdateSettingsInput) error
	UpdateMonitorState(ctx context.Context, state string) error
}

type updateSettingsRequest struct {
	User      string `json:"user" form:"user"`
	Pass      string `json:"pass" form:"pass"`
	NotifyURL string `json:"notifyUrl" form:"notifyUrl"`
	ReturnURL string `json:"returnUrl" form:"returnUrl"`
	Key       string `json:"key" form:"key"`
	Close     string `json:"close" form:"close"`
	PayQf     string `json:"payQf" form:"payQf"`
	Wxpay     string `json:"wxpay" form:"wxpay"`
	Zfbpay    string `json:"zfbpay" form:"zfbpay"`
}

func getSettingsHandler(service SettingsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "设置服务不可用", nil))
			return
		}
		result, err := service.Get(c.Request.Context())
		if err != nil {
			writeSettingsError(c, err)
			return
		}

		values := result.Values
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"user":      result.Username,
			"pass":      "",
			"notifyUrl": values[setting.NotifyURLKey],
			"returnUrl": values[setting.ReturnURLKey],
			"key":       values[setting.MerchantKey],
			"lastheart": values[setting.LastHeartKey],
			"lastpay":   values[setting.LastPayKey],
			"jkstate":   values[setting.MonitorStateKey],
			"close":     values[setting.CloseMinutesKey],
			"payQf":     values[setting.PriceAdjustKey],
			"pay_qf":    values[setting.PriceAdjustKey],
			"wxpay":     values[setting.WechatPayURLKey],
			"zfbpay":    values[setting.AlipayPayURLKey],
		}))
	}
}

func getMonitorSettingsHandler(service SettingsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "设置服务不可用", nil))
			return
		}
		result, err := service.Get(c.Request.Context())
		if err != nil {
			writeSettingsError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"jkstate":   result.Values[setting.MonitorStateKey],
			"lastheart": result.Values[setting.LastHeartKey],
			"lastpay":   result.Values[setting.LastPayKey],
		}))
	}
}

func updateSettingsHandler(service SettingsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "设置服务不可用", nil))
			return
		}
		var request updateSettingsRequest
		if err := c.ShouldBind(&request); err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "设置参数无效", nil))
			return
		}
		if err := service.Update(c.Request.Context(), usecase.UpdateSettingsInput{
			Username:     request.User,
			Password:     request.Pass,
			NotifyURL:    request.NotifyURL,
			ReturnURL:    request.ReturnURL,
			MerchantKey:  request.Key,
			CloseMinutes: request.Close,
			PriceAdjust:  request.PayQf,
			WechatPayURL: request.Wxpay,
			AlipayPayURL: request.Zfbpay,
		}); err != nil {
			writeSettingsError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "保存成功", nil))
	}
}

func writeSettingsError(c *gin.Context, err error) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		c.JSON(http.StatusOK, php.NewEnvelope(500, "设置服务异常", nil))
		return
	}
	switch appError.Code {
	case usecase.CodeInvalidArgument:
		c.JSON(http.StatusOK, php.NewEnvelope(400, appError.Message, nil))
	case usecase.CodeConfiguration:
		c.JSON(http.StatusOK, php.NewEnvelope(503, appError.Message, nil))
	default:
		c.JSON(http.StatusOK, php.NewEnvelope(500, "设置服务异常", nil))
	}
}

func updateMonitorSettingsHandler(service SettingsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "设置服务不可用", nil))
			return
		}
		var request struct {
			JK string `json:"jk" form:"jk"`
		}
		if err := c.ShouldBind(&request); err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "参数格式错误", nil))
			return
		}
		if err := service.UpdateMonitorState(c.Request.Context(), request.JK); err != nil {
			writeSettingsError(c, err)
			return
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "设置成功", nil))
	}
}
