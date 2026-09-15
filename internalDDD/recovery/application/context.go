package application

import (
	marketDomain "traiderBot/internalDDD/market/domain"
)


type MarketContext struct {

	Symbols map[string]marketDomain.Symbol

	OrderBooks map[string]marketDomain.OrderBook

	Fees map[string]marketDomain.FeeRate
}