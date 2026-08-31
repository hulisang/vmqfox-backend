package mysql

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	outboxPending   = 0
	outboxDelivered = 1
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Enqueue(ctx context.Context, message port.OutboxMessage) error {
	createdAt := message.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	nextAttemptAt := message.NextAttemptAt
	if nextAttemptAt.IsZero() {
		nextAttemptAt = createdAt
	}
	row := NotificationOutboxRow{
		ID:            message.ID,
		Topic:         message.Topic,
		AggregateID:   message.AggregateID,
		EventKey:      message.EventKey,
		Payload:       string(message.Payload),
		Status:        outboxPending,
		Attempts:      message.Attempts,
		NextAttemptAt: nextAttemptAt,
		LeaseToken:    "",
		CreatedAt:     createdAt,
	}
	result := databaseFromContext(ctx, r.db).Create(&row)
	if duplicateKey(result.Error) {
		return port.ErrAlreadyProcessed
	}
	return result.Error
}

func (r *OutboxRepository) ClaimPending(ctx context.Context, now time.Time, limit int, leaseDuration time.Duration) ([]port.OutboxMessage, error) {
	if leaseDuration <= 0 {
		return nil, errors.New("通知任务租约时长无效")
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	leaseToken, err := newLeaseToken()
	if err != nil {
		return nil, err
	}
	var rows []NotificationOutboxRow
	db := databaseFromContext(ctx, r.db)
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND next_attempt_at <= ? AND (locked_until IS NULL OR locked_until <= ?)", outboxPending, now, now).
			Order("created_at ASC, id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		result := tx.Model(&NotificationOutboxRow{}).
			Where("id IN ? AND status = ? AND (locked_until IS NULL OR locked_until <= ?)", ids, outboxPending, now).
			Updates(map[string]any{
				"lease_token":  leaseToken,
				"locked_until": now.Add(leaseDuration),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(rows)) {
			return port.ErrConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	messages := make([]port.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, port.OutboxMessage{
			ID:            row.ID,
			Topic:         row.Topic,
			AggregateID:   row.AggregateID,
			Payload:       []byte(row.Payload),
			EventKey:      row.EventKey,
			CreatedAt:     row.CreatedAt,
			Attempts:      row.Attempts,
			NextAttemptAt: row.NextAttemptAt,
			LeaseToken:    leaseToken,
		})
	}
	return messages, nil
}

func (r *OutboxRepository) MarkDelivered(ctx context.Context, id, leaseToken string, deliveredAt time.Time) error {
	if leaseToken == "" {
		return port.ErrConflict
	}
	result := databaseFromContext(ctx, r.db).
		Model(&NotificationOutboxRow{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, outboxPending, leaseToken).
		Updates(map[string]any{
			"status":       outboxDelivered,
			"delivered_at": deliveredAt,
			"lease_token":  "",
			"locked_until": nil,
			"last_error":   nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return port.ErrConflict
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id, leaseToken string, attempts int, nextAttemptAt time.Time, reason string) error {
	if leaseToken == "" || attempts < 1 {
		return port.ErrConflict
	}
	reason = strings.ToValidUTF8(reason, "�")
	if len(reason) > 1000 {
		reason = reason[:1000]
		for !utf8.ValidString(reason) {
			reason = reason[:len(reason)-1]
		}
	}
	result := databaseFromContext(ctx, r.db).
		Model(&NotificationOutboxRow{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, outboxPending, leaseToken).
		Updates(map[string]any{
			"attempts":        attempts,
			"next_attempt_at": nextAttemptAt,
			"lease_token":     "",
			"locked_until":    nil,
			"last_error":      reason,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return port.ErrConflict
	}
	return nil
}

func newLeaseToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", errors.New("生成通知任务租约失败")
	}
	return hex.EncodeToString(raw[:]), nil
}

var _ port.OutboxRepository = (*OutboxRepository)(nil)
