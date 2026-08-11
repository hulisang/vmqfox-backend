package mysql

import (
	"context"

	"github.com/hulisang/vmqfox-backend/internal/domain/admin"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminCredentialRepository struct {
	db *gorm.DB
}

func NewAdminCredentialRepository(db *gorm.DB) *AdminCredentialRepository {
	return &AdminCredentialRepository{db: db}
}

func (r *AdminCredentialRepository) Get(ctx context.Context) (admin.Credential, error) {
	return r.get(ctx, false)
}

func (r *AdminCredentialRepository) GetForUpdate(ctx context.Context) (admin.Credential, error) {
	return r.get(ctx, true)
}

func (r *AdminCredentialRepository) get(ctx context.Context, forUpdate bool) (admin.Credential, error) {
	query := databaseFromContext(ctx, r.db).Where("id = ?", admin.SingletonID)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row AdminCredentialRow
	err := query.Take(&row).Error
	if err = mapDatabaseError(err); err != nil {
		return admin.Credential{}, err
	}
	value := row.ToDomain()
	if err := value.Validate(); err != nil {
		return admin.Credential{}, err
	}
	return value, nil
}

func (r *AdminCredentialRepository) Update(ctx context.Context, value admin.Credential) error {
	value.ID = admin.SingletonID
	value.Username = admin.NormalizeUsername(value.Username)
	if err := value.Validate(); err != nil {
		return err
	}
	result := databaseFromContext(ctx, r.db).
		Model(&AdminCredentialRow{}).
		Where("id = ?", admin.SingletonID).
		Updates(map[string]any{
			"username":      value.Username,
			"password_hash": value.PasswordHash,
			"enabled":       value.Enabled,
			"updated_at":    value.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return port.ErrNotFound
	}
	return nil
}

var _ port.AdminCredentialRepository = (*AdminCredentialRepository)(nil)
