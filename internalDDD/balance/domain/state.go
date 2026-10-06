package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type State struct {
	balances map[string]*AssetBalance
}

func NewState() State {
	return State{
		balances: make(map[string]*AssetBalance),
	}
}

func (s State) Get(
	asset string,
) (*AssetBalance, bool) {
	balance, ok := s.balances[asset]

	return balance, ok
}

func (s *State) Set(
	asset string,
	total decimal.Decimal,
	exchangeLocked decimal.Decimal,
) {
	s.balances[asset] = ptr(
		NewAssetBalance(
			asset,
			total,
			exchangeLocked,
		),
	)
}

func (s *State) Update(
	asset string,
	total decimal.Decimal,
	exchangeLocked decimal.Decimal,
) {
	balance, ok := s.balances[asset]

	if !ok {
		s.Set(
			asset,
			total,
			exchangeLocked,
		)

		return
	}

	balance.Total = total
	balance.ExchangeLocked = exchangeLocked
}

func (s *State) Merge(
	incoming State,
) {
	for asset, incomingBalance := range incoming.balances {
		s.Update(
			asset,
			incomingBalance.Total,
			incomingBalance.ExchangeLocked,
		)
	}
}

func (s State) Available(
	asset string,
) decimal.Decimal {
	balance, ok := s.Get(asset)

	if !ok {
		return decimal.Zero
	}

	return balance.Available()
}

func (s State) Balances() []AssetBalance {
	result := make(
		[]AssetBalance,
		0,
		len(s.balances),
	)

	for _, balance := range s.balances {
		result = append(
			result,
			*balance,
		)
	}

	return result
}

func (s *State) Reserve(
	asset string,
	amount decimal.Decimal,
) error {
	balance, ok := s.Get(asset)

	if !ok {
		return fmt.Errorf(
			"asset %s not found",
			asset,
		)
	}

	return balance.Reserve(amount)
}

func (s *State) Release(
	asset string,
	amount decimal.Decimal,
) error {
	balance, ok := s.Get(asset)

	if !ok {
		return fmt.Errorf(
			"asset %s not found",
			asset,
		)
	}

	return balance.Release(amount)
}

func ptr[T any](v T) *T {
	return &v
}
