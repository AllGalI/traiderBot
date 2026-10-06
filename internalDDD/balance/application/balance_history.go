package application

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type BalanceHistoryPoint struct {
	Time   time.Time       `json:"time"`
	USDT   decimal.Decimal `json:"usdt"`
	Delta  decimal.Decimal `json:"delta"`
}

type BalanceHistory struct {
	mu sync.RWMutex

	points []BalanceHistoryPoint

	maxPoints int

	lastUSDT decimal.Decimal
	hasLast  bool
}

func NewBalanceHistory(
	maxPoints int,
) *BalanceHistory {
	if maxPoints <= 0 {
		maxPoints = 2880
	}

	return &BalanceHistory{
		points:    make([]BalanceHistoryPoint, 0, maxPoints),
		maxPoints: maxPoints,
	}
}

func (h *BalanceHistory) Add(
	now time.Time,
	usdt decimal.Decimal,
) BalanceHistoryPoint {
	h.mu.Lock()
	defer h.mu.Unlock()

	delta := decimal.Zero

	if h.hasLast {
		delta = usdt.Sub(h.lastUSDT)
	}

	point := BalanceHistoryPoint{
		Time:  now,
		USDT:  usdt,
		Delta: delta,
	}

	h.points = append(
		h.points,
		point,
	)

	if len(h.points) > h.maxPoints {
		excess := len(h.points) - h.maxPoints

		copy(
			h.points,
			h.points[excess:],
		)

		h.points = h.points[:h.maxPoints]
	}

	h.lastUSDT = usdt
	h.hasLast = true

	return point
}

func (h *BalanceHistory) Current() (
	decimal.Decimal,
	bool,
) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.hasLast {
		return decimal.Zero, false
	}

	return h.lastUSDT, true
}

func (h *BalanceHistory) Snapshot() []BalanceHistoryPoint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(
		[]BalanceHistoryPoint,
		len(h.points),
	)

	copy(result, h.points)

	return result
}