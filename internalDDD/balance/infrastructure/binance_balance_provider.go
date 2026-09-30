package infrastructure

import (
	"context"
	"fmt"

	balanceDomain "traiderBot/internalDDD/balance/domain"

	"github.com/shopspring/decimal"
)

type BinanceBalanceProvider struct {
	client BinanceAccountClient
}

func NewBinanceBalanceProvider(
	client BinanceAccountClient,
) *BinanceBalanceProvider {

	return &BinanceBalanceProvider{
		client: client,
	}
}

func (p *BinanceBalanceProvider) GetBalances(
	ctx context.Context,
) (
	balanceDomain.State,
	error,
) {

	account, err := p.client.GetAccount(
		ctx,
	)

	if err != nil {
		return balanceDomain.State{}, err
	}

	state := balanceDomain.NewState()

	for _, balance := range account.Balances {

		free, err := decimal.NewFromString(
			balance.Free,
		)

		if err != nil {
			return balanceDomain.State{}, fmt.Errorf(
				"parse free balance %s: %w",
				balance.Asset,
				err,
			)
		}

		locked, err := decimal.NewFromString(
			balance.Locked,
		)

		if err != nil {
			return balanceDomain.State{}, fmt.Errorf(
				"parse locked balance %s: %w",
				balance.Asset,
				err,
			)
		}

		total := free.Add(
			locked,
		)

		state.Set(
			balance.Asset,
			total,
		)
	}

	return state, nil
}
