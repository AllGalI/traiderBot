package infrastructure

import (
	"github.com/shopspring/decimal"

	simulationApplication "traiderBot/internalDDD/simulation/application"
)

type StaticFeeProvider struct {
	rate decimal.Decimal
}

func NewStaticFeeProvider(
	rate decimal.Decimal,
) *StaticFeeProvider {
	return &StaticFeeProvider{
		rate: rate,
	}
}

func (p *StaticFeeProvider) Get(
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	return simulationApplication.FeeConfig{
		Rate:  p.rate,
	}, false
}