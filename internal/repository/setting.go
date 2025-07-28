package repository

import (
	"vmqfox-api-go/internal/model"

	"gorm.io/gorm"
)

// SettingRepository 系统设置仓库接口
type SettingRepository interface {
	GetSetting(userID uint, key string) (*model.Setting, error)
	GetAllSettings(userID uint) ([]*model.Setting, error)
	UpdateSetting(userID uint, key, value string) error
	CreateSetting(setting *model.Setting) error
	GetSettingsMap(userID uint) (map[string]string, error)
	BatchUpdateSettings(userID uint, settings map[string]string) error
	DeleteSetting(userID uint, key string) error
	DeleteAllUserSettings(userID uint) error
	GetAllSettingsByKey(key string) ([]*model.Setting, error)
}

// settingRepository 系统设置仓库实现
type settingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 创建系统设置仓库
func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{
		db: db,
	}
}

// GetSetting 获取单个设置
func (r *settingRepository) GetSetting(userID uint, key string) (*model.Setting, error) {
	var setting model.Setting
	err := r.db.Where("vkey = ? AND user_id = ?", key, userID).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetAllSettings 获取所有设置
func (r *settingRepository) GetAllSettings(userID uint) ([]*model.Setting, error) {
	var settings []*model.Setting
	err := r.db.Where("user_id = ?", userID).Find(&settings).Error
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// UpdateSetting 更新设置
func (r *settingRepository) UpdateSetting(userID uint, key, value string) error {
	return r.db.Model(&model.Setting{}).
		Where("vkey = ? AND user_id = ?", key, userID).
		Update("vvalue", value).Error
}

// CreateSetting 创建设置
func (r *settingRepository) CreateSetting(setting *model.Setting) error {
	return r.db.Create(setting).Error
}

// GetSettingsMap 获取设置键值对映射
func (r *settingRepository) GetSettingsMap(userID uint) (map[string]string, error) {
	var settings []*model.Setting
	err := r.db.Where("user_id = ?", userID).Find(&settings).Error
	if err != nil {
		return nil, err
	}

	settingsMap := make(map[string]string)
	for _, setting := range settings {
		settingsMap[setting.Vkey] = setting.Vvalue
	}
	return settingsMap, nil
}

// BatchUpdateSettings 批量更新设置
func (r *settingRepository) BatchUpdateSettings(userID uint, settings map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range settings {
			err := tx.Model(&model.Setting{}).
				Where("vkey = ? AND user_id = ?", key, userID).
				Update("vvalue", value).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteSetting 删除单个设置
func (r *settingRepository) DeleteSetting(userID uint, key string) error {
	return r.db.Where("vkey = ? AND user_id = ?", key, userID).Delete(&model.Setting{}).Error
}

// DeleteAllUserSettings 删除用户的所有设置
func (r *settingRepository) DeleteAllUserSettings(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.Setting{}).Error
}

// GetAllSettingsByKey 根据key获取所有用户的设置
func (r *settingRepository) GetAllSettingsByKey(key string) ([]*model.Setting, error) {
	var settings []*model.Setting
	err := r.db.Where("vkey = ?", key).Find(&settings).Error
	if err != nil {
		return nil, err
	}
	return settings, nil
}
