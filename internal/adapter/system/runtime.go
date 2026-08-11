package system

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
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
