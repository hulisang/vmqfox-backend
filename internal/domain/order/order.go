package order

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hulisang/vmqfox-backend/internal/domain/payment"
)

type Status int8

const (
	StatusClosed       Status = -1
	StatusPending      Status = 0
	StatusPaid         Status = 1
	StatusNotifyFailed Status = 2
)

func (s Status) Valid() bool {
	switch s {
	case StatusClosed, StatusPending, StatusPaid, StatusNotifyFailed:
		return true
	default:
		return false
	}
}

func CanTransition(from, to Status) bool {
	if !from.Valid() || !to.Valid() {
		return false
	}
	if from == to {
		return true
	}
	switch from {
	case StatusPending:
		return to == StatusPaid || to == StatusClosed
	case StatusPaid:
		return to == StatusNotifyFailed
	case StatusNotifyFailed:
		return to == StatusPaid
	default:
		return false
	}
}

func CanReissue(status Status) bool {
	return status == StatusPending || status == StatusClosed || status == StatusNotifyFailed
}

var ErrInvalidAmount = errors.New("金额格式无效")

func ParseAmountCents(value string) (int64, error) {
	text := strings.TrimSpace(value)
	if text == "" || strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		return 0, ErrInvalidAmount
	}

	parts := strings.Split(text, ".")
	if len(parts) > 2 {
		return 0, ErrInvalidAmount
	}
	if parts[0] == "" {
		parts[0] = "0"
	}
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	if len(parts[1]) > 2 {
		return 0, ErrInvalidAmount
	}
	if len(parts[1]) == 1 {
		parts[1] += "0"
	}

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	fraction := int64(0)
	if parts[1] != "" {
		fraction, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, ErrInvalidAmount
		}
	}
	const maxInt64 = int64(^uint64(0) >> 1)
	if whole > (maxInt64-fraction)/100 {
		return 0, ErrInvalidAmount
	}
	return whole*100 + fraction, nil
}

func FormatCents(cents int64) string {
	if cents < 0 {
		absolute := -cents
		return fmt.Sprintf("-%d.%02d", absolute/100, absolute%100)
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

type PriceAdjustment int8

const (
	NoAdjustment PriceAdjustment = 0
	Increase     PriceAdjustment = 1
	Decrease     PriceAdjustment = 2
)

func CandidateAmounts(base int64, adjustment PriceAdjustment, attempts int) []int64 {
	if base <= 0 || attempts <= 0 {
		return nil
	}
	if adjustment == NoAdjustment {
		return []int64{base}
	}
	if adjustment != Increase && adjustment != Decrease {
		return nil
	}

	result := make([]int64, 0, attempts)
	for offset := int64(0); len(result) < attempts; offset++ {
		candidate := base + offset
		if adjustment == Decrease {
			candidate = base - offset
		}
		if candidate <= 0 {
			break
		}
		result = append(result, candidate)
	}
	return result
}

func IsExpired(status Status, createdAt time.Time, timeout time.Duration, now time.Time) bool {
	return status == StatusPending && timeout > 0 && now.After(createdAt.Add(timeout))
}

type Order struct {
	ID               int64
	OrderID          string
	PayID            string
	Type             payment.Type
	PriceCents       int64
	ReallyPriceCents int64
	PriceText        string
	ReallyPriceText  string
	State            Status
	Param            string
	PayURL           string
	IsAuto           bool
	NotifyURL        string
	ReturnURL        string
	CreatedAt        time.Time
	PaidAt           time.Time
	ClosedAt         time.Time
}
