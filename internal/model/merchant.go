package model

import "time"

// MerchantMapping 商户ID映射模型
type MerchantMapping struct {
	AppId     string    `json:"app_id" gorm:"primaryKey;size:255;comment:商户ID"`
	UserId    uint      `json:"user_id" gorm:"not null;uniqueIndex;comment:用户ID"`
	Status    int       `json:"status" gorm:"type:tinyint(1);not null;default:1;comment:状态：1=启用，0=禁用"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`

	// 关联关系
	User *User `json:"user,omitempty" gorm:"foreignKey:UserId"`
}

// TableName 指定表名
func (MerchantMapping) TableName() string {
	return "merchant_mapping"
}

// IsEnabled 检查商户映射是否启用
func (m *MerchantMapping) IsEnabled() bool {
	return m.Status == 1
}

// MerchantMappingRequest 商户映射请求
type MerchantMappingRequest struct {
	AppId string `json:"app_id" binding:"required,min=6,max=32"`
}

// MerchantMappingResponse 商户映射响应
type MerchantMappingResponse struct {
	AppId     string    `json:"app_id"`
	UserId    uint      `json:"user_id"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
