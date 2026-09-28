package usecase

import (
	"context"
	"testing"

	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
	"github.com/hulisang/vmqfox-backend/internal/domain/qrcode"
	"github.com/hulisang/vmqfox-backend/internal/domain/setting"
)

// TestResolvePayURLPriority 锁定选码顺序：精确金额固定码 → 码库通用码 → 系统设置兜底码 → 报错。
func TestResolvePayURLPriority(t *testing.T) {
	const amount int64 = 1001
	fixedAlipay := qrcode.QRCode{PayURL: "https://qr.alipay.com/fixed", PriceCents: amount, Type: payment.Alipay}
	genericAlipay := qrcode.QRCode{PayURL: "https://qr.alipay.com/generic", PriceCents: qrcode.AnyAmountCents, Type: payment.Alipay}

	cases := []struct {
		name     string
		codes    map[payment.Type]map[int64]qrcode.QRCode
		settings map[string]string
		typ      payment.Type
		wantURL  string
		wantAuto bool
		wantCode ErrorCode
	}{
		{
			name:    "精确金额固定码命中",
			codes:   map[payment.Type]map[int64]qrcode.QRCode{payment.Alipay: {amount: fixedAlipay}},
			typ:     payment.Alipay,
			wantURL: fixedAlipay.PayURL,
		},
		{
			name: "固定码优先于通用码",
			codes: map[payment.Type]map[int64]qrcode.QRCode{payment.Alipay: {
				amount:                fixedAlipay,
				qrcode.AnyAmountCents: genericAlipay,
			}},
			settings: map[string]string{setting.AlipayPayURLKey: "https://pay.test/alipay"},
			typ:      payment.Alipay,
			wantURL:  fixedAlipay.PayURL,
		},
		{
			name:     "仅有通用码时命中通用码",
			codes:    map[payment.Type]map[int64]qrcode.QRCode{payment.Alipay: {qrcode.AnyAmountCents: genericAlipay}},
			settings: map[string]string{setting.AlipayPayURLKey: "https://pay.test/alipay"},
			typ:      payment.Alipay,
			wantURL:  genericAlipay.PayURL,
			wantAuto: true,
		},
		{
			name:     "通用码不跨支付类型",
			codes:    map[payment.Type]map[int64]qrcode.QRCode{payment.Alipay: {qrcode.AnyAmountCents: genericAlipay}},
			settings: map[string]string{setting.WechatPayURLKey: "wxp://fallback"},
			typ:      payment.Wechat,
			wantURL:  "wxp://fallback",
			wantAuto: true,
		},
		{
			name:     "码库为空时回退系统设置",
			settings: map[string]string{setting.AlipayPayURLKey: "https://pay.test/alipay"},
			typ:      payment.Alipay,
			wantURL:  "https://pay.test/alipay",
			wantAuto: true,
		},
		{
			name:     "全部缺失时返回配置错误",
			typ:      payment.Alipay,
			wantCode: CodeConfiguration,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := &OrderService{qrcodes: amountQRCodes{codes: tc.codes}}
			settings := tc.settings
			if settings == nil {
				settings = map[string]string{}
			}

			payURL, isAuto, err := service.resolvePayURL(context.Background(), tc.typ, amount, settings)
			if tc.wantCode != "" {
				code, ok := ErrorCodeOf(err)
				if !ok || code != tc.wantCode {
					t.Fatalf("期望错误码 %v，实际 err=%v", tc.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("选码应成功，实际 err=%v", err)
			}
			if payURL != tc.wantURL || isAuto != tc.wantAuto {
				t.Fatalf("期望 (%q, isAuto=%v)，实际 (%q, isAuto=%v)", tc.wantURL, tc.wantAuto, payURL, isAuto)
			}
		})
	}
}
