package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	marketInfrastructure "traiderBot/internalDDD/market/infrastructure"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type workerBenchmarkMarketStructureReader struct {
	structure *marketDomain.MarketStructure
}

func (r *workerBenchmarkMarketStructureReader) Get() (
	*marketDomain.MarketStructure,
	bool,
) {
	return r.structure, true
}

type workerBenchmarkOrderBookReader struct {
	orderBooks map[string]marketDomain.OrderBook
}

func (r *workerBenchmarkOrderBookReader) Get(
	symbol string,
) (marketDomain.OrderBook, bool) {
	orderBook, ok := r.orderBooks[symbol]
	return orderBook, ok
}

type workerBenchmarkFeeProvider struct {
	fees map[string]simulationApplication.FeeConfig
}

func (p *workerBenchmarkFeeProvider) Get(
	ctx context.Context,
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := p.fees[symbol]
	return fee, ok
}

type workerBenchmarkOpportunityHandler struct{}

func (h *workerBenchmarkOpportunityHandler) Handle(
	opportunity *simulationDomain.Opportunity,
) {
}

type workerBenchmarkCountingSimulator struct {
	simulation simulationApplication.Simulator
	done       chan struct{}
}

func (s *workerBenchmarkCountingSimulator) Simulate(
	ctx context.Context,
	route marketDomain.Route,
	startAmount decimal.Decimal,
) simulationDomain.SimulationResult {
	result := s.simulation.Simulate(
		ctx,
		route,
		startAmount,
	)

	s.done <- struct{}{}

	return result
}

func workerBenchmarkSymbol(
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

func workerBenchmarkOrderBook(
	symbol string,
	price string,
	qty string,
) marketDomain.OrderBook {
	return marketDomain.OrderBook{
		Symbol:         symbol,
		LastUpdateTime: time.Now(),

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

func workerBenchmarkRoute() marketDomain.Route {
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

func workerBenchmarkStructure(
	routesCount int,
) *marketDomain.MarketStructure {
	routes := make(
		[]marketDomain.Route,
		0,
		routesCount,
	)

	for i := 0; i < routesCount; i++ {
		routes = append(
			routes,
			workerBenchmarkRoute(),
		)
	}

	return &marketDomain.MarketStructure{
		Symbols: []marketDomain.Symbol{
			workerBenchmarkSymbol(
				"BTCUSDT",
				"BTC",
				"USDT",
			),
			workerBenchmarkSymbol(
				"ETHBTC",
				"ETH",
				"BTC",
			),
			workerBenchmarkSymbol(
				"ETHUSDT",
				"ETH",
				"USDT",
			),
		},

		Routes:     routes,
		RouteIndex: marketDomain.BuildRouteIndex(routes),
	}
}

func workerBenchmarkSimulationService() *simulationApplication.SimulationService {
	orderBooks := map[string]marketDomain.OrderBook{
		"BTCUSDT": workerBenchmarkOrderBook(
			"BTCUSDT",
			"100000",
			"10",
		),

		"ETHBTC": workerBenchmarkOrderBook(
			"ETHBTC",
			"0.05",
			"100",
		),

		"ETHUSDT": workerBenchmarkOrderBook(
			"ETHUSDT",
			"5100",
			"100",
		),
	}

	fees := &workerBenchmarkFeeProvider{
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

	structureStore :=
		marketInfrastructure.NewInMemoryMarketStructureStore()

	structureStore.Set(
		&marketDomain.MarketStructure{
			Symbols: []marketDomain.Symbol{
				workerBenchmarkSymbol(
					"BTCUSDT",
					"BTC",
					"USDT",
				),
				workerBenchmarkSymbol(
					"ETHBTC",
					"ETH",
					"BTC",
				),
				workerBenchmarkSymbol(
					"ETHUSDT",
					"ETH",
					"USDT",
				),
			},
		},
	)

	symbols :=
		simulationApplication.NewMarketStructureSymbolProvider(
			structureStore,
		)

	freshness :=
		simulationApplication.NewFixedTTLFreshnessPolicy(
			map[string]time.Duration{
				"BTCUSDT": time.Hour,
				"ETHBTC":  time.Hour,
				"ETHUSDT": time.Hour,
			},
		)

	return simulationApplication.NewSimulationService(
		&workerBenchmarkOrderBookReader{
			orderBooks: orderBooks,
		},
		fees,
		symbols,
		freshness,
		decimal.RequireFromString("0.90"),
		decimal.Zero,
	)
}

func BenchmarkWorkerWithRealSimulation(b *testing.B) {
	cases := []struct {
		name   string
		routes int
	}{
		{
			name:   "10Routes",
			routes: 10,
		},
		{
			name:   "100Routes",
			routes: 100,
		},
		{
			name:   "500Routes",
			routes: 500,
		},
		{
			name:   "1000Routes",
			routes: 1000,
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			structure := workerBenchmarkStructure(
				tc.routes,
			)

			simulation := workerBenchmarkSimulationService()

			done := make(chan struct{}, b.N*tc.routes)

			countingSimulator :=
				&workerBenchmarkCountingSimulator{
					simulation: simulation,
					done:       done,
				}

			updates := make(chan string, 1024)

			worker := simulationApplication.NewWorker(
				&workerBenchmarkMarketStructureReader{
					structure: structure,
				},
				updates,
				countingSimulator,
				&workerBenchmarkOpportunityHandler{},
				decimal.NewFromInt(100),
			)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)

			workerDone := make(chan error, 1)

			go func() {
				workerDone <- worker.Run(ctx)
			}()

			expectedSimulations := b.N * tc.routes

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				updates <- "BTCUSDT"
			}

			for i := 0; i < expectedSimulations; i++ {
				<-done
			}

			b.StopTimer()

			cancel()

			err := <-workerDone

			if err != nil &&
				!errors.Is(err, context.Canceled) {
				b.Fatalf(
					"worker stopped with unexpected error: %v",
					err,
				)
			}

			b.ReportMetric(
				float64(expectedSimulations),
				"simulations",
			)

			b.ReportMetric(
				float64(expectedSimulations)/float64(b.N),
				"sim/update",
			)
		})
	}
}
