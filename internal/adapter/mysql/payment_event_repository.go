package mysql

import (
	"context"

	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
)

type PaymentEventRepository struct {
	db *gorm.DB
}

func NewPaymentEventRepository(db *gorm.DB) *PaymentEventRepository {
	return &PaymentEventRepository{db: db}
}

func (r *PaymentEventRepository) RecordIfNew(ctx context.Context, event port.PaymentEvent) (bool, error) {
	row := PaymentEventRow{
		EventKey:    event.EventKey,
		PaymentType: int(event.PaymentType),
		PriceCent:   event.PriceCents,
		EventTime:   event.EventTime,
		CreatedAt:   event.CreatedAt,
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

var _ port.PaymentEventRepository = (*PaymentEventRepository)(nil)
