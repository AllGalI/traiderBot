package infrastructure

import (
	"github.com/shopspring/decimal"

	simulationApplication "traiderBot/internalDDD/simulation/application"
)

type FeeConfigProvider struct {
	fees map[string]simulationApplication.FeeConfig
}

func NewFeeConfigProvider(
	fees map[string]simulationApplication.FeeConfig,
) *FeeConfigProvider {
	copied := make(map[string]simulationApplication.FeeConfig, len(fees))

	for symbol, fee := range fees {
		copied[symbol] = fee
	}

	return &FeeConfigProvider{
		fees: copied,
	}
}

func (p *FeeConfigProvider) Get(
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := p.fees[symbol]
	return fee, ok
}

func DefaultFeeConfigProvider(
	symbols []string,
	rate decimal.Decimal,
) *FeeConfigProvider {
	fees := make(map[string]simulationApplication.FeeConfig, len(symbols))

	for _, symbol := range symbols {
		fees[symbol] = simulationApplication.FeeConfig{
			Rate:  rate,
		}
	}

	return NewFeeConfigProvider(fees)
}
