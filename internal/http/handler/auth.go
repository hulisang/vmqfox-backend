package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hulisang/vmqfox-backend/internal/compat/php"
	"github.com/hulisang/vmqfox-backend/internal/usecase"
)

type Authenticator interface {
	Login(ctx context.Context, input usecase.LoginInput) (usecase.LoginResult, error)
}

type loginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func loginHandler(service Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "认证服务不可用", nil))
			return
		}

		var request loginRequest
		if err := c.ShouldBind(&request); err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(400, "用户名或密码不能为空", nil))
			return
		}
		result, err := service.Login(c.Request.Context(), usecase.LoginInput{
			Username: request.Username,
			Password: request.Password,
		})
		if err != nil {
			writeLoginError(c, err)
			return
		}

		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"accessToken": result.AccessToken,
			"username":    result.Username,
			"expiresAt":   result.ExpiresAt.Unix(),
		}))
	}
}

func logoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, php.NewEnvelope(200, "退出成功", nil))
	}
}

func writeLoginError(c *gin.Context, err error) {
	var appError *usecase.Error
	if !errors.As(err, &appError) {
		c.JSON(http.StatusOK, php.NewEnvelope(500, "认证服务异常", nil))
		return
	}

	switch appError.Code {
	case usecase.CodeInvalidArgument:
		c.JSON(http.StatusOK, php.NewEnvelope(400, appError.Message, nil))
	case usecase.CodeInvalidCredentials:
		c.JSON(http.StatusOK, php.NewEnvelope(400, appError.Message, nil))
	case usecase.CodeConfiguration:
		c.JSON(http.StatusOK, php.NewEnvelope(503, appError.Message, nil))
	default:
		c.JSON(http.StatusOK, php.NewEnvelope(500, "认证服务异常", nil))
	}
}

func userInfoHandler(settings SettingsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settings == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "设置服务不可用", nil))
			return
		}
		result, err := settings.Get(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(500, "获取用户信息失败", nil))
			return
		}

		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"userId":      1,
			"username":    result.Username,
			"userName":    result.Username,
			"realName":    "V免签管理员",
			"avatar":      "",
			"roles":       []string{"admin"},
			"permissions": []string{"dashboard", "order", "qrcode", "setting", "monitor"},
			"buttons":     []string{"dashboard", "order", "qrcode", "setting", "monitor"},
		}))
	}
}

func userListHandler(settings SettingsManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settings == nil {
			c.JSON(http.StatusServiceUnavailable, php.NewEnvelope(503, "设置服务不可用", nil))
			return
		}
		result, err := settings.Get(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusOK, php.NewEnvelope(500, "获取用户列表失败", nil))
			return
		}

		list := []gin.H{
			{
				"userId":     1,
				"username":   result.Username,
				"realName":   "V免签管理员",
				"status":     1,
				"createTime": time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		c.JSON(http.StatusOK, php.NewEnvelope(200, "成功", gin.H{
			"list":  list,
			"total": len(list),
		}))
	}
}

func menuHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		menu := []gin.H{
			{
				"name": "系统设置",
				"type": "url",
				"url":  "admin/setting.html",
			},
			{
				"name": "监控端设置",
				"type": "url",
				"url":  "admin/jk.html",
			},
			{
				"name": "微信二维码",
				"type": "menu",
				"node": []gin.H{
					{
						"name": "添加",
						"type": "url",
						"url":  "admin/addwxqrcode.html",
					},
					{
						"name": "管理",
						"type": "url",
						"url":  "admin/wxqrcodelist.html",
					},
				},
			},
			{
				"name": "支付宝二维码",
				"type": "menu",
				"node": []gin.H{
					{
						"name": "添加",
						"type": "url",
						"url":  "admin/addzfbqrcode.html",
					},
					{
						"name": "管理",
						"type": "url",
						"url":  "admin/zfbqrcodelist.html",
					},
				},
			},
			{
				"name": "订单列表",
				"type": "url",
				"url":  "admin/orderlist.html",
			},
			{
				"name": "Api说明",
				"type": "url",
				"url":  "api.html",
			},
		}
		c.JSON(http.StatusOK, menu)
	}
}
