package application

import (

	"context"
	"sync"

	balanceDomain "traiderBot/internalDDD/balance/domain"

	"github.com/shopspring/decimal"
)


type BalanceService struct {

	provider BalanceProvider

	mu sync.RWMutex

	state balanceDomain.State
}


func NewBalanceService(
	provider BalanceProvider,
) *BalanceService {

	return &BalanceService{
		provider: provider,
		state: balanceDomain.NewState(),
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

	defer s.mu.Unlock()

	s.state = state

	return nil
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

	return s.state.Reserve(
		asset,
		amount,
	)
}



func (s *BalanceService) Release(
	asset string,
	amount decimal.Decimal,
) error {

	s.mu.Lock()

	defer s.mu.Unlock()

	return s.state.Release(
		asset,
		amount,
	)
}