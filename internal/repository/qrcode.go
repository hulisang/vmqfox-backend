package repository

import (
	"gorm.io/gorm"

	"vmqfox-api-go/internal/model"
)

type QrcodeRepository interface {
	GetQrcodes(req *model.QrcodeListRequest) ([]*model.PayQrcode, int64, error)
	Create(qrcode *model.PayQrcode) error
	Delete(qrcodeID uint) error
	GetByID(qrcodeID uint) (*model.PayQrcode, error)
	GetQrcodeByPriceAndType(price float64, qrcodeType int) (*model.PayQrcode, error) // 新增方法
	UpdateStatus(qrcodeID uint, state int) error
	ExistsByUserAndPrice(userID uint, price float64, qrcodeType int) (bool, error)
}

type qrcodeRepository struct {
	db *gorm.DB
}

func NewQrcodeRepository(db *gorm.DB) QrcodeRepository {
	return &qrcodeRepository{
		db: db,
	}
}

// GetQrcodes 获取收款码列表
func (r *qrcodeRepository) GetQrcodes(req *model.QrcodeListRequest) ([]*model.PayQrcode, int64, error) {
	var qrcodes []*model.PayQrcode
	var total int64

	// 构建查询
	query := r.db.Model(&model.PayQrcode{})

	// 用户过滤（数据隔离）
	if req.User_id != nil {
		query = query.Where("user_id = ?", *req.User_id)
	}

	// 类型过滤
	if req.Type != nil {
		query = query.Where("type = ?", *req.Type)
	}

	// 状态过滤
	if req.State != nil {
		query = query.Where("state = ?", *req.State)
	}

	// 价格过滤
	if req.Price != nil {
		query = query.Where("price = ?", *req.Price)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Limit
	if err := query.Offset(offset).Limit(req.Limit).
		Order("id DESC").
		Find(&qrcodes).Error; err != nil {
		return nil, 0, err
	}

	return qrcodes, total, nil
}

// Create 创建收款码
func (r *qrcodeRepository) Create(qrcode *model.PayQrcode) error {
	return r.db.Create(qrcode).Error
}

// Delete 删除收款码
func (r *qrcodeRepository) Delete(qrcodeID uint) error {
	return r.db.Delete(&model.PayQrcode{}, qrcodeID).Error
}

// GetByID 根据ID获取收款码
func (r *qrcodeRepository) GetByID(qrcodeID uint) (*model.PayQrcode, error) {
	var qrcode model.PayQrcode
	err := r.db.First(&qrcode, qrcodeID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &qrcode, nil
}

// UpdateStatus 更新收款码状态
func (r *qrcodeRepository) UpdateStatus(qrcodeID uint, state int) error {
	return r.db.Model(&model.PayQrcode{}).
		Where("id = ?", qrcodeID).
		Update("state", state).Error
}

// ExistsByUserAndPrice 检查用户是否已有相同价格和类型的收款码
func (r *qrcodeRepository) ExistsByUserAndPrice(userID uint, price float64, qrcodeType int) (bool, error) {
	var count int64
	err := r.db.Model(&model.PayQrcode{}).
		Where("user_id = ? AND price = ? AND type = ?", userID, price, qrcodeType).
		Count(&count).Error
	return count > 0, err
}

// GetQrcodeByPriceAndType 根据价格和类型获取收款码
func (r *qrcodeRepository) GetQrcodeByPriceAndType(price float64, qrcodeType int) (*model.PayQrcode, error) {
	var qrcode model.PayQrcode
	err := r.db.Where("price = ? AND type = ? AND state = ?", price, qrcodeType, 0).First(&qrcode).Error
	if err != nil {
		return nil, err
	}
	return &qrcode, nil
}
