package integration

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	marketInfrastructure "traiderBot/internalDDD/market/infrastructure"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type freshnessOrderBookReader struct {
	orderBooks map[string]marketDomain.OrderBook
}

func (r *freshnessOrderBookReader) Get(
	symbol string,
) (marketDomain.OrderBook, bool) {
	orderBook, ok := r.orderBooks[symbol]
	return orderBook, ok
}

type freshnessFeeProvider struct {
	fees map[string]simulationApplication.FeeConfig
}

func (p *freshnessFeeProvider) Get(
	_ context.Context,
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := p.fees[symbol]
	return fee, ok
}

func freshnessSymbol(
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
			MinNotional: decimal.Zero,
			PriceStep:   decimal.RequireFromString("0.01"),
		},
	}
}

func freshnessOrderBook(
	symbol string,
	updatedAt time.Time,
	bidPrice string,
	bidQty string,
	askPrice string,
	askQty string,
) marketDomain.OrderBook {
	return marketDomain.OrderBook{
		Symbol:         symbol,
		LastUpdateTime: updatedAt,
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

func freshnessTriangleRoute() marketDomain.Route {
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

func freshnessMarketStructureStore() *marketInfrastructure.InMemoryMarketStructureStore {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	store.Set(&marketDomain.MarketStructure{
		Symbols: []marketDomain.Symbol{
			freshnessSymbol("BTCUSDT", "BTC", "USDT"),
			freshnessSymbol("ETHBTC", "ETH", "BTC"),
			freshnessSymbol("ETHUSDT", "ETH", "USDT"),
		},
	})

	return store
}

func freshnessFees() *freshnessFeeProvider {
	return &freshnessFeeProvider{
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
}

func newFreshnessSimulationService(
	orderBooks map[string]marketDomain.OrderBook,
	ttls map[string]time.Duration,
) *simulationApplication.SimulationService {
	store := freshnessMarketStructureStore()

	symbolProvider := simulationApplication.NewMarketStructureSymbolProvider(
		store,
	)

	freshness := simulationApplication.NewFixedTTLFreshnessPolicy(ttls)

	return simulationApplication.NewSimulationService(
		&freshnessOrderBookReader{
			orderBooks: orderBooks,
		},
		freshnessFees(),
		symbolProvider,
		freshness,
		decimal.RequireFromString("0.3"),
		decimal.Zero,
	)
}

func TestSimulationServiceRejectsStaleOrderBook(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	orderBooks := map[string]marketDomain.OrderBook{
		"BTCUSDT": freshnessOrderBook(
			"BTCUSDT",
			now.Add(-100*time.Millisecond),
			"100000",
			"1",
			"100000",
			"1",
		),
		"ETHBTC": freshnessOrderBook(
			"ETHBTC",
			now.Add(-100*time.Millisecond),
			"0.05",
			"10",
			"0.05",
			"10",
		),
		"ETHUSDT": freshnessOrderBook(
			"ETHUSDT",
			now.Add(-2*time.Second),
			"5100",
			"10",
			"5100",
			"10",
		),
	}

	service := newFreshnessSimulationService(
		orderBooks,
		map[string]time.Duration{
			"BTCUSDT": time.Second,
			"ETHBTC":  time.Second,
			"ETHUSDT": time.Second,
		},
	)

	result := service.Simulate(
		ctx,
		freshnessTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOrderBookStale {
		t.Fatalf(
			"expected SimulationOrderBookStale, got reason=%v",
			result.Reason,
		)
	}

	if result.Opportunity != nil {
		t.Fatal("expected no opportunity")
	}
}

func TestSimulationServiceAcceptsFreshOrderBooks(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	orderBooks := map[string]marketDomain.OrderBook{
		"BTCUSDT": freshnessOrderBook(
			"BTCUSDT",
			now.Add(-100*time.Millisecond),
			"100000",
			"1",
			"100000",
			"1",
		),
		"ETHBTC": freshnessOrderBook(
			"ETHBTC",
			now.Add(-200*time.Millisecond),
			"0.05",
			"10",
			"0.05",
			"10",
		),
		"ETHUSDT": freshnessOrderBook(
			"ETHUSDT",
			now.Add(-300*time.Millisecond),
			"5100",
			"10",
			"5100",
			"10",
		),
	}

	service := newFreshnessSimulationService(
		orderBooks,
		map[string]time.Duration{
			"BTCUSDT": time.Second,
			"ETHBTC":  time.Second,
			"ETHUSDT": time.Second,
		},
	)

	result := service.Simulate(
		ctx,
		freshnessTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK, got reason=%v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity")
	}
}

func TestSimulationServiceUsesTTLPerSymbol(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	orderBooks := map[string]marketDomain.OrderBook{
		"BTCUSDT": freshnessOrderBook(
			"BTCUSDT",
			now.Add(-500*time.Millisecond),
			"100000",
			"1",
			"100000",
			"1",
		),
		"ETHBTC": freshnessOrderBook(
			"ETHBTC",
			now.Add(-2*time.Second),
			"0.05",
			"10",
			"0.05",
			"10",
		),
		"ETHUSDT": freshnessOrderBook(
			"ETHUSDT",
			now.Add(-500*time.Millisecond),
			"5100",
			"10",
			"5100",
			"10",
		),
	}

	service := newFreshnessSimulationService(
		orderBooks,
		map[string]time.Duration{
			"BTCUSDT": time.Second,
			"ETHBTC":  3 * time.Second,
			"ETHUSDT": time.Second,
		},
	)

	result := service.Simulate(
		ctx,
		freshnessTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK, got reason=%v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity")
	}
}

func TestSimulationServiceRejectsUnknownSymbolTTL(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	orderBooks := map[string]marketDomain.OrderBook{
		"BTCUSDT": freshnessOrderBook(
			"BTCUSDT",
			now.Add(-100*time.Millisecond),
			"100000",
			"1",
			"100000",
			"1",
		),
		"ETHBTC": freshnessOrderBook(
			"ETHBTC",
			now.Add(-100*time.Millisecond),
			"0.05",
			"10",
			"0.05",
			"10",
		),
		"ETHUSDT": freshnessOrderBook(
			"ETHUSDT",
			now.Add(-100*time.Millisecond),
			"5100",
			"10",
			"5100",
			"10",
		),
	}

	service := newFreshnessSimulationService(
		orderBooks,
		map[string]time.Duration{
			"BTCUSDT": time.Second,
			"ETHBTC":  time.Second,
			// ETHUSDT намеренно отсутствует.
		},
	)

	result := service.Simulate(
		ctx,
		freshnessTriangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOrderBookStale {
		t.Fatalf(
			"expected SimulationOrderBookStale, got reason=%v",
			result.Reason,
		)
	}

	if result.Opportunity != nil {
		t.Fatal("expected no opportunity")
	}
}
