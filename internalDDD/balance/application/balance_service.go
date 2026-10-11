package application

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	balanceDomain "traiderBot/internalDDD/balance/domain"

	"github.com/shopspring/decimal"
)

type BalanceSnapshot struct {
	Asset          string
	Total          decimal.Decimal
	ExchangeLocked decimal.Decimal
	Reserved       decimal.Decimal
	Available      decimal.Decimal
}

type BalanceService struct {
	provider BalanceProvider

	mu    sync.RWMutex
	state balanceDomain.State

	history *BalanceHistory
}

func NewBalanceService(
	provider BalanceProvider,
) *BalanceService {
	return &BalanceService{
		provider: provider,
		state:    balanceDomain.NewState(),
		history: NewBalanceHistory(
			24 * 60 * 2,
		),
	}
}

func (s *BalanceService) Refresh(
	ctx context.Context,
) error {
	state, err := s.provider.GetBalances(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()

	s.state.Merge(state)

	usdt := s.state.Available("USDT")

	s.mu.Unlock()

	point := s.history.Add(
		time.Now(),
		usdt,
	)

	if point.Delta.IsZero() {
		return nil
	}

	log.Printf(
		"[BALANCE] USDT=%s Δ=%s",
		point.USDT.String(),
		point.Delta.String(),
	)

	return nil
}

func (s *BalanceService) StartAutoRefresh(
	ctx context.Context,
	interval time.Duration,
) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	go func() {
		if err := s.Refresh(ctx); err != nil {
			log.Printf(
				"[BALANCE] initial refresh failed: %v",
				err,
			)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := s.Refresh(ctx); err != nil {
					log.Printf(
						"[BALANCE] refresh failed: %v",
						err,
					)
				}
			}
		}
	}()
}

func (s *BalanceService) Available(
	asset string,
) decimal.Decimal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state.Available(asset)
}

func (s *BalanceService) Reserve(
	asset string,
	amount decimal.Decimal,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.Reserve(asset, amount)
}

func (s *BalanceService) Release(
	asset string,
	amount decimal.Decimal,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.Release(asset, amount)
}

func (s *BalanceService) Snapshot() []BalanceSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	balances := make(
		[]BalanceSnapshot,
		0,
	)

	for _, balance := range s.state.Balances() {
		balances = append(
			balances,
			BalanceSnapshot{
				Asset:          balance.Asset,
				Total:          balance.Total,
				ExchangeLocked: balance.ExchangeLocked,
				Reserved:       balance.Reserved,
				Available:      balance.Available(),
			},
		)
	}

	sort.Slice(
		balances,
		func(i, j int) bool {
			return balances[i].Asset < balances[j].Asset
		},
	)

	return balances
}

func (s *BalanceService) USDT() decimal.Decimal {
	return s.Available("USDT")
}

func (s *BalanceService) USDTString() string {
	return s.USDT().String()
}

func (s *BalanceService) USDTHistory() []BalanceHistoryPoint {
	return s.history.Snapshot()
}

func (s *BalanceService) USDTHistorySince(
	since time.Time,
) []BalanceHistoryPoint {
	points := s.history.Snapshot()

	result := make(
		[]BalanceHistoryPoint,
		0,
		len(points),
	)

	for _, point := range points {
		if point.Time.Before(since) {
			continue
		}

		result = append(
			result,
			point,
		)
	}

	return result
}

func (s *BalanceService) String() string {
	return fmt.Sprintf(
		"USDT=%s",
		s.USDT().String(),
	)
}
