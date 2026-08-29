package tests

import (
	"context"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationApplication "traiderBot/internalDDD/simulation/application"
)

type orderBookReaderStub struct {
	orderBooks map[string]marketDomain.OrderBook
}

func (s *orderBookReaderStub) Get(
	symbol string,
) (marketDomain.OrderBook, bool) {
	orderBook, ok := s.orderBooks[symbol]
	return orderBook, ok
}

type feeProviderStub struct {
	fees map[string]simulationApplication.FeeConfig
}

func (s *feeProviderStub) Get(
	ctx context.Context,
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := s.fees[symbol]
	return fee, ok
}

type symbolProviderStub struct {
	symbols map[string]marketDomain.Symbol
}

func (s *symbolProviderStub) Get(
	symbol string,
) (marketDomain.Symbol, bool) {
	result, ok := s.symbols[symbol]
	return result, ok
}

func newOrderBook(
	symbol string,
	price string,
	quantity string,
) marketDomain.OrderBook {
	return marketDomain.OrderBook{
		Symbol: symbol,
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.RequireFromString(price),
				Qty:   decimal.RequireFromString(quantity),
			},
		},
		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.RequireFromString(price),
				Qty:   decimal.RequireFromString(quantity),
			},
		},
	}
}

func triangleRoute() marketDomain.Route {
	return marketDomain.NewRoute([]marketDomain.GraphEdge{
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		{
			From:      "BTC",
			To:        "ETH",
			Symbol:    "ETHBTC",
			Direction: marketDomain.Buy,
		},
		{
			From:      "ETH",
			To:        "USDT",
			Symbol:    "ETHUSDT",
			Direction: marketDomain.Sell,
		},
	})
}
