package mysql

import (
	"context"
	"errors"

	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
)

type transactionContextKey struct{}

// TransactionManager 将事务对象放入 context；仓库只从 context 读取，不暴露 GORM 给用例层。
type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if m == nil || m.db == nil {
		return errors.New("数据库事务管理器未初始化")
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, transactionContextKey{}, tx))
	})
}

func databaseFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}

func mapDatabaseError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return port.ErrNotFound
	}
	return err
}
