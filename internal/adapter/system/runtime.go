package system

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/order"
	"github.com/hulisang/vmqfox-backend/internal/port"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now() }

type OrderIDGenerator struct{}

func (OrderIDGenerator) NewOrderID(now time.Time) (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	suffix := binary.BigEndian.Uint64(raw[:])%90000 + 10000
	return now.Format("20060102150405") + fmt.Sprintf("%05d", suffix), nil
}

// PublicTokenGenerator 使用 32 字节系统随机数生成 URL 安全的公开订单令牌。
type PublicTokenGenerator struct{}

func (PublicTokenGenerator) NewPublicToken() (string, error) {
	var raw [order.PublicTokenLength / 2]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

var _ port.OrderIDGenerator = OrderIDGenerator{}
var _ port.PublicTokenGenerator = PublicTokenGenerator{}
