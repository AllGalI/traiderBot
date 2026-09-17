package domain


import "github.com/shopspring/decimal"


type FeeRate struct {

	Maker decimal.Decimal

	Taker decimal.Decimal
}