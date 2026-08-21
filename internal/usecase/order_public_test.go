package usecase

import (
	"context"
	"testing"

	"github.com/hulisang/vmqfox-backend/internal/port"
)

type publicOrderSettings struct{}

func (publicOrderSettings) Get(context.Context, string) (string, error) {
	return "5", nil
}

func (publicOrderSettings) GetMany(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (publicOrderSettings) GetManyForUpdate(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (publicOrderSettings) Set(context.Context, string, string) error {
	return nil
}

func (publicOrderSettings) SetMany(context.Context, map[string]string) error {
	return nil
}

var _ port.SettingRepository = publicOrderSettings{}

// TestPublicEndpointsRejectLegacyOrderID 锁定公开端点只接受 publicToken，绝不回退到可枚举订单号。
func TestPublicEndpointsRejectLegacyOrderID(t *testing.T) {
	service := &OrderService{settings: publicOrderSettings{}}
	legacyOrderID := "2026082115304512345"
	ctx := context.Background()

	cases := map[string]func() error{
		"get": func() error {
			_, err := service.Get(ctx, legacyOrderID)
			return err
		},
		"check": func() error {
			_, err := service.Check(ctx, legacyOrderID)
			return err
		},
		"return-url": func() error {
			_, err := service.ReturnURL(ctx, legacyOrderID)
			return err
		},
	}

	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			err := request()
			code, ok := ErrorCodeOf(err)
			if !ok || code != CodeNotFound {
				t.Fatalf("旧订单号应得到统一的不存在错误，实际 err=%v code=%v ok=%v", err, code, ok)
			}
		})
	}
}
