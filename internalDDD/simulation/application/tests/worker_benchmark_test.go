package tests

// import (
// 	"context"
// 	"testing"

// 	"github.com/shopspring/decimal"

// 	marketDomain "traiderBot/internalDDD/market/domain"
// 	simulationApplication "traiderBot/internalDDD/simulation/application"
// 	simulationDomain "traiderBot/internalDDD/simulation/domain"
// )

// type benchmarkMarketStructureReader struct {
// 	structure *marketDomain.MarketStructure
// }

// func (r *benchmarkMarketStructureReader) Get() (
// 	*marketDomain.MarketStructure,
// 	bool,
// ) {
// 	return r.structure, true
// }

// type benchmarkSimulator struct {
// 	calls chan struct{}
// }

// func (s *benchmarkSimulator) Simulate(
// 	ctx context.Context,
// 	route marketDomain.Route,
// 	startAmount decimal.Decimal,
// ) simulationDomain.SimulationResult {
// 	s.calls <- struct{}{}

// 	return simulationDomain.SimulationResult{}
// }

// type benchmarkOpportunityHandler struct{}

// func (h *benchmarkOpportunityHandler) Handle(
// 	opportunity *simulationDomain.Opportunity,
// ) {
// }

// func benchmarkRoute(
// 	symbol string,
// ) marketDomain.Route {
// 	return marketDomain.Route{
// 		Edges: []marketDomain.GraphEdge{
// 			{
// 				From:      "USDT",
// 				To:        "BTC",
// 				Symbol:    symbol,
// 				Direction: marketDomain.Buy,
// 			},
// 			{
// 				From:      "BTC",
// 				To:        "ETH",
// 				Symbol:    "ETHBTC",
// 				Direction: marketDomain.Buy,
// 			},
// 			{
// 				From:      "ETH",
// 				To:        "USDT",
// 				Symbol:    "ETHUSDT",
// 				Direction: marketDomain.Sell,
// 			},
// 		},
// 	}
// }

// func BenchmarkWorker(b *testing.B) {
// 	const (
// 		symbol = "BTCUSDT"

// 		routesPerUpdate = 100
// 	)

// 	routes := make([]marketDomain.Route, 0, routesPerUpdate)

// 	for i := 0; i < routesPerUpdate; i++ {
// 		routes = append(
// 			routes,
// 			benchmarkRoute(symbol),
// 		)
// 	}

// 	structure := &marketDomain.MarketStructure{
// 		Routes:     routes,
// 		RouteIndex: marketDomain.BuildRouteIndex(routes),
// 	}

// 	updates := make(chan string, b.N)

// 	simulator := &benchmarkSimulator{
// 		calls: make(chan struct{}, b.N*routesPerUpdate),
// 	}

// 	worker := simulationApplication.NewWorker(
// 		&benchmarkMarketStructureReader{
// 			structure: structure,
// 		},
// 		updates,
// 		simulator,
// 		&benchmarkOpportunityHandler{},
// 		decimal.NewFromInt(100),
// 	)

// 	ctx, cancel := context.WithCancel(
// 		context.Background(),
// 	)

// 	done := make(chan error, 1)

// 	go func() {
// 		done <- worker.Run(ctx)
// 	}()

// 	b.ResetTimer()

// 	for i := 0; i < b.N; i++ {
// 		updates <- symbol
// 	}

// 	for i := 0; i < b.N*routesPerUpdate; i++ {
// 		<-simulator.calls
// 	}

// 	b.StopTimer()

// 	cancel()

// 	<-done

// 	b.ReportMetric(
// 		float64(b.N*routesPerUpdate),
// 		"simulations",
// 	)
// }
