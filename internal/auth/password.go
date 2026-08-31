package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordBytes = 8
	maxPasswordBytes = 72
)

var ErrInvalidPasswordLength = errors.New("管理员密码长度必须为 8 到 72 个字节")

type BcryptPasswordHasher struct {
	cost int
}

func NewBcryptPasswordHasher() *BcryptPasswordHasher {
	return &BcryptPasswordHasher{cost: bcrypt.DefaultCost}
}

func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	if len([]byte(password)) < minPasswordBytes || len([]byte(password)) > maxPasswordBytes {
		return "", ErrInvalidPasswordLength
	}
	encoded, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (h *BcryptPasswordHasher) Verify(passwordHash, password string) (bool, error) {
	if passwordHash == "" || password == "" || len([]byte(password)) > maxPasswordBytes {
		return false, nil
	}
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, err
}
