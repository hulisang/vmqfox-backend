package repository

import (
	"time"
	"vmqfox-api-go/internal/model"

	"gorm.io/gorm"
)

// OrderRepository 订单仓库接口
type OrderRepository interface {
	GetByID(id uint) (*model.Order, error)
	GetByOrderID(orderID string) (*model.Order, error)
	GetByOrderIDWithUser(orderID string) (*model.Order, error)
	GetByPayID(payID string) (*model.Order, error) // 新增：根据商户订单号查询
	GetOrders(req *model.OrderListRequest) ([]*model.Order, int64, error)
	Create(order *model.Order) error
	CreateOrder(order *model.Order) (*model.Order, error) // 新增：创建订单并返回
	Update(order *model.Order) error
	UpdateOrder(order *model.Order) error                   // 新增：更新订单的别名
	GetOrderByOrderID(orderID string) (*model.Order, error) // 新增：别名方法
	GetOrderByPayID(payID string) (*model.Order, error)     // 新增：别名方法
	Delete(id uint) error
	UpdateStatus(id uint, status int) error
	CloseExpiredOrders(userID *uint, limit int) (int64, error)
	CloseExpiredOrdersWithMinutes(userID *uint, limit int, expireMinutes int) (int64, error)
	DeleteExpiredOrders(userID *uint, limit int, onlyClosed bool, expireDays int) (int64, error)
	GetExpiredOrders(userID *uint, limit int) ([]*model.Order, error)
	GetExpiredOrdersWithMinutes(userID *uint, limit int, expireMinutes int) ([]*model.Order, error)
	GetUsersWithPendingOrders() ([]uint, error)
	GetRecentPendingOrderByPriceAndType(userID uint, price float64, orderType int) (*model.Order, error)
	ExistsByOrderID(orderID string) (bool, error)
	GetPayQrcode(userID uint, price float64, orderType int) (*model.PayQrcode, error)
	GetUserSetting(userID uint, key string) (*model.Setting, error)
	GetOrderStatsByDateRange(userID uint, startTime, endTime int64) (*model.OrderStats, error)
}

// orderRepository 订单仓库实现
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓库
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// GetByID 根据ID获取订单
func (r *orderRepository) GetByID(id uint) (*model.Order, error) {
	var order model.Order
	err := r.db.First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderID 根据订单号获取订单
func (r *orderRepository) GetByOrderID(orderID string) (*model.Order, error) {
	var order model.Order
	err := r.db.Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetByOrderIDWithUser 根据订单号获取订单（包含用户信息）
func (r *orderRepository) GetByOrderIDWithUser(orderID string) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("User").Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrders 获取订单列表
func (r *orderRepository) GetOrders(req *model.OrderListRequest) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	query := r.db.Model(&model.Order{})

	// 构建查询条件
	if req.State != nil {
		query = query.Where("state = ?", *req.State)
	}
	if req.Type != nil {
		query = query.Where("type = ?", *req.Type)
	}
	if req.Order_id != "" {
		query = query.Where("order_id LIKE ?", "%"+req.Order_id+"%")
	}
	if req.User_id != nil {
		query = query.Where("user_id = ?", *req.User_id)
	}
	if req.Start_at != "" {
		if startTime, err := time.Parse("2006-01-02", req.Start_at); err == nil {
			query = query.Where("create_date >= ?", startTime.Unix())
		}
	}
	if req.End_at != "" {
		if endTime, err := time.Parse("2006-01-02", req.End_at); err == nil {
			// 结束时间加一天，包含当天的所有记录
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("create_date < ?", endTime.Unix())
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit
	if err := query.Preload("User").Offset(offset).Limit(limit).Order("create_date DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// Create 创建订单
func (r *orderRepository) Create(order *model.Order) error {
	return r.db.Create(order).Error
}

// Update 更新订单
func (r *orderRepository) Update(order *model.Order) error {
	return r.db.Save(order).Error
}

// Delete 删除订单（软删除）
func (r *orderRepository) Delete(id uint) error {
	return r.db.Delete(&model.Order{}, id).Error
}

// UpdateStatus 更新订单状态
func (r *orderRepository) UpdateStatus(id uint, status int) error {
	updates := map[string]interface{}{
		"state": status,
	}

	// 根据状态设置相应的时间字段
	switch status {
	case model.OrderStatusPaid:
		updates["pay_date"] = time.Now().Unix()
	case model.OrderStatusClosed:
		updates["close_date"] = time.Now().Unix()
	}

	return r.db.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// CloseExpiredOrders 关闭过期订单（使用默认5分钟，与PHP版本保持一致）
func (r *orderRepository) CloseExpiredOrders(userID *uint, limit int) (int64, error) {
	return r.CloseExpiredOrdersWithMinutes(userID, limit, 5)
}

// CloseExpiredOrdersWithMinutes 关闭过期订单（使用指定分钟数）
func (r *orderRepository) CloseExpiredOrdersWithMinutes(userID *uint, limit int, expireMinutes int) (int64, error) {
	// 计算过期时间点
	expireTime := time.Now().Add(-time.Duration(expireMinutes) * time.Minute).Unix()

	query := r.db.Model(&model.Order{}).
		Where("state = ?", model.OrderStatusPending).
		Where("create_date < ?", expireTime)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	updates := map[string]interface{}{
		"state":      model.OrderStatusClosed,
		"close_date": time.Now().Unix(),
	}

	// 先获取要关闭的订单列表，用于清理tmp_price表
	var ordersToClose []*model.Order
	if err := query.Find(&ordersToClose).Error; err != nil {
		return 0, err
	}

	if len(ordersToClose) == 0 {
		return 0, nil
	}

	// 批量更新订单状态
	result := r.db.Model(&model.Order{}).
		Where("state = ?", model.OrderStatusPending).
		Where("create_date < ?", expireTime)

	if userID != nil {
		result = result.Where("user_id = ?", *userID)
	}

	if limit > 0 {
		// 获取要更新的订单ID列表
		var orderIDs []uint
		for _, order := range ordersToClose {
			orderIDs = append(orderIDs, order.Id)
		}
		result = result.Where("id IN ?", orderIDs)
	}

	updateResult := result.Updates(updates)
	if updateResult.Error != nil {
		return 0, updateResult.Error
	}

	// 清理tmp_price表中对应的记录
	if updateResult.RowsAffected > 0 {
		var orderIDsToDelete []string
		for _, order := range ordersToClose {
			orderIDsToDelete = append(orderIDsToDelete, order.Order_id)
		}

		if len(orderIDsToDelete) > 0 {
			// 删除tmp_price表中的记录
			r.db.Where("oid IN ?", orderIDsToDelete).Delete(&model.TmpPrice{})
		}
	}

	return updateResult.RowsAffected, updateResult.Error
}

// GetExpiredOrders 获取过期订单（使用默认30分钟）
func (r *orderRepository) GetExpiredOrders(userID *uint, limit int) ([]*model.Order, error) {
	return r.GetExpiredOrdersWithMinutes(userID, limit, 30)
}

// GetExpiredOrdersWithMinutes 获取过期订单（使用指定分钟数）
func (r *orderRepository) GetExpiredOrdersWithMinutes(userID *uint, limit int, expireMinutes int) ([]*model.Order, error) {
	var orders []*model.Order

	// 计算过期时间点
	expireTime := time.Now().Add(-time.Duration(expireMinutes) * time.Minute).Unix()

	query := r.db.Where("state = ?", model.OrderStatusPending).
		Where("create_date < ?", expireTime)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("create_date ASC").Find(&orders).Error
	return orders, err
}

// DeleteExpiredOrders 删除过期订单
func (r *orderRepository) DeleteExpiredOrders(userID *uint, limit int, onlyClosed bool, expireDays int) (int64, error) {
	// 计算过期时间（默认30天）
	if expireDays <= 0 {
		expireDays = 30
	}
	expireTime := time.Now().AddDate(0, 0, -expireDays).Unix()

	// 构建查询条件
	query := r.db.Model(&model.Order{}).Where("create_date < ?", expireTime)

	// 如果只删除已关闭的订单
	if onlyClosed {
		query = query.Where("state = ?", model.OrderStatusClosed)
	} else {
		// 否则删除已关闭和待支付的订单，但不删除已支付的订单
		query = query.Where("state != ?", model.OrderStatusPaid)
	}

	// 如果指定了用户ID
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// 如果指定了限制数量
	if limit > 0 {
		query = query.Limit(limit)
	}

	// 执行删除操作
	result := query.Delete(&model.Order{})
	return result.RowsAffected, result.Error
}

// ExistsByOrderID 检查订单号是否存在
func (r *orderRepository) ExistsByOrderID(orderID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Order{}).Where("order_id = ?", orderID).Count(&count).Error
	return count > 0, err
}

// GetPayQrcode 获取收款码
func (r *orderRepository) GetPayQrcode(userID uint, price float64, orderType int) (*model.PayQrcode, error) {
	var qrcode model.PayQrcode
	err := r.db.Where("user_id = ? AND price = ? AND type = ? AND state = 1", userID, price, orderType).
		First(&qrcode).Error
	if err != nil {
		return nil, err
	}
	return &qrcode, nil
}

// GetUserSetting 获取用户设置
func (r *orderRepository) GetUserSetting(userID uint, key string) (*model.Setting, error) {
	var setting model.Setting
	err := r.db.Where("vkey = ? AND user_id = ?", key, userID).
		First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetOrderStatsByDateRange 获取指定时间范围内的订单统计
func (r *orderRepository) GetOrderStatsByDateRange(userID uint, startTime, endTime int64) (*model.OrderStats, error) {
	var stats model.OrderStats

	query := r.db.Model(&model.Order{})

	// 添加用户过滤
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	// 添加时间范围过滤
	if startTime > 0 {
		query = query.Where("create_date >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("create_date < ?", endTime)
	}

	// 统计总订单数
	err := query.Count(&stats.TotalOrders).Error
	if err != nil {
		return nil, err
	}

	// 统计成功订单数和金额 - 创建新的查询避免污染原查询
	successQuery := r.db.Model(&model.Order{})
	if userID > 0 {
		successQuery = successQuery.Where("user_id = ?", userID)
	}
	if startTime > 0 {
		successQuery = successQuery.Where("create_date >= ?", startTime)
	}
	if endTime > 0 {
		successQuery = successQuery.Where("create_date < ?", endTime)
	}
	successQuery = successQuery.Where("state >= ?", model.OrderStatusPaid)

	err = successQuery.Count(&stats.SuccessOrders).Error
	if err != nil {
		return nil, err
	}

	// 统计成功订单总金额
	var totalAmount float64
	err = successQuery.Select("COALESCE(SUM(price), 0)").Scan(&totalAmount).Error
	if err != nil {
		return nil, err
	}
	stats.TotalAmount = totalAmount

	// 统计关闭订单数 - 创建新的查询避免污染原查询
	closedQuery := r.db.Model(&model.Order{})
	if userID > 0 {
		closedQuery = closedQuery.Where("user_id = ?", userID)
	}
	if startTime > 0 {
		closedQuery = closedQuery.Where("create_date >= ?", startTime)
	}
	if endTime > 0 {
		closedQuery = closedQuery.Where("create_date < ?", endTime)
	}
	closedQuery = closedQuery.Where("state = ?", model.OrderStatusClosed)

	err = closedQuery.Count(&stats.ClosedOrders).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetByPayID 根据商户订单号获取订单
func (r *orderRepository) GetByPayID(payID string) (*model.Order, error) {
	var order model.Order
	err := r.db.Where("pay_id = ?", payID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// CreateOrder 创建订单并返回
func (r *orderRepository) CreateOrder(order *model.Order) (*model.Order, error) {
	err := r.db.Create(order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

// UpdateOrder 更新订单（别名方法）
func (r *orderRepository) UpdateOrder(order *model.Order) error {
	return r.Update(order)
}

// GetOrderByOrderID 根据订单ID获取订单（别名方法）
func (r *orderRepository) GetOrderByOrderID(orderID string) (*model.Order, error) {
	return r.GetByOrderID(orderID)
}

// GetOrderByPayID 根据商户订单号获取订单（别名方法）
func (r *orderRepository) GetOrderByPayID(payID string) (*model.Order, error) {
	return r.GetByPayID(payID)
}

// GetUsersWithPendingOrders 获取所有有待支付订单的用户ID
func (r *orderRepository) GetUsersWithPendingOrders() ([]uint, error) {
	var userIDs []uint
	err := r.db.Model(&model.Order{}).
		Where("state = ?", model.OrderStatusPending).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// GetRecentPendingOrderByPriceAndType 根据价格和类型获取最近的待支付订单
func (r *orderRepository) GetRecentPendingOrderByPriceAndType(userID uint, price float64, orderType int) (*model.Order, error) {
	var order model.Order
	// 先尝试匹配 really_price（实际支付金额）
	err := r.db.Where("user_id = ? AND really_price = ? AND type = ? AND state = ?",
		userID, price, orderType, model.OrderStatusPending).
		Order("create_date DESC").
		First(&order).Error

	if err != nil {
		// 如果没找到，再尝试匹配 price（订单原始金额）
		err = r.db.Where("user_id = ? AND price = ? AND type = ? AND state = ?",
			userID, price, orderType, model.OrderStatusPending).
			Order("create_date DESC").
			First(&order).Error
	}

	if err != nil {
		return nil, err
	}
	return &order, nil
}
