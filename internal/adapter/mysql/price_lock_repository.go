package mysql

import (
	"context"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
)

type PriceLockRepository struct {
	db *gorm.DB
}

func NewPriceLockRepository(db *gorm.DB) *PriceLockRepository {
	return &PriceLockRepository{db: db}
}

func (r *PriceLockRepository) TryAcquire(ctx context.Context, orderID string, paymentType payment.Type, amountCents int64) (bool, error) {
	row := PriceLockRow{
		PaymentType: int(paymentType),
		AmountCent:  amountCents,
		OrderID:     orderID,
		CreatedAt:   time.Now().UTC(),
	}
	result := databaseFromContext(ctx, r.db).Create(&row)
	if result.Error == nil {
		return true, nil
	}
	if duplicateKey(result.Error) {
		return false, nil
	}
	return false, result.Error
}

func (r *PriceLockRepository) ReleaseByOrderID(ctx context.Context, orderID string) error {
	return databaseFromContext(ctx, r.db).Where("order_id = ?", orderID).Delete(&PriceLockRow{}).Error
}

func (r *PriceLockRepository) ReleaseBefore(ctx context.Context, before time.Time) error {
	return databaseFromContext(ctx, r.db).Where("created_at < ?", before).Delete(&PriceLockRow{}).Error
}

var _ port.PriceLockRepository = (*PriceLockRepository)(nil)
