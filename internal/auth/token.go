package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hulisang/vmqfox-backend/internal/config"
)

var (
	ErrMissingToken = errors.New("缺少访问令牌")
	ErrInvalidToken = errors.New("访问令牌无效或已过期")
)

type Claims struct {
	jwt.RegisteredClaims
}

type Identity struct {
	Subject   string
	ExpiresAt time.Time
}

type TokenService struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenService(cfg config.TokenConfig) (*TokenService, error) {
	if len(cfg.Secret) < 32 || cfg.Issuer == "" || cfg.TTL <= 0 {
		return nil, errors.New("Token 配置无效")
	}
	return &TokenService{
		secret: []byte(cfg.Secret),
		issuer: cfg.Issuer,
		ttl:    cfg.TTL,
		now:    time.Now,
	}, nil
}

func (s *TokenService) Issue(subject string) (string, time.Time, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", time.Time{}, errors.New("Token subject 不能为空")
	}

	now := s.now()
	expiresAt := now.Add(s.ttl)
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	return signed, expiresAt, err
}

func (s *TokenService) ParseAuthorization(value string) (Identity, error) {
	raw := strings.TrimSpace(value)
	if len(raw) >= 7 && strings.EqualFold(raw[:7], "bearer ") {
		raw = strings.TrimSpace(raw[7:])
	}
	if raw == "" {
		return Identity{}, ErrMissingToken
	}
	return s.Parse(raw)
}

func (s *TokenService) Parse(raw string) (Identity, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil || !token.Valid || claims.Subject == "" || claims.ExpiresAt == nil {
		return Identity{}, ErrInvalidToken
	}
	return Identity{Subject: claims.Subject, ExpiresAt: claims.ExpiresAt.Time}, nil
}

type identityContextKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}
