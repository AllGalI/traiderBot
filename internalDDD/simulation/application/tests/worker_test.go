package tests

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type workerMarketStructureReader struct {
	structure *marketDomain.MarketStructure
	ok        bool
}

func (r *workerMarketStructureReader) Get() (
	*marketDomain.MarketStructure,
	bool,
) {
	return r.structure, r.ok
}

type workerSimulator struct {
	calls []marketDomain.Route
}

func (s *workerSimulator) Simulate(
	ctx context.Context,
	route marketDomain.Route,
	startAmount decimal.Decimal,
) simulationDomain.SimulationResult {
	s.calls = append(
		s.calls,
		route,
	)

	return simulationDomain.SimulationResult{}
}

type workerOpportunityHandler struct {
	opportunities []*simulationDomain.Opportunity
}

func (h *workerOpportunityHandler) Handle(
	opportunity *simulationDomain.Opportunity,
) {
	h.opportunities = append(
		h.opportunities,
		opportunity,
	)
}

func workerRoute(
	symbol1 string,
	symbol2 string,
	symbol3 string,
) marketDomain.Route {
	return marketDomain.Route{
		Edges: []marketDomain.GraphEdge{
			{
				From:      "USDT",
				To:        "A",
				Symbol:    symbol1,
				Direction: marketDomain.Buy,
			},
			{
				From:      "A",
				To:        "B",
				Symbol:    symbol2,
				Direction: marketDomain.Buy,
			},
			{
				From:      "B",
				To:        "USDT",
				Symbol:    symbol3,
				Direction: marketDomain.Sell,
			},
		},
	}
}

func TestWorkerSimulatesOnlyRoutesAffectedByOrderBookUpdate(t *testing.T) {
	t.Log("TEST: starting")

	t.Log("creating routes")

	routeBTC := workerRoute(
		"BTCUSDT",
		"ETHBTC",
		"ETHUSDT",
	)

	routeSOL := workerRoute(
		"SOLUSDT",
		"DOGESOL",
		"DOGEUSDT",
	)

	routeXRP := workerRoute(
		"XRPUSDT",
		"ADAXRP",
		"ADAUSDT",
	)

	t.Logf(
		"routeBTC symbols=%v",
		routeBTC.Symbols(),
	)

	t.Logf(
		"routeSOL symbols=%v",
		routeSOL.Symbols(),
	)

	t.Logf(
		"routeXRP symbols=%v",
		routeXRP.Symbols(),
	)

	t.Log("building market structure")

	routes := []marketDomain.Route{
		routeBTC,
		routeSOL,
		routeXRP,
	}

	structure := &marketDomain.MarketStructure{
		Routes: routes,
		RouteIndex: marketDomain.BuildRouteIndex(
			routes,
		),
	}

	t.Logf(
		"market structure contains %d routes",
		len(structure.Routes),
	)

	t.Logf(
		"BTCUSDT indexed routes=%d",
		len(structure.RouteIndex.Get("BTCUSDT")),
	)

	if len(structure.RouteIndex.Get("BTCUSDT")) != 1 {
		t.Fatalf(
			"expected BTCUSDT to affect exactly 1 route, got %d",
			len(structure.RouteIndex.Get("BTCUSDT")),
		)
	}

	t.Log("creating updates channel")

	updates := make(chan string, 1)

	simulator := &workerSimulator{}
	opportunities := &workerOpportunityHandler{}

	t.Log("creating simulation worker")

	worker := simulationApplication.NewWorker(
		&workerMarketStructureReader{
			structure: structure,
			ok:        true,
		},
		updates,
		simulator,
		opportunities,
		decimal.NewFromInt(100),
	)

	t.Log("creating context")

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	done := make(chan error, 1)

	t.Log("starting worker")

	go func() {
		t.Log("worker.Run(): entered")

		err := worker.Run(ctx)

		t.Logf(
			"worker.Run(): finished err=%v",
			err,
		)

		done <- err
	}()

	t.Log("sending BTCUSDT update")

	updates <- "BTCUSDT"

	t.Log("waiting for worker to process update")

	deadline := time.After(2 * time.Second)

	for {
		if len(simulator.calls) == 1 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf(
				"timeout waiting for simulation, calls=%d",
				len(simulator.calls),
			)

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	t.Logf(
		"simulation calls=%d",
		len(simulator.calls),
	)

	if len(simulator.calls) != 1 {
		t.Fatalf(
			"expected 1 simulation call, got %d",
			len(simulator.calls),
		)
	}

	calledRoute := simulator.calls[0]

	t.Logf(
		"simulated route=%v",
		calledRoute.Symbols(),
	)

	found := false

	for _, edge := range calledRoute.Edges {
		if edge.Symbol == "BTCUSDT" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"worker simulated unrelated route: %v",
			calledRoute.Symbols(),
		)
	}

	t.Log("stopping worker")

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}

	t.Log("worker stopped")

	t.Log("TEST: finished")
}