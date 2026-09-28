package domain

import (
	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
)

type RecoveryStep struct {
	FromAsset string
	ToAsset   string
	Symbol    string
	Direction marketDomain.TradeDirection

	// RequestedInput — сколько входного актива хотели исполнить.
	RequestedInput decimal.Decimal

	// ExecutedInput — сколько входного актива реально исполнимо
	// по текущему order book.
	ExecutedInput decimal.Decimal

	// GrossOutput — результат до комиссии.
	GrossOutput decimal.Decimal

	// NetOutput — результат после комиссии.
	NetOutput decimal.Decimal

	// Средняя цена исполнения.
	SimulatedPrice decimal.Decimal

	// Комиссия.
	SimulatedFee decimal.Decimal

	// Доля реально исполнимого объёма.
	FillRatio decimal.Decimal
}