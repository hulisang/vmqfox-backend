package handler

import (
	"encoding/base64"
	"fmt"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/service"
	"vmqfox-api-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// PaymentHandler 支付页面处理器
type PaymentHandler struct {
	paymentService service.PaymentService
}

// NewPaymentHandler 创建支付页面处理器
func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// GetPaymentOrder 获取支付页面订单信息
// @Summary 获取支付页面订单信息
// @Description 获取支付页面的订单详细信息
// @Tags payment
// @Produce json
// @Param order_id path string true "订单ID"
// @Success 200 {object} response.Response{data=model.PaymentOrderResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/public/orders/{order_id} [get]
func (h *PaymentHandler) GetPaymentOrder(c *gin.Context) {
	var req model.PaymentOrderRequest
	if err := c.ShouldBindUri(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	order, err := h.paymentService.GetPaymentOrder(req.OrderID)
	if err != nil {
		if err == service.ErrPaymentOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		}
		response.InternalError(c, "Failed to get payment order")
		return
	}

	response.Success(c, order)
}

// CheckPaymentStatus 检查支付状态
// @Summary 检查支付状态
// @Description 检查订单的支付状态
// @Tags payment
// @Produce json
// @Param order_id path string true "订单ID"
// @Success 200 {object} response.Response{data=model.PaymentStatusResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/public/orders/{order_id}/status [get]
func (h *PaymentHandler) CheckPaymentStatus(c *gin.Context) {
	var req model.PaymentStatusRequest
	if err := c.ShouldBindUri(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	status, err := h.paymentService.CheckPaymentStatus(req.OrderID)
	if err != nil {
		if err == service.ErrPaymentOrderNotFound {
			response.Error(c, response.CodeNotFound, "Order not found")
			return
		}
		response.InternalError(c, "Failed to check payment status")
		return
	}

	response.Success(c, status)
}

// GeneratePaymentQrcode 生成支付页面二维码
// @Summary 生成支付页面二维码
// @Description 为支付页面生成二维码图片，直接返回PNG图片数据
// @Tags payment
// @Produce image/png
// @Param url query string true "二维码内容URL"
// @Param size query int false "二维码尺寸" default(300)
// @Success 200 {file} binary "PNG图片数据"
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/public/qrcode/generate [get]
func (h *PaymentHandler) GeneratePaymentQrcode(c *gin.Context) {
	var req model.PaymentQrcodeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationFailed(c, err.Error())
		return
	}

	qrcode, err := h.paymentService.GeneratePaymentQrcode(&req)
	if err != nil {
		response.Error(c, response.CodeBadRequest, "Failed to generate qrcode: "+err.Error())
		return
	}

	// 从base64数据URL中提取PNG数据
	if qrcode.Qrcode_url != "" && len(qrcode.Qrcode_url) > 22 { // "data:image/png;base64," 长度为22
		base64Data := qrcode.Qrcode_url[22:] // 去掉 "data:image/png;base64," 前缀

		// 解码base64数据
		imageData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			response.Error(c, response.CodeInternalError, "Failed to decode qrcode image")
			return
		}

		// 设置响应头并返回图片数据
		c.Header("Content-Type", "image/png")
		c.Header("Content-Length", fmt.Sprintf("%d", len(imageData)))
		c.Data(200, "image/png", imageData)
		return
	}

	response.Error(c, response.CodeInternalError, "Invalid qrcode data")
}

// GenerateReturnURL 生成返回URL（公开API，使用订单号）
func (h *PaymentHandler) GenerateReturnURL(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		response.BadRequest(c, "订单号不能为空")
		return
	}

	// 生成返回URL
	returnURL, err := h.paymentService.GenerateReturnURL(orderID)
	if err != nil {
		response.Error(c, response.CodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"returnUrl": returnURL,
	})
}
