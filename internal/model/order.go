package model

import (
	"time"

	"gorm.io/gorm"
)

// Order 订单模型 - 直接使用数据库字段名
type Order struct {
	Id           uint    `json:"id" gorm:"primarykey"`
	User_id      uint    `json:"user_id" gorm:"not null;default:1"`
	Order_id     string  `json:"order_id" gorm:"size:255"`
	Type         int     `json:"type" gorm:"not null"` // 1=微信, 2=支付宝
	Price        float64 `json:"price" gorm:"not null"`
	Really_price float64 `json:"really_price" gorm:"not null"`
	State        int     `json:"state" gorm:"not null"` // -1=已关闭, 0=待支付, 1=已支付, 2=通知失败
	Notify_url   string  `json:"notify_url" gorm:"size:255"`
	Return_url   string  `json:"return_url" gorm:"size:255"`
	Pay_id       string  `json:"pay_id" gorm:"size:255"`
	Pay_url      string  `json:"pay_url" gorm:"size:255"`
	Param        string  `json:"param" gorm:"size:255"`
	Is_auto      int     `json:"is_auto" gorm:"not null"`
	Create_date  int64   `json:"create_date" gorm:"not null"`
	Pay_date     int64   `json:"pay_date" gorm:"not null"`
	Close_date   int64   `json:"close_date" gorm:"not null"`

	// 关联关系
	User *User `json:"user,omitempty" gorm:"foreignKey:User_id"`
}

// 订单状态常量
const (
	OrderStatusClosed       = -1 // 已关闭/过期
	OrderStatusPending      = 0  // 待支付
	OrderStatusPaid         = 1  // 已支付
	OrderStatusNotifyFailed = 2  // 通知失败（PHP版本兼容）
)

// 订单类型常量
const (
	OrderTypeWechat = 1 // 微信支付
	OrderTypeAlipay = 2 // 支付宝
)

// TableName 指定表名
func (Order) TableName() string {
	return "pay_order"
}

// BeforeCreate 创建前钩子
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	o.Create_date = time.Now().Unix()
	return nil
}

// IsPaid 检查订单是否已支付
func (o *Order) IsPaid() bool {
	return o.State == OrderStatusPaid
}

// IsClosed 检查订单是否已关闭
func (o *Order) IsClosed() bool {
	return o.State == OrderStatusClosed
}

// IsExpired 检查订单是否过期（基于创建时间超过30分钟）
func (o *Order) IsExpired() bool {
	if o.State != OrderStatusPending {
		return false
	}
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute).Unix()
	return o.Create_date < thirtyMinutesAgo
}

// IsExpiredWithMinutes 检查订单是否过期（基于指定的分钟数）
func (o *Order) IsExpiredWithMinutes(expireMinutes int) bool {
	if o.State != OrderStatusPending {
		return false
	}
	expireTime := time.Now().Add(-time.Duration(expireMinutes) * time.Minute).Unix()
	return o.Create_date < expireTime
}

// GetStatusText 获取状态文本
func (o *Order) GetStatusText() string {
	switch o.State {
	case OrderStatusClosed:
		return "已关闭"
	case OrderStatusPending:
		return "待支付"
	case OrderStatusPaid:
		return "已支付"
	case OrderStatusNotifyFailed:
		return "通知失败"
	default:
		return "未知状态"
	}
}

// GetTypeText 获取类型文本
func (o *Order) GetTypeText() string {
	switch o.Type {
	case OrderTypeAlipay:
		return "支付宝"
	case OrderTypeWechat:
		return "微信支付"
	default:
		return "未知类型"
	}
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	Type       int     `json:"type" binding:"required,oneof=1 2"`
	Price      float64 `json:"price" binding:"required,gt=0"`
	Subject    string  `json:"subject" binding:"required,max=200"`
	Body       string  `json:"body" binding:"omitempty,max=500"`
	Notify_url string  `json:"notify_url" binding:"omitempty,url,max=255"`
	Return_url string  `json:"return_url" binding:"omitempty,url,max=255"`
}

// UpdateOrderRequest 更新订单请求
type UpdateOrderRequest struct {
	Subject    string `json:"subject" binding:"omitempty,max=200"`
	Body       string `json:"body" binding:"omitempty,max=500"`
	Notify_url string `json:"notify_url" binding:"omitempty,url,max=255"`
	Return_url string `json:"return_url" binding:"omitempty,url,max=255"`
}

// OrderListRequest 订单列表请求
type OrderListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	State    *int   `form:"state" binding:"omitempty,min=0,max=2"`
	Type     *int   `form:"type" binding:"omitempty,oneof=1 2"`
	Order_id string `form:"order_id" binding:"omitempty,max=255"`
	User_id  *uint  `form:"user_id" binding:"omitempty,min=1"`
	Start_at string `form:"start_at" binding:"omitempty"`
	End_at   string `form:"end_at" binding:"omitempty"`
}

// OrderStatusResponse 订单状态响应
type OrderStatusResponse struct {
	Order_id   string `json:"order_id"`
	State      int    `json:"state"`
	State_text string `json:"state_text"`
	Is_paid    bool   `json:"is_paid"`
	Is_expired bool   `json:"is_expired"`
	Pay_date   int64  `json:"pay_date"`
}

// CloseExpiredOrdersRequest 关闭过期订单请求
type CloseExpiredOrdersRequest struct {
	User_id *uint `json:"user_id" binding:"omitempty,min=1"`
	Limit   int   `json:"limit" binding:"omitempty,min=1,max=1000"`
}

// DeleteExpiredOrdersRequest 删除过期订单请求
type DeleteExpiredOrdersRequest struct {
	User_id *uint `json:"user_id" binding:"omitempty,min=1"`
	Limit   int   `json:"limit" binding:"omitempty,min=1,max=1000"`
	// 只删除已关闭的订单，默认为true
	OnlyClosed bool `json:"only_closed" binding:"omitempty"`
	// 过期时间（天），默认为30天
	ExpireDays int `json:"expire_days" binding:"omitempty,min=1"`
}

// OrderResponse 订单响应
type OrderResponse struct {
	Id           uint    `json:"id"`
	User_id      uint    `json:"user_id"`
	Order_id     string  `json:"order_id"`
	Type         int     `json:"type"`
	Type_text    string  `json:"type_text"`
	Price        float64 `json:"price"`
	Really_price float64 `json:"really_price"`
	State        int     `json:"state"`
	State_text   string  `json:"state_text"`
	Notify_url   string  `json:"notify_url"`
	Return_url   string  `json:"return_url"`
	Pay_id       string  `json:"pay_id"`
	Pay_url      string  `json:"pay_url"`
	Param        string  `json:"param"`
	Is_auto      int     `json:"is_auto"`
	Create_date  int64   `json:"create_date"`
	Pay_date     int64   `json:"pay_date"`
	Close_date   int64   `json:"close_date"`
	User         *User   `json:"user,omitempty"`
}

// ToResponse 转换为响应格式
func (o *Order) ToResponse() *OrderResponse {
	resp := &OrderResponse{
		Id:           o.Id,
		User_id:      o.User_id,
		Order_id:     o.Order_id,
		Type:         o.Type,
		Type_text:    o.GetTypeText(),
		Price:        o.Price,
		Really_price: o.Really_price,
		State:        o.State,
		State_text:   o.GetStatusText(),
		Notify_url:   o.Notify_url,
		Return_url:   o.Return_url,
		Pay_id:       o.Pay_id,
		Pay_url:      o.Pay_url,
		Param:        o.Param,
		Is_auto:      o.Is_auto,
		Create_date:  o.Create_date,
		Pay_date:     o.Pay_date,
		Close_date:   o.Close_date,
	}

	if o.User != nil {
		resp.User = o.User
	}

	return resp
}

// OrderStats 订单统计结构
type OrderStats struct {
	TotalOrders   int64   `json:"total_orders"`
	SuccessOrders int64   `json:"success_orders"`
	ClosedOrders  int64   `json:"closed_orders"`
	TotalAmount   float64 `json:"total_amount"`
}
