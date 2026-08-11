package mysql

import (
	"context"
	"sort"

	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) Get(ctx context.Context, key string) (string, error) {
	var row SettingRow
	err := databaseFromContext(ctx, r.db).Where("setting_key = ?", key).Take(&row).Error
	if err = mapDatabaseError(err); err != nil {
		return "", err
	}
	return row.Value, nil
}

func (r *SettingRepository) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return result, nil
	}

	var rows []SettingRow
	if err := databaseFromContext(ctx, r.db).Where("setting_key IN ?", keys).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

func (r *SettingRepository) GetManyForUpdate(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return result, nil
	}

	uniqueKeys := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		uniqueKeys = append(uniqueKeys, key)
	}
	sort.Strings(uniqueKeys)

	var rows []SettingRow
	if err := databaseFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("setting_key IN ?", uniqueKeys).
		Order("setting_key ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(uniqueKeys) {
		return nil, port.ErrNotFound
	}
	for _, row := range rows {
		result[row.Key] = row.Value
	}
	return result, nil
}

func (r *SettingRepository) Set(ctx context.Context, key, value string) error {
	row := SettingRow{Key: key, Value: value}
	return databaseFromContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"setting_value"}),
		}).
		Create(&row).Error
}

func (r *SettingRepository) SetMany(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]SettingRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, SettingRow{Key: key, Value: values[key]})
	}
	return databaseFromContext(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"setting_value"}),
		}).
		Create(&rows).Error
}

var _ port.SettingRepository = (*SettingRepository)(nil)
