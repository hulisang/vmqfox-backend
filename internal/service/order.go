package service

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/repository"

	"gorm.io/gorm"
)

// 订单相关错误
var (
	ErrOrderNotFound    = errors.New("order not found")
	ErrOrderExists      = errors.New("order already exists")
	ErrOrderExpired     = errors.New("order expired")
	ErrOrderPaid        = errors.New("order already paid")
	ErrOrderClosed      = errors.New("order closed")
	ErrInvalidAmount    = errors.New("invalid amount")
	ErrInvalidOrderType = errors.New("invalid order type")
)

// OrderService 订单服务接口
type OrderService interface {
	GetOrders(req *model.OrderListRequest) ([]*model.Order, int64, error)
	GetOrderByID(id uint) (*model.Order, error)
	GetOrderByOrderID(orderID string) (*model.Order, error)
	GetOrderByPayID(payID string) (*model.Order, error) // 新增：根据商户订单号查询
	GetOrderStatus(orderID string) (*model.OrderStatusResponse, error)
	CreateOrder(userID uint, req *model.CreateOrderRequest, clientIP, userAgent string) (*model.Order, error)
	CreatePublicOrder(order *model.Order) (*model.Order, error) // 新增：公开API创建订单
	UpdateOrder(id uint, req *model.UpdateOrderRequest) (*model.Order, error)
	DeleteOrder(id uint) error
	CloseOrder(id uint) error
	CloseExpiredOrder(orderID string) error // 新增：关闭指定订单
	CloseExpiredOrders(req *model.CloseExpiredOrdersRequest) (int64, error)
	DeleteExpiredOrders(req *model.DeleteExpiredOrdersRequest) (int64, error)
	GenerateReturnURL(orderID string) (string, error)
}

// orderService 订单服务实现
type orderService struct {
	orderRepo   repository.OrderRepository
	userRepo    repository.UserRepository
	settingRepo repository.SettingRepository
}

// NewOrderService 创建订单服务
func NewOrderService(orderRepo repository.OrderRepository, userRepo repository.UserRepository, settingRepo repository.SettingRepository) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		userRepo:    userRepo,
		settingRepo: settingRepo,
	}
}

// GetOrders 获取订单列表
func (s *orderService) GetOrders(req *model.OrderListRequest) ([]*model.Order, int64, error) {
	return s.orderRepo.GetOrders(req)
}

// GetOrderByID 根据ID获取订单
func (s *orderService) GetOrderByID(id uint) (*model.Order, error) {
	order, err := s.orderRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

// GetOrderByOrderID 根据订单号获取订单
func (s *orderService) GetOrderByOrderID(orderID string) (*model.Order, error) {
	order, err := s.orderRepo.GetByOrderIDWithUser(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

// GetOrderStatus 获取订单状态
func (s *orderService) GetOrderStatus(orderID string) (*model.OrderStatusResponse, error) {
	order, err := s.orderRepo.GetByOrderID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	return &model.OrderStatusResponse{
		Order_id:   order.Order_id,
		State:      order.State,
		State_text: order.GetStatusText(),
		Is_paid:    order.IsPaid(),
		Is_expired: order.IsExpired(),
		Pay_date:   order.Pay_date,
	}, nil
}

// CreateOrder 创建订单
func (s *orderService) CreateOrder(userID uint, req *model.CreateOrderRequest, clientIP, userAgent string) (*model.Order, error) {
	// 验证用户是否存在
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 验证金额
	if req.Price <= 0 {
		return nil, ErrInvalidAmount
	}

	// 验证订单类型
	if req.Type != model.OrderTypeAlipay && req.Type != model.OrderTypeWechat {
		return nil, ErrInvalidOrderType
	}

	// 生成订单号
	var typeStr string
	if req.Type == model.OrderTypeAlipay {
		typeStr = "alipay"
	} else {
		typeStr = "wechat"
	}
	orderID := s.generateOrderID(userID, typeStr)

	// 检查订单号是否已存在（极小概率）
	exists, err := s.orderRepo.ExistsByOrderID(orderID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrderExists
	}

	// 创建订单 - 使用简化字段名
	// 将Subject和Body存储在Param字段中（JSON格式）
	paramData := map[string]string{
		"subject": req.Subject,
		"body":    req.Body,
	}
	paramJSON, _ := json.Marshal(paramData)

	// 生成商户订单号（Pay_id）
	payID := s.generatePayID(userID, typeStr)

	// 设置支付URL（Pay_url）- 与PHP版本逻辑一致
	payURL := s.generatePayURL(userID, req.Type, req.Price)

	order := &model.Order{
		Order_id:     orderID,
		User_id:      userID,
		Type:         req.Type,
		Price:        req.Price,
		Really_price: req.Price,
		State:        model.OrderStatusPending,
		Notify_url:   req.Notify_url,
		Return_url:   req.Return_url,
		Pay_id:       payID,
		Pay_url:      payURL,
		Param:        string(paramJSON),
		Is_auto:      1, // 自动订单
	}

	if err := s.orderRepo.Create(order); err != nil {
		return nil, err
	}

	return order, nil
}

// UpdateOrder 更新订单
func (s *orderService) UpdateOrder(id uint, req *model.UpdateOrderRequest) (*model.Order, error) {
	// 获取现有订单
	order, err := s.orderRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// 检查订单状态（只有待支付的订单才能更新）
	if order.State != model.OrderStatusPending {
		return nil, ErrOrderClosed
	}

	// 更新字段 - 适配现有数据库结构
	needUpdate := false

	// 解析现有的Param字段
	var paramData map[string]string
	if order.Param != "" {
		json.Unmarshal([]byte(order.Param), &paramData)
	} else {
		paramData = make(map[string]string)
	}

	// 更新Subject和Body
	if req.Subject != "" {
		paramData["subject"] = req.Subject
		needUpdate = true
	}
	if req.Body != "" {
		paramData["body"] = req.Body
		needUpdate = true
	}

	// 更新Param字段
	if needUpdate {
		paramJSON, _ := json.Marshal(paramData)
		order.Param = string(paramJSON)
	}

	// 更新URL字段
	if req.Notify_url != "" {
		order.Notify_url = req.Notify_url
	}
	if req.Return_url != "" {
		order.Return_url = req.Return_url
	}

	// 保存更新
	if err := s.orderRepo.Update(order); err != nil {
		return nil, err
	}

	return order, nil
}

// DeleteOrder 删除订单
func (s *orderService) DeleteOrder(id uint) error {
	// 检查订单是否存在
	order, err := s.orderRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrderNotFound
		}
		return err
	}

	// 检查订单状态（已支付的订单不能删除）
	if order.IsPaid() {
		return ErrOrderPaid
	}

	// 删除订单
	return s.orderRepo.Delete(id)
}

// CloseOrder 关闭订单
func (s *orderService) CloseOrder(id uint) error {
	// 获取订单
	order, err := s.orderRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrderNotFound
		}
		return err
	}

	// 检查订单状态
	if order.IsPaid() {
		return ErrOrderPaid
	}
	if order.IsClosed() {
		return ErrOrderClosed
	}

	// 更新订单状态为已关闭
	return s.orderRepo.UpdateStatus(id, model.OrderStatusClosed)
}

// getOrderExpireMinutes 获取订单过期时间（分钟）
// 根据用户ID从setting表中读取close字段值，如果没有则使用默认值5分钟
func (s *orderService) getOrderExpireMinutes(userID uint) int {
	setting, err := s.settingRepo.GetSetting(userID, "close")
	if err != nil {
		// 如果获取失败，使用默认值5分钟
		return 5
	}

	minutes, err := strconv.Atoi(setting.Vvalue)
	if err != nil || minutes <= 0 {
		// 如果转换失败或值无效，使用默认值5分钟
		return 5
	}

	return minutes
}

// closeExpiredOrdersForAllUsers 为所有用户关闭过期订单，使用各自的配置
func (s *orderService) closeExpiredOrdersForAllUsers(limit int) (int64, error) {
	// 获取所有有待支付订单的用户ID
	userIDs, err := s.orderRepo.GetUsersWithPendingOrders()
	if err != nil {
		return 0, err
	}

	totalClosed := int64(0)
	for _, userID := range userIDs {
		// 获取该用户的过期时间配置
		expireMinutes := s.getOrderExpireMinutes(userID)

		// 为该用户关闭过期订单
		closed, err := s.orderRepo.CloseExpiredOrdersWithMinutes(&userID, limit, expireMinutes)
		if err != nil {
			// 记录错误但继续处理其他用户
			log.Printf("关闭用户%d的过期订单失败: %v", userID, err)
			continue
		}

		totalClosed += closed

		// 如果已经达到限制，停止处理
		if totalClosed >= int64(limit) {
			break
		}
	}

	return totalClosed, nil
}

// CloseExpiredOrders 关闭过期订单
func (s *orderService) CloseExpiredOrders(req *model.CloseExpiredOrdersRequest) (int64, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // 默认限制100条
	}

	// 如果指定了用户ID，使用该用户的过期时间设置
	if req.User_id != nil {
		expireMinutes := s.getOrderExpireMinutes(*req.User_id)
		return s.orderRepo.CloseExpiredOrdersWithMinutes(req.User_id, limit, expireMinutes)
	}

	// 如果没有指定用户ID，需要处理所有用户的过期订单
	// 获取所有有订单的用户，分别使用各自的过期时间配置
	return s.closeExpiredOrdersForAllUsers(limit)
}

// DeleteExpiredOrders 删除过期订单
func (s *orderService) DeleteExpiredOrders(req *model.DeleteExpiredOrdersRequest) (int64, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // 默认限制100条
	}

	expireDays := req.ExpireDays
	if expireDays <= 0 {
		expireDays = 30 // 默认30天
	}

	// 默认只删除已关闭的订单
	onlyClosed := true
	if req.OnlyClosed == false {
		onlyClosed = false
	}

	return s.orderRepo.DeleteExpiredOrders(req.User_id, limit, onlyClosed, expireDays)
}

// GenerateReturnURL 生成返回URL
func (s *orderService) GenerateReturnURL(orderID string) (string, error) {
	// 检查订单是否存在
	order, err := s.orderRepo.GetByOrderID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrOrderNotFound
		}
		return "", err
	}

	// 检查订单是否已支付
	if order.State != model.OrderStatusPaid {
		return "", fmt.Errorf("订单未支付，无法生成返回URL")
	}

	// 检查是否有返回URL
	if order.Return_url == "" {
		return "", fmt.Errorf("订单未设置返回URL")
	}

	// 获取用户密钥
	key, err := s.settingRepo.GetSetting(order.User_id, "key")
	if err != nil {
		return "", fmt.Errorf("获取用户密钥失败: %v", err)
	}

	// 格式化价格，保证精度一致
	priceStr := fmt.Sprintf("%.2f", order.Price)
	reallyPriceStr := fmt.Sprintf("%.2f", order.Really_price)

	// 构建参数字符串
	params := fmt.Sprintf("payId=%s&param=%s&type=%d&price=%s&reallyPrice=%s",
		order.Pay_id, order.Param, order.Type, priceStr, reallyPriceStr)

	// 计算签名：payId + param + type + price + reallyPrice + key
	signStr := order.Pay_id + order.Param + fmt.Sprintf("%d", order.Type) + priceStr + reallyPriceStr + key.Vvalue
	sign := fmt.Sprintf("%x", md5.Sum([]byte(signStr)))

	// 添加签名到参数
	params += "&sign=" + sign

	// 构建完整URL
	returnURL := order.Return_url
	separator := "?"
	if strings.Contains(returnURL, "?") {
		separator = "&"
	}
	fullReturnURL := returnURL + separator + params

	return fullReturnURL, nil
}

// generateOrderID 生成订单号
func (s *orderService) generateOrderID(userID uint, orderType string) string {
	// 格式: VMQ + 时间戳 + 用户ID + 随机数
	timestamp := time.Now().Format("20060102150405")

	// 生成MD5哈希作为随机部分
	data := fmt.Sprintf("%d_%s_%d", userID, orderType, time.Now().UnixNano())
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	// 取前6位作为随机部分
	random := hash[:6]

	return fmt.Sprintf("VMQ%s%04d%s", timestamp, userID%10000, random)
}

// generatePayID 生成商户订单号
func (s *orderService) generatePayID(userID uint, orderType string) string {
	// 格式: PAY + 时间戳 + 用户ID + 随机数
	timestamp := time.Now().Format("20060102150405")

	// 生成MD5哈希作为随机部分
	data := fmt.Sprintf("pay_%d_%s_%d", userID, orderType, time.Now().UnixNano())
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	// 取前8位作为随机部分
	random := hash[:8]

	return fmt.Sprintf("PAY%s%04d%s", timestamp, userID%10000, random)
}

// generatePayURL 生成支付URL - 与PHP版本逻辑一致
func (s *orderService) generatePayURL(userID uint, orderType int, price float64) string {
	// 1. 优先查找与金额完全匹配的收款码
	qrcode, err := s.orderRepo.GetPayQrcode(userID, price, orderType)
	if err == nil && qrcode.Pay_url != "" {
		// 找到匹配的收款码，使用固定的URL
		return qrcode.Pay_url
	}

	// 2. 降级使用用户配置的通用收款码
	var settingKey string
	if orderType == model.OrderTypeAlipay {
		settingKey = "zfbpay" // 支付宝
	} else {
		settingKey = "wxpay" // 微信
	}

	setting, err := s.orderRepo.GetUserSetting(userID, settingKey)
	if err == nil && setting.Vvalue != "" {
		return setting.Vvalue
	}

	// 3. 如果都没有找到，返回测试用的URL
	var paymentMethod string
	if orderType == model.OrderTypeAlipay {
		paymentMethod = "alipay"
	} else {
		paymentMethod = "wechat"
	}

	return fmt.Sprintf("https://test-payment.vmqfox.com/%s?amount=%.2f", paymentMethod, price)
}

// GetOrderByPayID 根据商户订单号查询订单
func (s *orderService) GetOrderByPayID(payID string) (*model.Order, error) {
	return s.orderRepo.GetOrderByPayID(payID)
}

// CreatePublicOrder 创建公开订单（供第三方商户使用）
func (s *orderService) CreatePublicOrder(order *model.Order) (*model.Order, error) {
	// 直接创建订单，不需要额外的权限检查
	return s.orderRepo.CreateOrder(order)
}

// CloseExpiredOrder 关闭指定的过期订单
func (s *orderService) CloseExpiredOrder(orderID string) error {
	order, err := s.orderRepo.GetOrderByOrderID(orderID)
	if err != nil {
		return err
	}

	if order.State != 0 {
		return nil // 订单已经不是未支付状态，无需关闭
	}

	// 更新订单状态为已关闭
	order.State = model.OrderStatusClosed
	order.Close_date = time.Now().Unix()

	return s.orderRepo.UpdateOrder(order)
}
