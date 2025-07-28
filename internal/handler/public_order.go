package handler

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vmqfox-api-go/internal/config"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PublicOrderHandler 公开订单处理器（供第三方商户使用）
type PublicOrderHandler struct {
	orderService   service.OrderService
	settingService service.SettingService
	qrcodeService  service.QrcodeService
	db             *gorm.DB
}

// NewPublicOrderHandler 创建公开订单处理器
func NewPublicOrderHandler(orderService service.OrderService, settingService service.SettingService, qrcodeService service.QrcodeService, db *gorm.DB) *PublicOrderHandler {
	return &PublicOrderHandler{
		orderService:   orderService,
		settingService: settingService,
		qrcodeService:  qrcodeService,
		db:             db,
	}
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	PayID     string  `json:"payId" form:"payId" binding:"required"` // 商户订单号
	Param     string  `json:"param" form:"param"`                    // 自定义参数
	Type      int     `json:"type" form:"type" binding:"required"`   // 支付类型：1=微信，2=支付宝
	Price     float64 `json:"price" form:"price" binding:"required"` // 订单金额
	Sign      string  `json:"sign" form:"sign" binding:"required"`   // 签名
	NotifyURL string  `json:"notifyUrl" form:"notifyUrl"`            // 异步通知地址
	ReturnURL string  `json:"returnUrl" form:"returnUrl"`            // 同步返回地址
	IsHTML    int     `json:"isHtml" form:"isHtml"`                  // 是否返回HTML：1=返回HTML跳转页面，0=返回JSON
}

// CreateOrder 创建订单（公开API，供第三方商户使用）
// @Summary 创建订单（第三方商户）
// @Description 第三方商户创建支付订单，使用签名验证
// @Tags 公开API
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "创建订单请求"
// @Success 200 {object} response.Response{data=object}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/public/order [post]
func (h *PublicOrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest

	// 支持JSON和表单数据两种格式
	contentType := c.GetHeader("Content-Type")
	var err error
	if strings.Contains(contentType, "application/json") {
		err = c.ShouldBindJSON(&req)
	} else {
		err = c.ShouldBind(&req)
	}

	if err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	// 验证支付类型
	if req.Type != 1 && req.Type != 2 {
		response.ValidationFailed(c, "支付类型错误")
		return
	}

	// 验证价格
	if req.Price <= 0 {
		response.ValidationFailed(c, "价格错误")
		return
	}

	// 通过AppID识别用户ID
	userID, err := h.identifyUserByAppID(req.Param)
	if err != nil {
		response.ValidationFailed(c, "无效的商户ID")
		return
	}

	// 获取该用户的密钥
	key, err := h.settingService.GetUserSettingValue(userID, "key")
	if err != nil || key == "" {
		response.Error(c, response.CodeInternalError, "商户密钥配置错误")
		return
	}

	// 验证签名
	paramValue := req.Param
	if paramValue == "" {
		paramValue = ""
	}
	// 修复价格格式化问题：使用与PHP插件一致的格式
	// PHP插件直接使用原始价格值，这里使用固定2位小数格式
	priceStr := fmt.Sprintf("%.2f", req.Price)

	signStr := fmt.Sprintf("payId=%s&param=%s&type=%d&price=%s&key=%s",
		req.PayID, paramValue, req.Type, priceStr, key)
	expectedSign := fmt.Sprintf("%x", md5.Sum([]byte(signStr)))

	if expectedSign != req.Sign {
		response.ValidationFailed(c, "签名错误")
		return
	}

	// 检查该用户的监控端状态
	jkstate, err := h.settingService.GetUserSettingValue(userID, "jkstate")
	if err != nil || jkstate != "1" {
		response.Error(c, response.CodeBadRequest, "监控端状态异常，请检查")
		return
	}

	// 检查商户订单号是否已存在
	existingOrder, _ := h.orderService.GetOrderByPayID(req.PayID)
	if existingOrder != nil {
		response.ValidationFailed(c, "商户订单号已存在，请勿重复提交")
		return
	}

	// 生成订单号
	orderID := fmt.Sprintf("%s%05d", time.Now().Format("20060102150405"), rand.Intn(100000))

	// 金额处理和二维码匹配逻辑
	reallyPrice, payURL, isAuto, err := h.processOrderAmount(req.Type, req.Price, orderID)
	if err != nil {
		response.Error(c, response.CodeBadRequest, err.Error())
		return
	}

	// 获取回调地址
	finalNotifyURL := req.NotifyURL
	if finalNotifyURL == "" {
		finalNotifyURL, _ = h.settingService.GetSettingValue("notifyUrl")
	}

	finalReturnURL := req.ReturnURL
	if finalReturnURL == "" {
		finalReturnURL, _ = h.settingService.GetSettingValue("returnUrl")
	}

	// 创建订单数据
	orderData := &model.Order{
		Pay_id:       req.PayID,
		Order_id:     orderID,
		Create_date:  time.Now().Unix(),
		Type:         req.Type,
		Price:        req.Price,
		Really_price: reallyPrice,
		State:        0, // 未支付
		Param:        paramValue,
		Pay_url:      payURL,
		Is_auto:      isAuto,
		Notify_url:   finalNotifyURL,
		Return_url:   finalReturnURL,
		Pay_date:     0,
		Close_date:   0,
		User_id:      userID, // 使用AppID识别出的用户ID
	}

	// 保存订单
	_, err = h.orderService.CreatePublicOrder(orderData)
	if err != nil {
		response.InternalError(c, "创建订单失败")
		return
	}

	// 如果isHtml=1，返回HTML跳转页面
	if req.IsHTML == 1 {
		// 从配置文件获取前端URL
		frontendURL := config.AppConfig.Server.FrontendURL
		if frontendURL == "" {
			frontendURL = "http://localhost:3000" // 默认前端地址
		}

		// 构建跳转URL
		redirectURL := fmt.Sprintf("%s/#/payment/%s", frontendURL, orderID)

		htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>正在跳转到支付页面...</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <style>
        body {
            background-color: #f5f5f5;
            font-family: Arial, sans-serif;
            text-align: center;
            padding-top: 100px;
        }
        .loading {
            display: inline-block;
            width: 50px;
            height: 50px;
            border: 3px solid rgba(0,0,0,.3);
            border-radius: 50%%;
            border-top-color: #333;
            animation: spin 1s ease-in-out infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        .text {
            margin-top: 20px;
            color: #333;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="loading"></div>
    <div class="text">正在跳转到支付页面，请稍候...</div>
    <script>
        window.location.href = "%s";
    </script>
</body>
</html>`, redirectURL)

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, htmlContent)
		return
	}

	// 返回JSON格式的订单信息
	// 从配置文件获取前端URL
	frontendURL := config.AppConfig.Server.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	response.Success(c, gin.H{
		"payId":       req.PayID,
		"orderId":     orderID,
		"payType":     req.Type,
		"price":       req.Price,
		"reallyPrice": reallyPrice,
		"payUrl":      payURL,
		"isAuto":      isAuto,
		"redirectUrl": fmt.Sprintf("%s/#/payment/%s", frontendURL, orderID),
	})
}

// processOrderAmount 处理订单金额和二维码匹配逻辑
func (h *PublicOrderHandler) processOrderAmount(payType int, price float64, orderID string) (float64, string, int, error) {
	// 使用分进行金额计算，避免精度问题
	reallyPriceCent := int(price * 100)

	// 获取payQf配置，用于金额递增/递减
	payQf, _ := h.settingService.GetSettingValue("payQf")

	// 使用tmp_price表避免金额冲突，最多尝试10次
	var payURL string
	isAuto := 1 // 默认为自动模式
	var finalReallyPrice float64

	ok := false
	for i := 0; i < 10; i++ {
		reallyPrice := float64(reallyPriceCent) / 100
		tmpPrice := fmt.Sprintf("%d-%d", reallyPriceCent, payType) // 格式：金额分-支付类型

		// 尝试插入tmp_price表，使用原子性操作避免金额冲突
		tmpPriceRecord := &model.TmpPrice{
			Price: tmpPrice,
			Oid:   orderID,
		}

		// 使用数据库的唯一约束来确保原子性
		err := h.db.Create(tmpPriceRecord).Error
		if err == nil {
			// 成功插入，表示这个金额可用
			ok = true
			finalReallyPrice = reallyPrice

			// 查找与金额匹配的二维码
			qrcode, qrErr := h.qrcodeService.GetQrcodeByPriceAndType(reallyPrice, payType)
			if qrErr == nil && qrcode != nil {
				payURL = qrcode.Pay_url
				isAuto = 0 // 找到匹配的二维码，使用固定模式
			}
			break
		}

		// 如果插入失败（金额冲突），根据配置微调价格
		if payQf == "1" { // 金额递增模式
			reallyPriceCent++
		} else if payQf == "2" { // 金额递减模式
			reallyPriceCent--
		} else {
			// 不调整金额，直接使用第一次尝试的金额
			finalReallyPrice = reallyPrice
			ok = true
			break
		}
	}

	if !ok {
		return 0, "", 0, fmt.Errorf("订单超出负荷，请稍后重试")
	}

	// 如果没有找到匹配的二维码，使用通用收款码
	if payURL == "" {
		settingKey := "zfbpay"
		if payType == 1 {
			settingKey = "wxpay"
		}

		var err error
		payURL, err = h.settingService.GetSettingValue(settingKey)
		if err != nil || payURL == "" {
			payMethodName := "支付宝"
			if payType == 1 {
				payMethodName = "微信"
			}
			return 0, "", 0, fmt.Errorf("暂无可用支付二维码，请在后台【系统设置】或【%s二维码】中配置", payMethodName)
		}
	}

	return finalReallyPrice, payURL, isAuto, nil
}

// GetOrder 获取订单详情（公开API，供支付页面使用）
// @Summary 获取订单详情
// @Description 获取订单详情，供支付页面显示
// @Tags 公开API
// @Produce json
// @Param order_id path string true "订单ID"
// @Success 200 {object} response.Response{data=object}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/public/order/{order_id} [get]
func (h *PublicOrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		response.ValidationFailed(c, "订单号不能为空")
		return
	}

	// 查询订单
	order, err := h.orderService.GetOrderByOrderID(orderID)
	if err != nil {
		response.NotFound(c, "订单不存在")
		return
	}

	// 获取关闭时间配置
	closeTimeSetting, _ := h.settingService.GetSettingValue("close")
	closeTime := 5 // 默认5分钟
	if closeTimeSetting != "" {
		if ct, err := strconv.Atoi(closeTimeSetting); err == nil && ct > 0 {
			closeTime = ct
		}
	}

	// 计算剩余时间
	timeoutSeconds := closeTime * 60
	elapsedSeconds := int(time.Now().Unix() - order.Create_date)
	remainingSeconds := timeoutSeconds - elapsedSeconds
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	// 返回订单详情
	response.Success(c, gin.H{
		"payId":            order.Pay_id,
		"orderId":          order.Order_id,
		"payType":          order.Type,
		"price":            order.Price,
		"reallyPrice":      order.Really_price,
		"payUrl":           order.Pay_url,
		"isAuto":           order.Is_auto,
		"state":            order.State,
		"stateText":        h.getOrderStateText(order.State),
		"timeOut":          closeTime,
		"date":             order.Create_date,
		"remainingSeconds": remainingSeconds,
		"return_url":       order.Return_url,
		"param":            order.Param,
	})
}

// CheckOrderStatus 检查订单状态（公开API）
// @Summary 检查订单状态
// @Description 检查订单支付状态，供支付页面轮询使用
// @Tags 公开API
// @Produce json
// @Param order_id path string true "订单ID"
// @Success 200 {object} response.Response{data=object}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/public/order/{order_id}/status [get]
func (h *PublicOrderHandler) CheckOrderStatus(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		response.ValidationFailed(c, "订单号不能为空")
		return
	}

	// 查询订单
	order, err := h.orderService.GetOrderByOrderID(orderID)
	if err != nil {
		response.NotFound(c, "订单不存在")
		return
	}

	// 获取订单超时时间设置
	closeTimeSetting, _ := h.settingService.GetSettingValue("close")
	closeTime := 5 // 默认5分钟
	if closeTimeSetting != "" {
		if ct, err := strconv.Atoi(closeTimeSetting); err == nil && ct > 0 {
			closeTime = ct
		}
	}

	timeoutSeconds := closeTime * 60
	elapsedSeconds := int(time.Now().Unix() - order.Create_date)
	remainingSeconds := timeoutSeconds - elapsedSeconds
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	// 根据订单状态返回不同响应
	switch order.State {
	case 1: // 已支付
		response.Success(c, gin.H{
			"redirectUrl":      order.Return_url,
			"remainingSeconds": 0,
			"return_url":       order.Return_url,
			"param":            order.Param,
		})

	case -1: // 已关闭/过期
		response.Success(c, gin.H{
			"state":            -1,
			"remainingSeconds": 0,
			"return_url":       order.Return_url,
			"param":            order.Param,
		})

	default: // 未支付
		// 检查是否过期但状态未更新
		if elapsedSeconds > timeoutSeconds {
			// 订单已过期，更新状态
			err := h.orderService.CloseExpiredOrder(order.Order_id)
			if err != nil {
				// 记录错误但继续返回过期状态
			}

			response.Success(c, gin.H{
				"state":            -1,
				"remainingSeconds": 0,
				"return_url":       order.Return_url,
				"param":            order.Param,
			})
		} else {
			// 订单未支付，返回剩余时间
			response.Success(c, gin.H{
				"state":            0,
				"remainingSeconds": remainingSeconds,
				"return_url":       order.Return_url,
				"param":            order.Param,
			})
		}
	}
}

// getOrderStateText 获取订单状态文本
func (h *PublicOrderHandler) getOrderStateText(state int) string {
	switch state {
	case model.OrderStatusClosed:
		return "已关闭"
	case model.OrderStatusPending:
		return "未支付"
	case model.OrderStatusPaid:
		return "已支付"
	case model.OrderStatusNotifyFailed:
		return "通知失败"
	default:
		return "未知状态"
	}
}

// identifyUserByAppID 通过AppID识别用户ID
func (h *PublicOrderHandler) identifyUserByAppID(appId string) (uint, error) {
	if appId == "" {
		return 0, fmt.Errorf("AppID不能为空")
	}

	// 通过AppID查找用户ID
	userID, err := h.settingService.GetUserIDByAppID(appId)
	if err != nil {
		return 0, fmt.Errorf("无效的商户ID: %s", appId)
	}

	return userID, nil
}
