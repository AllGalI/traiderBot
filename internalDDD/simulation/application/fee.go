package application

import "github.com/shopspring/decimal"

type FeeConfig struct {
	Rate  decimal.Decimal
	Asset string
}

type FeeProvider interface {
	Get(symbol string) (FeeConfig, bool)
}
