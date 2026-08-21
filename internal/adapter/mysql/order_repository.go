package mysql

import (
	"context"
	"errors"
	"strings"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/port"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id int64) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).Where("id = ?", id).Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) FindByIDForUpdate(ctx context.Context, id int64) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) FindByOrderID(ctx context.Context, orderID string) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).Where("order_id = ?", orderID).Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) FindByOrderIDForUpdate(ctx context.Context, orderID string) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ?", orderID).
		Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

// FindByPublicToken 仅按高熵公开令牌查询，避免公开接口回到可枚举的内部订单号。
func (r *OrderRepository) FindByPublicToken(ctx context.Context, token string) (order.Order, error) {
	if !order.IsValidPublicToken(token) {
		return order.Order{}, port.ErrNotFound
	}
	var row OrderRow
	err := databaseFromContext(ctx, r.db).Where("public_token = ?", token).Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) FindByPayID(ctx context.Context, payID string) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).Where("pay_id = ?", payID).Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) FindByPayIDForUpdate(ctx context.Context, payID string) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("pay_id = ?", payID).
		Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) List(ctx context.Context, filter port.OrderFilter) (port.OrderPage, error) {
	page, limit := normalizePage(filter.Page, filter.Limit)
	query := databaseFromContext(ctx, r.db).Model(&OrderRow{})
	if filter.State != nil {
		query = query.Where("state = ?", int(*filter.State))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return port.OrderPage{}, err
	}
	var rows []OrderRow
	if err := query.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return port.OrderPage{}, err
	}

	items := make([]order.Order, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.ToDomain())
	}
	return port.OrderPage{Items: items, Total: total}, nil
}

func (r *OrderRepository) Statistics(ctx context.Context, dayStart, dayEnd time.Time) (port.OrderStatistics, error) {
	var row struct {
		TodayOrders       int64 `gorm:"column:today_orders"`
		TodayPaidOrders   int64 `gorm:"column:today_paid_orders"`
		TodayClosedOrders int64 `gorm:"column:today_closed_orders"`
		TodayPaidCents    int64 `gorm:"column:today_paid_cents"`
		TotalOrders       int64 `gorm:"column:total_orders"`
		TotalPaidCents    int64 `gorm:"column:total_paid_cents"`
	}
	err := databaseFromContext(ctx, r.db).
		Model(&OrderRow{}).
		Select(`
			COUNT(*) AS total_orders,
			COALESCE(SUM(CASE WHEN state IN (?, ?) THEN price_cent ELSE 0 END), 0) AS total_paid_cents,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS today_orders,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND state IN (?, ?) THEN 1 ELSE 0 END), 0) AS today_paid_orders,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND state = ? THEN 1 ELSE 0 END), 0) AS today_closed_orders,
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND state IN (?, ?) THEN price_cent ELSE 0 END), 0) AS today_paid_cents`,
			int(order.StatusPaid), int(order.StatusNotifyFailed),
			dayStart, dayEnd,
			dayStart, dayEnd, int(order.StatusPaid), int(order.StatusNotifyFailed),
			dayStart, dayEnd, int(order.StatusClosed),
			dayStart, dayEnd, int(order.StatusPaid), int(order.StatusNotifyFailed),
		).
		Scan(&row).Error
	if err != nil {
		return port.OrderStatistics{}, err
	}
	return port.OrderStatistics{
		TodayOrders:       row.TodayOrders,
		TodayPaidOrders:   row.TodayPaidOrders,
		TodayClosedOrders: row.TodayClosedOrders,
		TodayPaidCents:    row.TodayPaidCents,
		TotalOrders:       row.TotalOrders,
		TotalPaidCents:    row.TotalPaidCents,
	}, nil
}

func (r *OrderRepository) Create(ctx context.Context, value order.Order) (order.Order, error) {
	if !order.IsValidPublicToken(value.PublicToken) {
		return order.Order{}, errors.New("公开订单令牌格式无效")
	}
	row := orderToRow(value)
	result := databaseFromContext(ctx, r.db).Create(&row)
	if isPublicTokenDuplicate(result.Error) {
		return order.Order{}, port.ErrPublicTokenConflict
	}
	if duplicateKey(result.Error) {
		return order.Order{}, port.ErrConflict
	}
	if result.Error != nil {
		return order.Order{}, result.Error
	}
	return row.ToDomain(), nil
}

func (r *OrderRepository) FindPendingForUpdate(ctx context.Context, typ payment.Type, reallyPriceCents int64) (order.Order, error) {
	var row OrderRow
	err := databaseFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("payment_type = ? AND state = ? AND really_price_cent = ?", int(typ), int(order.StatusPending), reallyPriceCents).
		Order("id ASC").
		Take(&row).Error
	return row.ToDomain(), mapDatabaseError(err)
}

func (r *OrderRepository) FindExpiredPendingForUpdate(ctx context.Context, before time.Time, limit int) ([]order.Order, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	var rows []OrderRow
	err := databaseFromContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("state = ? AND created_at < ?", int(order.StatusPending), before).
		Order("created_at ASC, id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]order.Order, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.ToDomain())
	}
	return items, nil
}

func (r *OrderRepository) Transition(ctx context.Context, id int64, from, to order.Status, changedAt time.Time) (bool, error) {
	if !order.CanTransition(from, to) {
		return false, errors.New("订单状态转换无效")
	}

	updates := map[string]any{"state": int(to)}
	switch {
	case from == order.StatusPending && to == order.StatusPaid:
		updates["paid_at"] = changedAt
	case to == order.StatusClosed:
		updates["closed_at"] = changedAt
	}
	result := databaseFromContext(ctx, r.db).
		Model(&OrderRow{}).
		Where("id = ? AND state = ?", id, int(from)).
		Updates(updates)
	return result.RowsAffected == 1, result.Error
}

func (r *OrderRepository) Delete(ctx context.Context, id int64) (bool, error) {
	result := databaseFromContext(ctx, r.db).Where("id = ?", id).Delete(&OrderRow{})
	return result.RowsAffected == 1, result.Error
}

func (r *OrderRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := databaseFromContext(ctx, r.db).Where("created_at < ?", before).Delete(&OrderRow{})
	return result.RowsAffected, result.Error
}

func (r *OrderRepository) DeleteExpiredBefore(ctx context.Context, before time.Time) (int64, error) {
	result := databaseFromContext(ctx, r.db).
		Where("created_at < ? AND state IN (?, ?)", before, int(order.StatusPending), int(order.StatusClosed)).
		Delete(&OrderRow{})
	return result.RowsAffected, result.Error
}

func orderToRow(value order.Order) OrderRow {
	return OrderRow{
		ID:              value.ID,
		OrderID:         value.OrderID,
		PublicToken:     value.PublicToken,
		PayID:           value.PayID,
		PaymentType:     int(value.Type),
		PriceCent:       value.PriceCents,
		ReallyPriceCent: value.ReallyPriceCents,
		PriceText:       value.PriceText,
		ReallyPriceText: value.ReallyPriceText,
		State:           int(value.State),
		Param:           value.Param,
		PayURL:          value.PayURL,
		IsAuto:          value.IsAuto,
		NotifyURL:       value.NotifyURL,
		ReturnURL:       value.ReturnURL,
		CreatedAt:       value.CreatedAt,
		PaidAt:          timePointer(value.PaidAt),
		ClosedAt:        timePointer(value.ClosedAt),
	}
}

func timePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copy := value
	return &copy
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

func duplicateKey(err error) bool {
	var driverError *drivermysql.MySQLError
	return errors.As(err, &driverError) && driverError.Number == 1062
}

// isPublicTokenDuplicate 只将公开令牌唯一索引冲突交给用例重试，其他唯一约束仍保持业务冲突语义。
func isPublicTokenDuplicate(err error) bool {
	var driverError *drivermysql.MySQLError
	message := ""
	if errors.As(err, &driverError) {
		message = strings.ToLower(driverError.Message)
	}
	return driverError != nil && driverError.Number == 1062 &&
		(strings.Contains(message, "uq_orders_public_token") || strings.Contains(message, "public_token"))
}

var _ port.OrderRepository = (*OrderRepository)(nil)
