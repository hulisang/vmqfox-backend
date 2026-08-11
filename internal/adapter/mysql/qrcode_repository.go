package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
)

type QRCodeRepository struct {
	db *gorm.DB
}

func NewQRCodeRepository(db *gorm.DB) *QRCodeRepository {
	return &QRCodeRepository{db: db}
}

func (r *QRCodeRepository) FindEnabledByAmount(ctx context.Context, typ payment.Type, amountCents int64) (qrcode.QRCode, error) {
	var row QRCodeRow
	err := databaseFromContext(ctx, r.db).
		Where("payment_type = ? AND state = ? AND price_cent = ?", int(typ), int(qrcode.StateEnabled), amountCents).
		Order("id ASC").
		Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *QRCodeRepository) FindByID(ctx context.Context, id int64) (qrcode.QRCode, error) {
	var row QRCodeRow
	err := databaseFromContext(ctx, r.db).Where("id = ?", id).Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *QRCodeRepository) List(ctx context.Context, filter port.QRCodeFilter) (port.QRCodePage, error) {
	page, limit := normalizePage(filter.Page, filter.Limit)
	query := databaseFromContext(ctx, r.db).Model(&QRCodeRow{})
	if filter.Type != nil {
		query = query.Where("payment_type = ?", int(*filter.Type))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return port.QRCodePage{}, err
	}
	var rows []QRCodeRow
	if err := query.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return port.QRCodePage{}, err
	}

	items := make([]qrcode.QRCode, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.ToDomain())
	}
	return port.QRCodePage{Items: items, Total: total}, nil
}

func (r *QRCodeRepository) Create(ctx context.Context, value qrcode.QRCode) (qrcode.QRCode, error) {
	now := time.Now().UTC()
	row := QRCodeRow{
		ID:          value.ID,
		PayURL:      value.PayURL,
		PriceCent:   value.PriceCents,
		PaymentType: int(value.Type),
		State:       int(value.State),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	result := databaseFromContext(ctx, r.db).Create(&row)
	if duplicateKey(result.Error) {
		return qrcode.QRCode{}, port.ErrConflict
	}
	if result.Error != nil {
		return qrcode.QRCode{}, result.Error
	}
	return row.ToDomain(), nil
}

func (r *QRCodeRepository) SetState(ctx context.Context, id int64, from, to qrcode.State) (bool, error) {
	if !from.Valid() || !to.Valid() {
		return false, errors.New("二维码状态无效")
	}
	result := databaseFromContext(ctx, r.db).
		Model(&QRCodeRow{}).
		Where("id = ? AND state = ?", id, int(from)).
		Updates(map[string]any{
			"state":      int(to),
			"updated_at": time.Now().UTC(),
		})
	return result.RowsAffected == 1, result.Error
}

func (r *QRCodeRepository) Delete(ctx context.Context, id int64) (bool, error) {
	result := databaseFromContext(ctx, r.db).Where("id = ?", id).Delete(&QRCodeRow{})
	return result.RowsAffected == 1, result.Error
}

var _ port.QRCodeRepository = (*QRCodeRepository)(nil)
