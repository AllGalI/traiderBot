package tests

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	marketInfrastructure "traiderBot/internalDDD/market/infrastructure"
	simulationApplication "traiderBot/internalDDD/simulation/application"
)

type benchmarkFeeProvider struct {
	fees map[string]simulationApplication.FeeConfig
}

func (p *benchmarkFeeProvider) Get(
	ctx context.Context,
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := p.fees[symbol]
	return fee, ok
}

type benchmarkOrderBookReader struct {
	orderBooks map[string]marketDomain.OrderBook
}

func (r *benchmarkOrderBookReader) Get(
	symbol string,
) (marketDomain.OrderBook, bool) {
	orderBook, ok := r.orderBooks[symbol]
	return orderBook, ok
}

func benchmarkSymbol(
	name string,
	base string,
	quote string,
) marketDomain.Symbol {
	return marketDomain.Symbol{
		Name:       name,
		BaseAsset:  base,
		QuoteAsset: quote,
		Constraints: marketDomain.SymbolConstraints{
			MinQty:      decimal.RequireFromString("0.000001"),
			MaxQty:      decimal.RequireFromString("1000000"),
			StepSize:    decimal.RequireFromString("0.000001"),
			MinNotional: decimal.Zero,
			PriceStep:   decimal.RequireFromString("0.01"),
		},
	}
}

func benchmarkOrderBook(
	symbol string,
	price string,
	qty string,
) marketDomain.OrderBook {
	now := time.Now()

	return marketDomain.OrderBook{
		Symbol:         symbol,
		LastUpdateTime: now,

		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.RequireFromString(price),
				Qty:   decimal.RequireFromString(qty),
			},
		},

		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.RequireFromString(price),
				Qty:   decimal.RequireFromString(qty),
			},
		},
	}
}

func benchmarkStructure() *marketInfrastructure.InMemoryMarketStructureStore {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	store.Set(
		&marketDomain.MarketStructure{
			Symbols: []marketDomain.Symbol{
				benchmarkSymbol(
					"BTCUSDT",
					"BTC",
					"USDT",
				),
				benchmarkSymbol(
					"ETHBTC",
					"ETH",
					"BTC",
				),
				benchmarkSymbol(
					"ETHUSDT",
					"ETH",
					"USDT",
				),
			},
		},
	)

	return store
}

func benchmarkRoute() marketDomain.Route {
	return marketDomain.Route{
		Edges: []marketDomain.GraphEdge{
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
		},
	}
}

func newBenchmarkSimulationService() *simulationApplication.SimulationService {
	orderBooks := map[string]marketDomain.OrderBook{
		"BTCUSDT": benchmarkOrderBook(
			"BTCUSDT",
			"100000",
			"10",
		),

		"ETHBTC": benchmarkOrderBook(
			"ETHBTC",
			"0.05",
			"100",
		),

		"ETHUSDT": benchmarkOrderBook(
			"ETHUSDT",
			"5100",
			"100",
		),
	}

	fees := &benchmarkFeeProvider{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate:  decimal.Zero,
			},
			"ETHBTC": {
				Rate:  decimal.Zero,
			},
			"ETHUSDT": {
				Rate:  decimal.Zero,
			},
		},
	}

	structureStore := benchmarkStructure()

	symbols := simulationApplication.NewMarketStructureSymbolProvider(
		structureStore,
	)

	freshness := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Hour,
			"ETHBTC":  time.Hour,
			"ETHUSDT": time.Hour,
		},
	)

	return simulationApplication.NewSimulationService(
		&benchmarkOrderBookReader{
			orderBooks: orderBooks,
		},
		fees,
		symbols,
		freshness,
		decimal.RequireFromString("0.90"),
		decimal.Zero,
	)
}

func BenchmarkSimulationService(b *testing.B) {
	service := newBenchmarkSimulationService()

	route := benchmarkRoute()

	startAmount := decimal.NewFromInt(100)

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.Simulate(
			ctx,
			route,
			startAmount,
		)
	}
}