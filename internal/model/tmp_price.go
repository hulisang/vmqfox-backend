package model

// TmpPrice 临时价格表模型 - 用于避免金额冲突
type TmpPrice struct {
	Price string `json:"price" gorm:"primaryKey;size:255;not null;comment:价格-类型组合"`
	Oid   string `json:"oid" gorm:"size:255;not null;comment:订单ID"`
}

// TableName 指定表名
func (TmpPrice) TableName() string {
	return "tmp_price"
}
