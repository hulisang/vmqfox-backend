package service

import (
	"crypto/md5"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/repository"

	"gorm.io/gorm"
)

// 支付页面相关错误
var (
	ErrPaymentOrderNotFound = errors.New("payment order not found")
	ErrPaymentOrderExpired  = errors.New("payment order expired")
)

// PaymentService 支付页面服务接口
type PaymentService interface {
	GetPaymentOrder(orderID string) (*model.PaymentOrderResponse, error)
	CheckPaymentStatus(orderID string) (*model.PaymentStatusResponse, error)
	GeneratePaymentQrcode(req *model.PaymentQrcodeRequest) (*model.PaymentQrcodeResponse, error)
	GenerateReturnURL(orderID string) (string, error)
}

// paymentService 支付页面服务实现
type paymentService struct {
	orderRepo     repository.OrderRepository
	settingRepo   repository.SettingRepository
	qrcodeService QrcodeService
}

// NewPaymentService 创建支付页面服务
func NewPaymentService(orderRepo repository.OrderRepository, settingRepo repository.SettingRepository, qrcodeService QrcodeService) PaymentService {
	return &paymentService{
		orderRepo:     orderRepo,
		settingRepo:   settingRepo,
		qrcodeService: qrcodeService,
	}
}

// getOrderExpireMinutes 获取订单过期时间（分钟）
// 根据用户ID从setting表中读取close字段值，如果没有则使用默认值5分钟
func (s *paymentService) getOrderExpireMinutes(userID uint) int {
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

// GetPaymentOrder 获取支付页面订单信息
func (s *paymentService) GetPaymentOrder(orderID string) (*model.PaymentOrderResponse, error) {
	// 获取订单信息
	order, err := s.orderRepo.GetByOrderID(orderID)
	if err != nil {
		return nil, ErrPaymentOrderNotFound
	}

	// 获取该用户的订单过期时间配置（分钟）
	expireMinutes := s.getOrderExpireMinutes(order.User_id)
	expiredAt := order.Create_date + int64(expireMinutes*60) // 根据用户配置计算过期时间
	currentTime := time.Now().Unix()

	// 如果订单已过期且状态为待支付，自动关闭订单
	if order.State == model.OrderStatusPending && currentTime > expiredAt {
		order.State = model.OrderStatusClosed
		order.Close_date = currentTime
		s.orderRepo.Update(order)
	}

	// 转换为支付页面响应格式
	response := order.ToPaymentResponseWithExpireTime(expireMinutes)
	response.Is_expired = order.State == model.OrderStatusPending && currentTime > expiredAt

	return response, nil
}

// CheckPaymentStatus 检查支付状态
func (s *paymentService) CheckPaymentStatus(orderID string) (*model.PaymentStatusResponse, error) {
	// 获取订单信息
	order, err := s.orderRepo.GetByOrderID(orderID)
	if err != nil {
		return nil, ErrPaymentOrderNotFound
	}

	// 获取该用户的订单过期时间配置（分钟）
	expireMinutes := s.getOrderExpireMinutes(order.User_id)
	expiredAt := order.Create_date + int64(expireMinutes*60) // 根据用户配置计算过期时间
	currentTime := time.Now().Unix()
	isExpired := order.State == model.OrderStatusPending && currentTime > expiredAt

	// 如果订单已过期且状态为待支付，自动关闭订单
	if isExpired {
		order.State = model.OrderStatusClosed
		order.Close_date = currentTime
		s.orderRepo.Update(order)
	}

	// 计算剩余时间（秒）
	remainingSeconds := 0
	if order.State == model.OrderStatusPending && !isExpired {
		remainingSeconds = int(expiredAt - currentTime)
		if remainingSeconds < 0 {
			remainingSeconds = 0
		}
	}

	// 构建响应
	response := &model.PaymentStatusResponse{
		Order_id:         order.Order_id,
		State:            order.State,
		State_text:       order.GetStatusText(),
		Is_paid:          order.State == model.OrderStatusPaid,
		Is_expired:       isExpired,
		Pay_date:         order.Pay_date,
		RemainingSeconds: remainingSeconds,
	}

	// 根据状态设置消息
	switch order.State {
	case model.OrderStatusPending:
		if isExpired {
			response.Message = "订单已过期"
		} else {
			response.Message = "等待支付中"
		}
	case model.OrderStatusPaid:
		response.Message = "支付成功"
	case model.OrderStatusClosed:
		response.Message = "订单已关闭"
	default:
		response.Message = "未知状态"
	}

	return response, nil
}

// GeneratePaymentQrcode 生成支付页面二维码
func (s *paymentService) GeneratePaymentQrcode(req *model.PaymentQrcodeRequest) (*model.PaymentQrcodeResponse, error) {
	// 设置默认尺寸
	if req.Size == 0 {
		req.Size = 300
	}

	// 调用二维码服务生成二维码
	qrcodeReq := &model.GenerateQrcodeRequest{
		Text: req.URL,
		Size: req.Size,
	}

	qrcodeResp, err := s.qrcodeService.GenerateQrcode(qrcodeReq)
	if err != nil {
		return nil, err
	}

	// 转换为支付页面响应格式
	return &model.PaymentQrcodeResponse{
		Qrcode_url: qrcodeResp.Qrcode_url,
		Size:       qrcodeResp.Size,
		Format:     qrcodeResp.Format,
	}, nil
}

// GenerateReturnURL 生成返回URL
func (s *paymentService) GenerateReturnURL(orderID string) (string, error) {
	// 检查订单是否存在
	order, err := s.orderRepo.GetByOrderID(orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrPaymentOrderNotFound
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
