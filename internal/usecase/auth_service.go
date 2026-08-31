package usecase

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/admin"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

type LoginInput struct {
	Username string
	Password string
}

type LoginResult struct {
	AccessToken string
	Username    string
	ExpiresAt   time.Time
}

type AuthServiceDeps struct {
	Credentials port.AdminCredentialRepository
	Passwords   port.PasswordHasher
	Tokens      port.TokenIssuer
}

type AuthService struct {
	credentials port.AdminCredentialRepository
	passwords   port.PasswordHasher
	tokens      port.TokenIssuer
}

func NewAuthService(deps AuthServiceDeps) (*AuthService, error) {
	if deps.Credentials == nil || deps.Passwords == nil || deps.Tokens == nil {
		return nil, errors.New("认证用例依赖不完整")
	}
	return &AuthService{
		credentials: deps.Credentials,
		passwords:   deps.Passwords,
		tokens:      deps.Tokens,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	username := admin.NormalizeUsername(input.Username)
	if username == "" || input.Password == "" {
		return LoginResult{}, fail(CodeInvalidArgument, "用户名或密码不能为空")
	}

	credential, err := s.credentials.Get(ctx)
	if errors.Is(err, port.ErrNotFound) {
		return LoginResult{}, fail(CodeConfiguration, "管理员凭据未初始化")
	}
	if err != nil {
		return LoginResult{}, wrap(CodeDependency, "读取管理员凭据失败", err)
	}

	passwordMatches, err := s.passwords.Verify(credential.PasswordHash, input.Password)
	if err != nil {
		return LoginResult{}, wrap(CodeDependency, "校验管理员密码失败", err)
	}
	if !credential.Enabled || !constantTimeStringEqual(username, credential.Username) || !passwordMatches {
		return LoginResult{}, fail(CodeInvalidCredentials, "用户名或密码错误")
	}

	token, expiresAt, err := s.tokens.Issue(credential.Username)
	if err != nil {
		return LoginResult{}, wrap(CodeDependency, "签发访问令牌失败", err)
	}
	return LoginResult{AccessToken: token, Username: credential.Username, ExpiresAt: expiresAt}, nil
}

func constantTimeStringEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
