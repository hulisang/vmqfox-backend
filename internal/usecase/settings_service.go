package usecase

import (
	"context"
	"errors"
	"strconv"

	"github.com/hulisang/vmqfox-backend/internal/domain/admin"
	"github.com/hulisang/vmqfox-backend/internal/domain/setting"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

type SettingsView struct {
	Username string
	Values   map[string]string
}

type UpdateSettingsInput struct {
	Username     string
	Password     string
	NotifyURL    string
	ReturnURL    string
	MerchantKey  string
	CloseMinutes string
	PriceAdjust  string
	WechatPayURL string
	AlipayPayURL string
}

type SettingsServiceDeps struct {
	Transactions port.TransactionManager
	Credentials  port.AdminCredentialRepository
	Passwords    port.PasswordHasher
	Settings     port.SettingRepository
	Clock        port.Clock
}

type SettingsService struct {
	transactions port.TransactionManager
	credentials  port.AdminCredentialRepository
	passwords    port.PasswordHasher
	settings     port.SettingRepository
	clock        port.Clock
}

func NewSettingsService(deps SettingsServiceDeps) (*SettingsService, error) {
	if deps.Transactions == nil || deps.Credentials == nil || deps.Passwords == nil || deps.Settings == nil || deps.Clock == nil {
		return nil, errors.New("设置用例依赖不完整")
	}
	return &SettingsService{
		transactions: deps.Transactions,
		credentials:  deps.Credentials,
		passwords:    deps.Passwords,
		settings:     deps.Settings,
		clock:        deps.Clock,
	}, nil
}

func (s *SettingsService) Get(ctx context.Context) (SettingsView, error) {
	credential, err := s.credentials.Get(ctx)
	if errors.Is(err, port.ErrNotFound) {
		return SettingsView{}, fail(CodeConfiguration, "管理员凭据未初始化")
	}
	if err != nil {
		return SettingsView{}, wrap(CodeDependency, "读取管理员凭据失败", err)
	}
	values, err := s.settings.GetMany(ctx, setting.AllKeys())
	if err != nil {
		return SettingsView{}, wrap(CodeDependency, "读取系统设置失败", err)
	}
	return SettingsView{Username: credential.Username, Values: values}, nil
}

func (s *SettingsService) Update(ctx context.Context, input UpdateSettingsInput) error {
	username := admin.NormalizeUsername(input.Username)
	if username == "" || len([]byte(username)) > 128 {
		return fail(CodeInvalidArgument, "管理员账号无效")
	}
	if input.MerchantKey == "" {
		return fail(CodeInvalidArgument, "通讯密钥不能为空")
	}
	closeMinutes, err := strconv.Atoi(input.CloseMinutes)
	if err != nil || closeMinutes <= 0 {
		return fail(CodeInvalidArgument, "订单有效期无效")
	}
	if input.PriceAdjust != "1" && input.PriceAdjust != "2" {
		return fail(CodeInvalidArgument, "金额区分方式无效")
	}

	passwordHash := ""
	if input.Password != "" {
		passwordHash, err = s.passwords.Hash(input.Password)
		if err != nil {
			return wrap(CodeInvalidArgument, "管理员密码不符合要求", err)
		}
	}

	now := s.clock.Now()
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		credential, err := s.credentials.GetForUpdate(txCtx)
		if errors.Is(err, port.ErrNotFound) {
			return fail(CodeConfiguration, "管理员凭据未初始化")
		}
		if err != nil {
			return wrap(CodeDependency, "读取管理员凭据失败", err)
		}
		credential.Username = username
		credential.UpdatedAt = now
		if passwordHash != "" {
			credential.PasswordHash = passwordHash
		}
		if err := s.credentials.Update(txCtx, credential); err != nil {
			return wrap(CodeDependency, "更新管理员凭据失败", err)
		}

		if err := s.settings.SetMany(txCtx, map[string]string{
			setting.NotifyURLKey:    input.NotifyURL,
			setting.ReturnURLKey:    input.ReturnURL,
			setting.MerchantKey:     input.MerchantKey,
			setting.CloseMinutesKey: strconv.Itoa(closeMinutes),
			setting.PriceAdjustKey:  input.PriceAdjust,
			setting.WechatPayURLKey: input.WechatPayURL,
			setting.AlipayPayURLKey: input.AlipayPayURL,
		}); err != nil {
			return wrap(CodeDependency, "更新系统设置失败", err)
		}
		return nil
	})
}

func (s *SettingsService) UpdateMonitorState(ctx context.Context, state string) error {
	if state != "1" && state != "0" && state != "-1" {
		return fail(CodeInvalidArgument, "监控状态无效")
	}
	return s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.settings.Set(txCtx, setting.MonitorStateKey, state); err != nil {
			return wrap(CodeDependency, "更新系统设置失败", err)
		}
		return nil
	})
}
