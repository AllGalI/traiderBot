package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	balanceDomain "traiderBot/internalDDD/balance/domain"
	binanceRest "traiderBot/internalDDD/exchange/infrastructure/binance/rest"

	"github.com/shopspring/decimal"
)

const accountPath = "/api/v3/account"

type BinanceBalanceProvider struct {
	client *binanceRest.Client
}

func NewBinanceBalanceProvider(
	client *binanceRest.Client,
) *BinanceBalanceProvider {

	return &BinanceBalanceProvider{
		client: client,
	}
}

type accountResponse struct {
	Balances []accountBalance `json:"balances"`
}

type accountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

func (p *BinanceBalanceProvider) GetBalances(
	ctx context.Context,
) (
	balanceDomain.State,
	error,
) {

	body, err := p.client.SignedRequest(
		ctx,
		"GET",
		accountPath,
		url.Values{},
	)

	if err != nil {
		return balanceDomain.State{}, fmt.Errorf(
			"request account: %w",
			err,
		)
	}

	var response accountResponse

	if err := json.Unmarshal(
		body,
		&response,
	); err != nil {
		return balanceDomain.State{}, fmt.Errorf(
			"decode account response: %w",
			err,
		)
	}

	state := balanceDomain.NewState()

	for _, balance := range response.Balances {

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

		total := free.Add(locked)
		if total.GreaterThan(decimal.Zero) {
			state.Set(
				balance.Asset,
				total,
				locked,
			)
		}

	}

	return state, nil
}