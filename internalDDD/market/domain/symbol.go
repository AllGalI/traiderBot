package domain

import "github.com/shopspring/decimal"

type Symbol struct {
	Name       string
	BaseAsset  string
	QuoteAsset string

	Constraints SymbolConstraints
}

type SymbolConstraints struct {
	MinQty      decimal.Decimal
	MaxQty      decimal.Decimal
	StepSize    decimal.Decimal
	MinNotional decimal.Decimal
	PriceStep   decimal.Decimal
}
