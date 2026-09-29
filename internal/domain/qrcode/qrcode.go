package qrcode

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
)

type State int8

const (
	StateEnabled  State = 0
	StateDisabled State = 1
)

// AnyAmountCents 是通用任意金额收款码的金额标记，精确金额未命中时作为码库内的兜底。
const AnyAmountCents int64 = 0

func (s State) Valid() bool {
	return s == StateEnabled || s == StateDisabled
}

func CacheKey(content string) string {
	sum := md5.Sum([]byte(content))
	return hex.EncodeToString(sum[:])
}

type QRCode struct {
	ID         int64
	PayURL     string
	PriceCents int64
	Type       payment.Type
	State      State
}
