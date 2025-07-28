package model

// PayQrcode 收款码模型
type PayQrcode struct {
	Id      uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	User_id uint    `json:"user_id" gorm:"not null;index"`
	Pay_url string  `json:"pay_url" gorm:"size:500;not null"`
	Price   float64 `json:"price" gorm:"type:decimal(10,2);not null"`
	Type    int     `json:"type" gorm:"not null;comment:1=微信,2=支付宝"`
	State   int     `json:"state" gorm:"default:1;comment:0=禁用,1=启用"`
}

// TableName 指定表名
func (PayQrcode) TableName() string {
	return "pay_qrcode"
}

// ==================== 收款码相关请求和响应结构 ====================

// QrcodeListRequest 收款码列表请求
type QrcodeListRequest struct {
	Page    int      `form:"page" binding:"min=1"`
	Limit   int      `form:"limit" binding:"min=1,max=100"`
	User_id *uint    `form:"user_id"`
	Type    *int     `form:"type"`
	State   *int     `form:"state"`
	Price   *float64 `form:"price"`
}

// CreateQrcodeRequest 创建收款码请求
type CreateQrcodeRequest struct {
	Pay_url string  `json:"pay_url" binding:"required,url"`
	Price   float64 `json:"price" binding:"required,min=0"`
	Type    int     `json:"type" binding:"required,oneof=1 2"`
}

// UpdateQrcodeStatusRequest 更新收款码状态请求
type UpdateQrcodeStatusRequest struct {
	State *int `json:"state" binding:"required,oneof=0 1"`
}

// ParseQrcodeRequest 解析收款码请求
type ParseQrcodeRequest struct {
	Qrcode_url string `json:"qrcode_url" binding:"required,url"`
}

// ParseQrcodeResponse 解析收款码响应
type ParseQrcodeResponse struct {
	Type   int     `json:"type"`
	Amount float64 `json:"amount"`
	Url    string  `json:"url"`
}

// GenerateQrcodeRequest 生成二维码请求
type GenerateQrcodeRequest struct {
	Text string `form:"text"`
	Url  string `form:"url"`
	Size int    `form:"size" binding:"omitempty,min=100,max=1000"`
}

// GenerateQrcodeResponse 生成二维码响应
type GenerateQrcodeResponse struct {
	Qrcode_url string `json:"qrcode_url"`
	Size       int    `json:"size"`
	Format     string `json:"format"`
}

// QrcodeResponse 收款码响应结构
type QrcodeResponse struct {
	Id      uint    `json:"id"`
	User_id uint    `json:"user_id"`
	Pay_url string  `json:"pay_url"`
	Price   float64 `json:"price"`
	Type    int     `json:"type"`
	State   int     `json:"state"`
}

// ToResponse 转换为响应格式
func (q *PayQrcode) ToResponse() *QrcodeResponse {
	return &QrcodeResponse{
		Id:      q.Id,
		User_id: q.User_id,
		Pay_url: q.Pay_url,
		Price:   q.Price,
		Type:    q.Type,
		State:   q.State,
	}
}
