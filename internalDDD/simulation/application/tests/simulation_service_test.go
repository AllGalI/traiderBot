package tests

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	marketInfrastructure "traiderBot/internalDDD/market/infrastructure"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type integrationOrderBookReader struct {
	orderBooks map[string]marketDomain.OrderBook
}

func (r *integrationOrderBookReader) Get(
	symbol string,
) (marketDomain.OrderBook, bool) {
	orderBook, ok := r.orderBooks[symbol]
	return orderBook, ok
}

type integrationFeeProvider struct {
	fees map[string]simulationApplication.FeeConfig
}

func (p *integrationFeeProvider) Get(
	ctx context.Context,
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := p.fees[symbol]
	return fee, ok
}

func integrationSymbol(
	name string,
	baseAsset string,
	quoteAsset string,
) marketDomain.Symbol {
	return marketDomain.Symbol{
		Name:       name,
		BaseAsset:  baseAsset,
		QuoteAsset: quoteAsset,
		Constraints: marketDomain.SymbolConstraints{
			MinQty:      decimal.RequireFromString("0.000001"),
			MaxQty:      decimal.RequireFromString("1000000"),
			StepSize:    decimal.RequireFromString("0.000001"),
			MinNotional: decimal.RequireFromString("0.000001"),
			PriceStep:   decimal.RequireFromString("0.01"),
		},
	}
}

func integrationOrderBook(
	symbol string,
	bidPrice string,
	bidQty string,
	askPrice string,
	askQty string,
) marketDomain.OrderBook {
	return marketDomain.OrderBook{
		Symbol: symbol,
		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.RequireFromString(bidPrice),
				Qty:   decimal.RequireFromString(bidQty),
			},
		},
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.RequireFromString(askPrice),
				Qty:   decimal.RequireFromString(askQty),
			},
		},
	}
}

func integrationTriangleRoute() marketDomain.Route {
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

func TestSimulationServiceUsesMarketStructureSymbolProvider(t *testing.T) {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	structure := &marketDomain.MarketStructure{
		Symbols: []marketDomain.Symbol{
			integrationSymbol("BTCUSDT", "BTC", "USDT"),
			integrationSymbol("ETHBTC", "ETH", "BTC"),
			integrationSymbol("ETHUSDT", "ETH", "USDT"),
		},
	}

	store.Set(structure)

	symbolProvider := simulationApplication.NewMarketStructureSymbolProvider(store)

	orderBooks := &integrationOrderBookReader{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": integrationOrderBook(
				"BTCUSDT",
				"100000",
				"1",
				"100000",
				"1",
			),
			"ETHBTC": integrationOrderBook(
				"ETHBTC",
				"0.05",
				"10",
				"0.05",
				"10",
			),
			"ETHUSDT": integrationOrderBook(
				"ETHUSDT",
				"5100",
				"10",
				"5100",
				"10",
			),
		},
	}

	fees := &integrationFeeProvider{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate: decimal.Zero,
			},
			"ETHBTC": {
				Rate: decimal.Zero,
			},
			"ETHUSDT": {
				Rate: decimal.Zero,
			},
		},
	}

	service := simulationApplication.NewSimulationService(
		orderBooks,
		fees,
		symbolProvider,
		alwaysFreshPolicy{},
		decimal.RequireFromString("0.3"),
		decimal.Zero,
	)

	result := service.Simulate(
		context.Background(),
		integrationTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK, got %v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity")
	}

	if len(result.Opportunity.Steps) != 3 {
		t.Fatalf(
			"expected 3 simulation steps, got %d",
			len(result.Opportunity.Steps),
		)
	}

	if result.Opportunity.StartAsset != "USDT" {
		t.Fatalf(
			"expected start asset USDT, got %q",
			result.Opportunity.StartAsset,
		)
	}

	if !result.Opportunity.RequestedStartAmount.Equal(
		decimal.NewFromInt(100),
	) {
		t.Fatalf(
			"expected requested start amount 100, got %s",
			result.Opportunity.RequestedStartAmount,
		)
	}

	for _, symbol := range []string{
		"BTCUSDT",
		"ETHBTC",
		"ETHUSDT",
	} {
		if _, ok := symbolProvider.Get(symbol); !ok {
			t.Fatalf(
				"expected symbol %s to be available through market structure",
				symbol,
			)
		}
	}

}

func TestSimulationServiceIntegrationOrderBookMissing(t *testing.T) {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	store.Set(&marketDomain.MarketStructure{
		Symbols: []marketDomain.Symbol{
			integrationSymbol("BTCUSDT", "BTC", "USDT"),
			integrationSymbol("ETHBTC", "ETH", "BTC"),
			integrationSymbol("ETHUSDT", "ETH", "USDT"),
		},
	})

	symbolProvider := simulationApplication.NewMarketStructureSymbolProvider(store)

	orderBooks := &integrationOrderBookReader{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": integrationOrderBook(
				"BTCUSDT",
				"100000",
				"1",
				"100000",
				"1",
			),
			"ETHBTC": integrationOrderBook(
				"ETHBTC",
				"0.05",
				"10",
				"0.05",
				"10",
			),
			// ETHUSDT намеренно отсутствует.
		},
	}

	fees := &integrationFeeProvider{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate: decimal.Zero,
			},
			"ETHBTC": {
				Rate: decimal.Zero,
			},
			"ETHUSDT": {
				Rate: decimal.Zero,
			},
		},
	}

	service := simulationApplication.NewSimulationService(
		orderBooks,
		fees,
		symbolProvider,
		alwaysFreshPolicy{},
		decimal.RequireFromString("0.3"),
		decimal.Zero,
	)

	result := service.Simulate(
		context.Background(),
		integrationTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOrderBookNotFound {
		t.Fatalf(
			"expected SimulationOrderBookNotFound, got %v",
			result.Reason,
		)
	}

	if result.Opportunity != nil {
		t.Fatal("expected no opportunity")
	}

}

func TestSimulationServiceIntegrationSymbolMissing(t *testing.T) {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	store.Set(&marketDomain.MarketStructure{
		Symbols: []marketDomain.Symbol{
			integrationSymbol("BTCUSDT", "BTC", "USDT"),
			integrationSymbol("ETHBTC", "ETH", "BTC"),
			// ETHUSDT намеренно отсутствует.
		},
	})

	symbolProvider := simulationApplication.NewMarketStructureSymbolProvider(store)

	orderBooks := &integrationOrderBookReader{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": integrationOrderBook(
				"BTCUSDT",
				"100000",
				"1",
				"100000",
				"1",
			),
			"ETHBTC": integrationOrderBook(
				"ETHBTC",
				"0.05",
				"10",
				"0.05",
				"10",
			),
			"ETHUSDT": integrationOrderBook(
				"ETHUSDT",
				"5100",
				"10",
				"5100",
				"10",
			),
		},
	}

	fees := &integrationFeeProvider{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate: decimal.Zero,
			},
			"ETHBTC": {
				Rate: decimal.Zero,
			},
			"ETHUSDT": {
				Rate: decimal.Zero,
			},
		},
	}

	service := simulationApplication.NewSimulationService(
		orderBooks,
		fees,
		symbolProvider,		
		alwaysFreshPolicy{},
		decimal.RequireFromString("0.3"),
		decimal.Zero,
	)

	result := service.Simulate(
		context.Background(),
		integrationTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationFiltersNotFound {
		t.Fatalf(
			"expected SimulationFiltersNotFound, got %v",
			result.Reason,
		)
	}

	if result.Opportunity != nil {
		t.Fatal("expected no opportunity")
	}

}
