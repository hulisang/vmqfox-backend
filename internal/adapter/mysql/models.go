package mysql

import (
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/admin"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
)

// OrderRow 是 Go 原生订单表映射，金额始终以整数分为业务主表示。
type OrderRow struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID         string     `gorm:"column:order_id;not null;uniqueIndex:uq_orders_order_id"`
	PublicToken     string     `gorm:"column:public_token;type:char(64);not null;uniqueIndex:uq_orders_public_token"`
	PayID           string     `gorm:"column:pay_id;not null;uniqueIndex:uq_orders_pay_id"`
	PaymentType     int        `gorm:"column:payment_type;not null"`
	PriceCent       int64      `gorm:"column:price_cent;not null"`
	ReallyPriceCent int64      `gorm:"column:really_price_cent;not null"`
	PriceText       string     `gorm:"column:price_text;not null"`
	ReallyPriceText string     `gorm:"column:really_price_text;not null"`
	State           int        `gorm:"column:state;not null"`
	Param           string     `gorm:"column:param;not null"`
	PayURL          string     `gorm:"column:pay_url;not null;type:text"`
	IsAuto          bool       `gorm:"column:is_auto;not null"`
	NotifyURL       string     `gorm:"column:notify_url;not null;type:text"`
	ReturnURL       string     `gorm:"column:return_url;not null;type:text"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null"`
	PaidAt          *time.Time `gorm:"column:paid_at"`
	ClosedAt        *time.Time `gorm:"column:closed_at"`
}

func (OrderRow) TableName() string { return "orders" }

func (m OrderRow) ToDomain() order.Order {
	return order.Order{
		ID:               m.ID,
		OrderID:          m.OrderID,
		PublicToken:      m.PublicToken,
		PayID:            m.PayID,
		Type:             payment.Type(m.PaymentType),
		PriceCents:       m.PriceCent,
		ReallyPriceCents: m.ReallyPriceCent,
		PriceText:        m.PriceText,
		ReallyPriceText:  m.ReallyPriceText,
		State:            order.Status(m.State),
		Param:            m.Param,
		PayURL:           m.PayURL,
		IsAuto:           m.IsAuto,
		NotifyURL:        m.NotifyURL,
		ReturnURL:        m.ReturnURL,
		CreatedAt:        m.CreatedAt,
		PaidAt:           timeValue(m.PaidAt),
		ClosedAt:         timeValue(m.ClosedAt),
	}
}

// QRCodeRow 是 Go 原生二维码表映射。
type QRCodeRow struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	PayURL      string    `gorm:"column:pay_url;not null;type:text"`
	PriceCent   int64     `gorm:"column:price_cent;not null"`
	PaymentType int       `gorm:"column:payment_type;not null"`
	State       int       `gorm:"column:state;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (QRCodeRow) TableName() string { return "qrcodes" }

func (m QRCodeRow) ToDomain() qrcode.QRCode {
	return qrcode.QRCode{
		ID:         m.ID,
		PayURL:     m.PayURL,
		PriceCents: m.PriceCent,
		Type:       payment.Type(m.PaymentType),
		State:      qrcode.State(m.State),
	}
}

// SettingRow 将设置键和值与旧 PHP 的 vkey/vvalue 语义隔离。
type SettingRow struct {
	Key   string `gorm:"column:setting_key;primaryKey"`
	Value string `gorm:"column:setting_value;not null;type:text"`
}

func (SettingRow) TableName() string { return "settings" }

// AdminCredentialRow 是单用户管理员凭据映射，只保存不可逆密码哈希。
type AdminCredentialRow struct {
	ID             uint8     `gorm:"column:id;primaryKey"`
	Username       string    `gorm:"column:username;not null"`
	PasswordHash   string    `gorm:"column:password_hash;not null;type:varbinary(255)"`
	Enabled        bool      `gorm:"column:enabled;not null"`
	SingletonGuard uint8     `gorm:"column:singleton_guard;->"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (AdminCredentialRow) TableName() string { return "admin_credentials" }

func (m AdminCredentialRow) ToDomain() admin.Credential {
	return admin.Credential{
		ID:           m.ID,
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
		Enabled:      m.Enabled,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// PriceLockRow 用复合主键表达“同一支付类型的同一金额只能被一个订单占用”。
type PriceLockRow struct {
	PaymentType int       `gorm:"column:payment_type;primaryKey"`
	AmountCent  int64     `gorm:"column:amount_cent;primaryKey"`
	OrderID     string    `gorm:"column:order_id;not null;uniqueIndex:uq_price_locks_order_id"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (PriceLockRow) TableName() string { return "price_locks" }

// PaymentEventRow 是到账事件幂等表，不承载订单状态。
type PaymentEventRow struct {
	EventKey    string    `gorm:"column:event_key;primaryKey"`
	PaymentType int       `gorm:"column:payment_type;not null"`
	PriceCent   int64     `gorm:"column:price_cent;not null"`
	EventTime   string    `gorm:"column:event_time;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (PaymentEventRow) TableName() string { return "payment_events" }

// NotificationOutboxRow 保存预签名通知载荷，worker 不需要读取商户密钥。
type NotificationOutboxRow struct {
	ID            string     `gorm:"column:id;primaryKey"`
	Topic         string     `gorm:"column:topic;not null;uniqueIndex:uq_outbox_event_topic"`
	AggregateID   string     `gorm:"column:aggregate_id;not null"`
	EventKey      string     `gorm:"column:event_key;not null;uniqueIndex:uq_outbox_event_topic"`
	Payload       string     `gorm:"column:payload;not null;type:longtext"`
	Status        int        `gorm:"column:status;not null"`
	Attempts      int        `gorm:"column:attempts;not null"`
	NextAttemptAt time.Time  `gorm:"column:next_attempt_at;not null"`
	LeaseToken    string     `gorm:"column:lease_token;not null"`
	LockedUntil   *time.Time `gorm:"column:locked_until"`
	LastError     *string    `gorm:"column:last_error;type:text"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
	DeliveredAt   *time.Time `gorm:"column:delivered_at"`
}

func (NotificationOutboxRow) TableName() string { return "notification_outbox" }

func timeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
