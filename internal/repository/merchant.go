package repository

import (
	"vmqfox-api-go/internal/model"

	"gorm.io/gorm"
)

// MerchantRepository 商户映射仓库接口
type MerchantRepository interface {
	GetUserIDByAppID(appId string) (uint, error)
	UpsertMapping(appId string, userId uint) error
	GetMappingByUserID(userId uint) (*model.MerchantMapping, error)
	GetMappingByAppID(appId string) (*model.MerchantMapping, error)
	CheckAppIdExists(appId string, excludeUserId uint) (bool, error)
	DeleteMapping(appId string) error
	GetAllMappings() ([]*model.MerchantMapping, error)
}

// merchantRepository 商户映射仓库实现
type merchantRepository struct {
	db *gorm.DB
}

// NewMerchantRepository 创建商户映射仓库
func NewMerchantRepository(db *gorm.DB) MerchantRepository {
	return &merchantRepository{
		db: db,
	}
}

// GetUserIDByAppID 通过AppID获取用户ID
func (r *merchantRepository) GetUserIDByAppID(appId string) (uint, error) {
	var mapping model.MerchantMapping
	err := r.db.Where("app_id = ? AND status = 1", appId).First(&mapping).Error
	if err != nil {
		return 0, err
	}
	return mapping.UserId, nil
}

// UpsertMapping 创建或更新商户映射
func (r *merchantRepository) UpsertMapping(appId string, userId uint) error {
	// 先检查是否存在
	var existing model.MerchantMapping
	err := r.db.Where("app_id = ?", appId).First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 不存在，创建新记录
			mapping := &model.MerchantMapping{
				AppId:  appId,
				UserId: userId,
				Status: 1,
			}
			return r.db.Create(mapping).Error
		}
		return err
	}

	// 存在，只更新必要字段
	return r.db.Model(&existing).Updates(map[string]interface{}{
		"user_id": userId,
		"status":  1,
	}).Error
}

// GetMappingByUserID 根据用户ID获取商户映射
func (r *merchantRepository) GetMappingByUserID(userId uint) (*model.MerchantMapping, error) {
	var mapping model.MerchantMapping
	err := r.db.Where("user_id = ?", userId).First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetMappingByAppID 根据AppID获取商户映射
func (r *merchantRepository) GetMappingByAppID(appId string) (*model.MerchantMapping, error) {
	var mapping model.MerchantMapping
	err := r.db.Where("app_id = ?", appId).First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

// CheckAppIdExists 检查AppID是否已存在（排除指定用户）
func (r *merchantRepository) CheckAppIdExists(appId string, excludeUserId uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.MerchantMapping{}).Where("app_id = ?", appId)
	if excludeUserId > 0 {
		query = query.Where("user_id != ?", excludeUserId)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// DeleteMapping 删除商户映射
func (r *merchantRepository) DeleteMapping(appId string) error {
	return r.db.Where("app_id = ?", appId).Delete(&model.MerchantMapping{}).Error
}

// GetAllMappings 获取所有商户映射
func (r *merchantRepository) GetAllMappings() ([]*model.MerchantMapping, error) {
	var mappings []*model.MerchantMapping
	err := r.db.Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, nil
}
