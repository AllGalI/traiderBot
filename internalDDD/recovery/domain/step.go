package domain

import (
	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
)


type RecoveryStep struct {

	FromAsset string

	ToAsset string


	Symbol string

	Direction marketDomain.TradeDirection


	InputAmount decimal.Decimal


	// результат Simulation
	SimulatedOutput decimal.Decimal


	// ожидаемая цена
	SimulatedPrice decimal.Decimal


	// комиссия
	SimulatedFee decimal.Decimal
}