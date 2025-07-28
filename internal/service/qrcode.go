package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"

	"vmqfox-api-go/internal/model"
	"vmqfox-api-go/internal/repository"
)

var (
	ErrQrcodeNotFound = errors.New("qrcode not found")
	ErrQrcodeExists   = errors.New("qrcode already exists")
	ErrInvalidQrcode  = errors.New("invalid qrcode format")
)

type QrcodeService interface {
	GetQrcodes(req *model.QrcodeListRequest) ([]*model.PayQrcode, int64, error)
	CreateQrcode(userID uint, req *model.CreateQrcodeRequest) (*model.PayQrcode, error)
	DeleteQrcode(qrcodeID uint) error
	GetQrcodeByID(qrcodeID uint) (*model.PayQrcode, error)
	GetQrcodeByPriceAndType(price float64, qrcodeType int) (*model.PayQrcode, error) // 新增方法
	UpdateQrcodeStatus(qrcodeID uint, state int) error
	ParseQrcode(req *model.ParseQrcodeRequest) (*model.ParseQrcodeResponse, error)
	GenerateQrcode(req *model.GenerateQrcodeRequest) (*model.GenerateQrcodeResponse, error)
}

type qrcodeService struct {
	qrcodeRepo repository.QrcodeRepository
}

func NewQrcodeService(qrcodeRepo repository.QrcodeRepository) QrcodeService {
	return &qrcodeService{
		qrcodeRepo: qrcodeRepo,
	}
}

// GetQrcodes 获取收款码列表
func (s *qrcodeService) GetQrcodes(req *model.QrcodeListRequest) ([]*model.PayQrcode, int64, error) {
	return s.qrcodeRepo.GetQrcodes(req)
}

// CreateQrcode 创建收款码
func (s *qrcodeService) CreateQrcode(userID uint, req *model.CreateQrcodeRequest) (*model.PayQrcode, error) {
	// 检查是否已存在相同的收款码
	exists, err := s.qrcodeRepo.ExistsByUserAndPrice(userID, req.Price, req.Type)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrQrcodeExists
	}

	// 创建收款码
	qrcode := &model.PayQrcode{
		User_id: userID,
		Pay_url: req.Pay_url,
		Price:   req.Price,
		Type:    req.Type,
		State:   1, // 默认启用
	}

	err = s.qrcodeRepo.Create(qrcode)
	if err != nil {
		return nil, err
	}

	return qrcode, nil
}

// DeleteQrcode 删除收款码
func (s *qrcodeService) DeleteQrcode(qrcodeID uint) error {
	return s.qrcodeRepo.Delete(qrcodeID)
}

// GetQrcodeByID 根据ID获取收款码
func (s *qrcodeService) GetQrcodeByID(qrcodeID uint) (*model.PayQrcode, error) {
	qrcode, err := s.qrcodeRepo.GetByID(qrcodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQrcodeNotFound
		}
		return nil, err
	}
	return qrcode, nil
}

// UpdateQrcodeStatus 更新收款码状态
func (s *qrcodeService) UpdateQrcodeStatus(qrcodeID uint, state int) error {
	// 验证状态值
	if state != 0 && state != 1 {
		return errors.New("invalid state value, must be 0 or 1")
	}

	return s.qrcodeRepo.UpdateStatus(qrcodeID, state)
}

// ParseQrcode 解析收款码
func (s *qrcodeService) ParseQrcode(req *model.ParseQrcodeRequest) (*model.ParseQrcodeResponse, error) {
	// 解析收款码URL
	parsedURL, err := url.Parse(req.Qrcode_url)
	if err != nil {
		return nil, ErrInvalidQrcode
	}

	var paymentType int
	var amount float64

	// 根据域名判断支付类型
	if strings.Contains(parsedURL.Host, "alipay") {
		paymentType = model.OrderTypeAlipay
		amount, err = s.parseAlipayQrcode(req.Qrcode_url)
	} else if strings.Contains(parsedURL.Host, "pay.weixin") || strings.Contains(parsedURL.Host, "wxp.qq") {
		paymentType = model.OrderTypeWechat
		amount, err = s.parseWechatQrcode(req.Qrcode_url)
	} else {
		return nil, ErrInvalidQrcode
	}

	if err != nil {
		return nil, err
	}

	return &model.ParseQrcodeResponse{
		Type:   paymentType,
		Amount: amount,
		Url:    req.Qrcode_url,
	}, nil
}

// GenerateQrcode 生成二维码
func (s *qrcodeService) GenerateQrcode(req *model.GenerateQrcodeRequest) (*model.GenerateQrcodeResponse, error) {
	// 确定要生成二维码的内容：优先使用url参数，如果没有则使用text参数
	var content string
	if req.Url != "" {
		content = req.Url
	} else if req.Text != "" {
		content = req.Text
	} else {
		return nil, fmt.Errorf("either url or text parameter is required")
	}

	// 设置默认尺寸
	size := req.Size
	if size < 100 {
		size = 300 // 默认300px
	}

	// 使用 go-qrcode 库本地生成二维码
	qrCode, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to create qrcode: %w", err)
	}

	// 生成PNG格式的二维码图片数据
	pngData, err := qrCode.PNG(size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PNG: %w", err)
	}

	// 转换为base64编码，便于前端直接使用
	base64Data := base64.StdEncoding.EncodeToString(pngData)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Data)

	return &model.GenerateQrcodeResponse{
		Qrcode_url: dataURL, // 返回base64数据URL
		Size:       size,
		Format:     "PNG",
	}, nil
}

// parseAlipayQrcode 解析支付宝收款码
func (s *qrcodeService) parseAlipayQrcode(qrcodeURL string) (float64, error) {
	// 支付宝收款码格式示例：https://qr.alipay.com/fkx14847me57yiilohnqtcb
	// 这里简化处理，实际需要调用支付宝API解析

	// 从URL中提取金额信息（如果有）
	parsedURL, err := url.Parse(qrcodeURL)
	if err != nil {
		return 0, err
	}

	// 检查查询参数中是否有金额
	if amountStr := parsedURL.Query().Get("amount"); amountStr != "" {
		return strconv.ParseFloat(amountStr, 64)
	}

	// 如果没有金额信息，返回0表示动态金额
	return 0, nil
}

// parseWechatQrcode 解析微信收款码
func (s *qrcodeService) parseWechatQrcode(qrcodeURL string) (float64, error) {
	// 微信收款码格式示例：wxp://f2f0xxx
	// 这里简化处理，实际需要调用微信API解析

	// 使用正则表达式提取金额（如果URL中包含）
	re := regexp.MustCompile(`amount=([0-9.]+)`)
	matches := re.FindStringSubmatch(qrcodeURL)
	if len(matches) > 1 {
		return strconv.ParseFloat(matches[1], 64)
	}

	// 如果没有金额信息，返回0表示动态金额
	return 0, nil
}

// GetQrcodeByPriceAndType 根据价格和类型获取收款码
func (s *qrcodeService) GetQrcodeByPriceAndType(price float64, qrcodeType int) (*model.PayQrcode, error) {
	return s.qrcodeRepo.GetQrcodeByPriceAndType(price, qrcodeType)
}
