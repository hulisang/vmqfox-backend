package model

import (
	"time"
)

// PaymentOrderRequest 支付页面订单请求
type PaymentOrderRequest struct {
	OrderID string `uri:"order_id" binding:"required"`
}

// PaymentOrderResponse 支付页面订单响应
type PaymentOrderResponse struct {
	Order_id         string  `json:"orderId"`
	Pay_id           string  `json:"payId"`
	Type             int     `json:"payType"`
	Type_text        string  `json:"type_text"`
	Price            float64 `json:"price"`
	Really_price     float64 `json:"reallyPrice"`
	Pay_url          string  `json:"payUrl"`
	State            int     `json:"state"`
	State_text       string  `json:"stateText"`
	Is_auto          int     `json:"isAuto"`
	Create_date      int64   `json:"date"`
	Pay_date         int64   `json:"pay_date"`
	Close_date       int64   `json:"close_date"`
	Is_expired       bool    `json:"is_expired"`
	Expired_at       int64   `json:"expired_at"`
	TimeOut          int     `json:"timeOut"`          // 超时时间（分钟）
	RemainingSeconds int     `json:"remainingSeconds"` // 剩余秒数
	Return_url       string  `json:"return_url"`       // 商户返回URL
	Param            string  `json:"param"`            // 自定义参数
}

// PaymentStatusRequest 支付状态检查请求
type PaymentStatusRequest struct {
	OrderID string `uri:"order_id" binding:"required"`
}

// PaymentStatusResponse 支付状态检查响应
type PaymentStatusResponse struct {
	Order_id         string `json:"order_id"`
	State            int    `json:"state"`
	State_text       string `json:"state_text"`
	Is_paid          bool   `json:"is_paid"`
	Is_expired       bool   `json:"is_expired"`
	Pay_date         int64  `json:"pay_date"`
	Message          string `json:"message"`
	RemainingSeconds int    `json:"remainingSeconds"`
}

// PaymentQrcodeRequest 支付页面二维码生成请求
type PaymentQrcodeRequest struct {
	URL  string `form:"url" binding:"required,url"`
	Size int    `form:"size" binding:"omitempty,min=100,max=1000"`
}

// PaymentQrcodeResponse 支付页面二维码生成响应
type PaymentQrcodeResponse struct {
	Qrcode_url string `json:"qrcode_url"`
	Size       int    `json:"size"`
	Format     string `json:"format"`
}

// ToPaymentResponse 转换为支付页面响应格式（使用默认30分钟过期时间）
func (o *Order) ToPaymentResponse() *PaymentOrderResponse {
	// 计算过期时间（创建时间 + 30分钟）
	expiredAt := o.Create_date + 30*60
	isExpired := o.State == OrderStatusPending && expiredAt < getCurrentTimestamp()

	return &PaymentOrderResponse{
		Order_id:     o.Order_id,
		Pay_id:       o.Pay_id,
		Type:         o.Type,
		Type_text:    o.GetTypeText(),
		Price:        o.Price,
		Really_price: o.Really_price,
		Pay_url:      o.Pay_url,
		State:        o.State,
		State_text:   o.GetStatusText(),
		Is_auto:      o.Is_auto,
		Create_date:  o.Create_date,
		Pay_date:     o.Pay_date,
		Close_date:   o.Close_date,
		Is_expired:   isExpired,
		Expired_at:   expiredAt,
	}
}

// ToPaymentResponseWithExpireTime 转换为支付页面响应格式（使用指定的过期时间）
func (o *Order) ToPaymentResponseWithExpireTime(expireMinutes int) *PaymentOrderResponse {
	// 计算过期时间（创建时间 + 指定分钟数）
	expiredAt := o.Create_date + int64(expireMinutes*60)
	isExpired := o.State == OrderStatusPending && expiredAt < getCurrentTimestamp()

	return &PaymentOrderResponse{
		Order_id:     o.Order_id,
		Pay_id:       o.Pay_id,
		Type:         o.Type,
		Type_text:    o.GetTypeText(),
		Price:        o.Price,
		Really_price: o.Really_price,
		Pay_url:      o.Pay_url,
		State:        o.State,
		State_text:   o.GetStatusText(),
		Is_auto:      o.Is_auto,
		Create_date:  o.Create_date,
		Pay_date:     o.Pay_date,
		Close_date:   o.Close_date,
		Is_expired:   isExpired,
		Expired_at:   expiredAt,
	}
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
