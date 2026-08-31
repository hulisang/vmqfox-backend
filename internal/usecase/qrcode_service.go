package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

const maxQRCodePayURLBytes = 4096

type ListQRCodesInput struct {
	Type  *payment.Type
	Page  int
	Limit int
}

type CreateQRCodeInput struct {
	Type   payment.Type
	PayURL string
	Price  string
}

type QRCodeServiceDeps struct {
	QRCodes port.QRCodeRepository
}

// QRCodeService 统一校验支付类型和整数分金额，持久化细节留在 repository adapter。
type QRCodeService struct {
	qrcodes port.QRCodeRepository
}

func NewQRCodeService(deps QRCodeServiceDeps) (*QRCodeService, error) {
	if deps.QRCodes == nil {
		return nil, errors.New("二维码管理用例依赖不完整")
	}
	return &QRCodeService{qrcodes: deps.QRCodes}, nil
}

func (s *QRCodeService) List(ctx context.Context, input ListQRCodesInput) (port.QRCodePage, error) {
	if input.Type != nil && !input.Type.Valid() {
		return port.QRCodePage{}, fail(CodeInvalidArgument, "支付类型错误")
	}
	page, err := s.qrcodes.List(ctx, port.QRCodeFilter{
		Type:  input.Type,
		Page:  input.Page,
		Limit: input.Limit,
	})
	if err != nil {
		return port.QRCodePage{}, wrap(CodeDependency, "查询二维码列表失败", err)
	}
	return page, nil
}

func (s *QRCodeService) Create(ctx context.Context, input CreateQRCodeInput) (qrcode.QRCode, error) {
	if !input.Type.Valid() {
		return qrcode.QRCode{}, fail(CodeInvalidArgument, "支付类型错误")
	}
	payURL := strings.TrimSpace(input.PayURL)
	if payURL == "" {
		return qrcode.QRCode{}, fail(CodeInvalidArgument, "收款码不能为空")
	}
	if len([]byte(payURL)) > maxQRCodePayURLBytes {
		return qrcode.QRCode{}, fail(CodeInvalidArgument, "收款码内容过长")
	}

	price := strings.TrimSpace(input.Price)
	if price == "" {
		price = "0"
	}
	priceCents, err := order.ParseAmountCents(price)
	if err != nil {
		return qrcode.QRCode{}, fail(CodeInvalidArgument, "二维码金额无效")
	}

	created, err := s.qrcodes.Create(ctx, qrcode.QRCode{
		PayURL:     payURL,
		PriceCents: priceCents,
		Type:       input.Type,
		State:      qrcode.StateEnabled,
	})
	if errors.Is(err, port.ErrConflict) {
		return qrcode.QRCode{}, fail(CodeConflict, "二维码记录冲突")
	}
	if err != nil {
		return qrcode.QRCode{}, wrap(CodeDependency, "添加二维码失败", err)
	}
	return created, nil
}

func (s *QRCodeService) SetState(ctx context.Context, id int64, state qrcode.State) error {
	if id <= 0 {
		return fail(CodeInvalidArgument, "二维码ID无效")
	}
	if !state.Valid() {
		return fail(CodeInvalidArgument, "二维码状态无效")
	}
	current, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if current.State == state {
		return nil
	}
	changed, err := s.qrcodes.SetState(ctx, id, current.State, state)
	if err != nil {
		return wrap(CodeDependency, "设置二维码状态失败", err)
	}
	if !changed {
		return fail(CodeConflict, "二维码状态已变化，请重试")
	}
	return nil
}

func (s *QRCodeService) Delete(ctx context.Context, id int64, expectedType *payment.Type) error {
	if id <= 0 {
		return fail(CodeInvalidArgument, "二维码ID无效")
	}
	if expectedType != nil && !expectedType.Valid() {
		return fail(CodeInvalidArgument, "支付类型错误")
	}
	current, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if expectedType != nil && current.Type != *expectedType {
		return fail(CodeInvalidArgument, "该二维码不是"+paymentTypeName(*expectedType)+"二维码")
	}
	deleted, err := s.qrcodes.Delete(ctx, id)
	if err != nil {
		return wrap(CodeDependency, "删除二维码失败", err)
	}
	if !deleted {
		return fail(CodeNotFound, "二维码不存在")
	}
	return nil
}

func (s *QRCodeService) find(ctx context.Context, id int64) (qrcode.QRCode, error) {
	value, err := s.qrcodes.FindByID(ctx, id)
	if errors.Is(err, port.ErrNotFound) {
		return qrcode.QRCode{}, fail(CodeNotFound, "二维码不存在")
	}
	if err != nil {
		return qrcode.QRCode{}, wrap(CodeDependency, "查询二维码失败", err)
	}
	return value, nil
}

func paymentTypeName(value payment.Type) string {
	if value == payment.Wechat {
		return "微信"
	}
	return "支付宝"
}
