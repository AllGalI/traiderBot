package binance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"

	binanceRest "traiderBot/internalDDD/exchange/infrastructure/binance/rest"
	simulationApplication "traiderBot/internalDDD/simulation/application"
)

const accountCommissionPath = "/api/v3/account/commission"

type FeeProvider struct {
	client *binanceRest.Client
}

func NewFeeProvider(client *binanceRest.Client) *FeeProvider {
	return &FeeProvider{
		client: client,
	}
}

type commissionResponse struct {
	Symbol string `json:"symbol"`

	StandardCommission struct {
		Maker string `json:"maker"`
		Taker string `json:"taker"`
	} `json:"standardCommission"`
}

func (p *FeeProvider) Get(
	ctx context.Context,
	symbol string,
) (
	simulationApplication.FeeConfig,
	error,
) {

	body, err := p.client.GetAccountCommission(
		ctx,
		symbol,
	)

	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf(
				"request commission: %w",
			)
	}

	var dto commissionResponse

	if err := json.Unmarshal(body, &dto); err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf(
				"decode commission response: %w",
				err,
			)
	}

	taker, err := decimal.NewFromString(
		dto.StandardCommission.Taker,
	)

	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf(
				"parse taker commission: %w",
				err,
			)
	}

	return simulationApplication.FeeConfig{
		Rate: taker,
	}, nil
}
