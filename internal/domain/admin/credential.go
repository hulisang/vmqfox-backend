package admin

import (
	"errors"
	"strings"
	"time"
)

const SingletonID uint8 = 1

var ErrInvalidCredential = errors.New("管理员凭据无效")

type Credential struct {
	ID           uint8
	Username     string
	PasswordHash string
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NormalizeUsername(value string) string {
	return strings.TrimSpace(value)
}

func (c Credential) Validate() error {
	username := NormalizeUsername(c.Username)
	if c.ID != SingletonID || username == "" || len([]byte(username)) > 128 || c.PasswordHash == "" {
		return ErrInvalidCredential
	}
	return nil
}
