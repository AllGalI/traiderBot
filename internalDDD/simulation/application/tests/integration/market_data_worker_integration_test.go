package integration

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	marketApplication "traiderBot/internalDDD/market/application"
	marketDomain "traiderBot/internalDDD/market/domain"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type integrationOrderBookProvider struct {
	orderBooks map[string]marketDomain.OrderBook
}

func (p *integrationOrderBookProvider) Get(
	ctx context.Context,
	symbol string,
	depth int,
) (marketDomain.OrderBook, error) {
	orderBook, ok := p.orderBooks[symbol]
	if !ok {
		return marketDomain.OrderBook{}, context.Canceled
	}

	return orderBook, nil
}

type integrationOrderBookStream struct {
	handler marketApplication.OrderBookUpdateHandler
}

func (s *integrationOrderBookStream) Subscribe(
	ctx context.Context,
	symbols []string,
	depth int,
	handler marketApplication.OrderBookUpdateHandler,
) error {
	s.handler = handler
	return nil
}

type integrationOrderBookStore struct {
	orderBooks map[string]marketDomain.OrderBook
}

func newIntegrationOrderBookStore() *integrationOrderBookStore {
	return &integrationOrderBookStore{
		orderBooks: make(map[string]marketDomain.OrderBook),
	}
}

func (s *integrationOrderBookStore) Get(
	symbol string,
) (marketDomain.OrderBook, bool) {
	orderBook, ok := s.orderBooks[symbol]
	return orderBook, ok
}

func (s *integrationOrderBookStore) Set(
	orderBook marketDomain.OrderBook,
) {
	s.orderBooks[orderBook.Symbol] = orderBook
}

type integrationMarketStructureReader struct {
	structure *marketDomain.MarketStructure
}

func (r *integrationMarketStructureReader) Get() (
	*marketDomain.MarketStructure,
	bool,
) {
	return r.structure, true
}

type integrationSimulator struct {
	calls chan marketDomain.Route
}

func (s *integrationSimulator) Simulate(
	ctx context.Context,
	route marketDomain.Route,
	startAmount decimal.Decimal,
) simulationDomain.SimulationResult {
	s.calls <- route

	return simulationDomain.SimulationResult{}
}

type integrationOpportunityHandler struct{}

func (h *integrationOpportunityHandler) Handle(
	opportunity *simulationDomain.Opportunity,
) {
}

func integrationRoute(
	first string,
	second string,
	third string,
) marketDomain.Route {
	return marketDomain.Route{
		Edges: []marketDomain.GraphEdge{
			{
				From:      "USDT",
				To:        first,
				Symbol:    first + "USDT",
				Direction: marketDomain.Buy,
			},
			{
				From:      first,
				To:        second,
				Symbol:    second + first,
				Direction: marketDomain.Buy,
			},
			{
				From:      second,
				To:        "USDT",
				Symbol:    third,
				Direction: marketDomain.Sell,
			},
		},
	}
}

func TestMarketDataServiceToWorkerIntegration(
	t *testing.T,
) {
	t.Log("TEST: starting")

	routeBTC := integrationRoute(
		"BTC",
		"ETH",
		"ETHUSDT",
	)

	routeETH := integrationRoute(
		"ETH",
		"BTC",
		"BTCUSDT",
	)

	routeSOL := integrationRoute(
		"SOL",
		"BTC",
		"BTCUSDT",
	)

	t.Log("TEST: building market structure")

	routes := []marketDomain.Route{
		routeBTC,
		routeETH,
		routeSOL,
	}

	structure := &marketDomain.MarketStructure{
		Routes:    routes,
		RouteIndex: marketDomain.BuildRouteIndex(routes),
	}

	t.Logf(
		"TEST: market structure routes=%d",
		len(structure.Routes),
	)

	t.Logf(
		"TEST: BTCUSDT indexed routes=%d",
		len(structure.RouteIndex.Get("BTCUSDT")),
	)

	provider := &integrationOrderBookProvider{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
		},
	}

	stream := &integrationOrderBookStream{}
	store := newIntegrationOrderBookStore()

	t.Log("TEST: creating MarketDataService")

	marketDataService := marketApplication.NewMarketDataService(
		provider,
		stream,
		store,
	)

	t.Log("TEST: starting MarketDataService")

	err := marketDataService.Start(
		context.Background(),
		[]string{"BTCUSDT"},
		10,
	)
	if err != nil {
		t.Fatalf(
			"MarketDataService.Start() failed: %v",
			err,
		)
	}

	if stream.handler == nil {
		t.Fatal("expected stream handler")
	}

	t.Log("TEST: initial snapshot loaded")

	updates := marketDataService.Updates()

	if updates == nil {
		t.Fatal("expected updates channel")
	}

	t.Log("TEST: creating simulation worker")

	simulator := &integrationSimulator{
		calls: make(chan marketDomain.Route, 10),
	}

	worker := simulationApplication.NewWorker(
		&integrationMarketStructureReader{
			structure: structure,
		},
		updates,
		simulator,
		&integrationOpportunityHandler{},
		decimal.NewFromInt(100),
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	done := make(chan error, 1)

	t.Log("TEST: starting simulation worker")

	go func() {
		t.Log("TEST: worker.Run() entered")

		err := worker.Run(ctx)

		t.Logf(
			"TEST: worker.Run() finished: %v",
			err,
		)

		done <- err
	}()

	t.Log("TEST: sending BTCUSDT order book update")

	stream.handler(
		marketDomain.OrderBook{
			Symbol:       "BTCUSDT",
			LastUpdateID: 200,
			LastUpdateTime: time.Now(),
		},
	)

	t.Log("TEST: waiting for simulation")

	var calls []marketDomain.Route

	timeout := time.After(2 * time.Second)

	for len(calls) < 3 {
		select {
		case route := <-simulator.calls:
			calls = append(calls, route)

			t.Logf(
				"TEST: simulation call #%d: %v",
				len(calls),
				route.Symbols(),
			)

		case <-timeout:
			t.Fatalf(
				"timeout waiting for simulations, got %d calls",
				len(calls),
			)
		}
	}

	t.Logf(
		"TEST: received %d simulation calls",
		len(calls),
	)

	if len(calls) != 3 {
		t.Fatalf(
			"expected 3 simulation calls for BTCUSDT, got %d",
			len(calls),
		)
	}

	for _, route := range calls {
		found := false

		for _, edge := range route.Edges {
			if edge.Symbol == "BTCUSDT" {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf(
				"worker simulated unrelated route: %v",
				route.Symbols(),
			)
		}
	}

	t.Log("TEST: stopping worker")

	cancel()

	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf(
				"worker stopped with unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}

	t.Log("TEST: finished successfully")
}


func TestMarketDataServiceToWorkerProcessesMultipleUpdates(
	t *testing.T,
) {
	t.Log("TEST: starting multiple updates integration test")

	routeBTC := integrationRoute(
		"BTC",
		"ETH",
		"ETHUSDT",
	)

	routeETH := integrationRoute(
		"ETH",
		"BTC",
		"BTCUSDT",
	)

	routeSOL := integrationRoute(
		"SOL",
		"BTC",
		"BTCUSDT",
	)

	routes := []marketDomain.Route{
		routeBTC,
		routeETH,
		routeSOL,
	}

	structure := &marketDomain.MarketStructure{
		Routes:     routes,
		RouteIndex: marketDomain.BuildRouteIndex(routes),
	}

	provider := &integrationOrderBookProvider{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
			"ETHUSDT": {
				Symbol:       "ETHUSDT",
				LastUpdateID: 200,
			},
		},
	}

	stream := &integrationOrderBookStream{}
	store := newIntegrationOrderBookStore()

	t.Log("TEST: creating MarketDataService")

	marketDataService := marketApplication.NewMarketDataService(
		provider,
		stream,
		store,
	)

	t.Log("TEST: starting MarketDataService")

	err := marketDataService.Start(
		context.Background(),
		[]string{
			"BTCUSDT",
			"ETHUSDT",
		},
		10,
	)
	if err != nil {
		t.Fatalf(
			"MarketDataService.Start() failed: %v",
			err,
		)
	}

	if stream.handler == nil {
		t.Fatal("expected stream handler")
	}

	t.Log("TEST: creating simulation worker")

	simulator := &integrationSimulator{
		calls: make(chan marketDomain.Route, 20),
	}

	worker := simulationApplication.NewWorker(
		&integrationMarketStructureReader{
			structure: structure,
		},
		marketDataService.Updates(),
		simulator,
		&integrationOpportunityHandler{},
		decimal.NewFromInt(100),
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	done := make(chan error, 1)

	t.Log("TEST: starting worker")

	go func() {
		err := worker.Run(ctx)
		done <- err
	}()

	/*
		Порядок событий:

		1. BTCUSDT
		2. ETHUSDT
		3. BTCUSDT

		Для каждого события worker должен обработать
		только routes, связанные с соответствующим symbol.
	*/

	updates := []string{
		"BTCUSDT",
		"ETHUSDT",
		"BTCUSDT",
	}

	t.Logf(
		"TEST: sending %d updates",
		len(updates),
	)

	for i, symbol := range updates {
		t.Logf(
			"TEST: sending update #%d symbol=%s",
			i+1,
			symbol,
		)

		stream.handler(
			marketDomain.OrderBook{
				Symbol:         symbol,
				LastUpdateID:   int64(300 + i),
				LastUpdateTime: time.Now(),
			},
		)
	}

	/*
		В нашей структуре:

		BTCUSDT -> 3 routes
		ETHUSDT -> 1 route

		Значит ожидаем:

		BTCUSDT -> 3
		ETHUSDT -> 1
		BTCUSDT -> 3

		Итого 7 simulation calls.
	*/

	expectedCalls := 7

	t.Logf(
		"TEST: waiting for %d simulation calls",
		expectedCalls,
	)

	var calls []marketDomain.Route

	timeout := time.After(2 * time.Second)

	for len(calls) < expectedCalls {
		select {
		case route := <-simulator.calls:
			calls = append(calls, route)

			t.Logf(
				"TEST: simulation call #%d: %v",
				len(calls),
				route.Symbols(),
			)

		case <-timeout:
			t.Fatalf(
				"timeout waiting for simulation calls: got %d, expected %d",
				len(calls),
				expectedCalls,
			)
		}
	}

	t.Logf(
		"TEST: received all %d simulation calls",
		len(calls),
	)

	if len(calls) != expectedCalls {
		t.Fatalf(
			"expected %d simulation calls, got %d",
			expectedCalls,
			len(calls),
		)
	}

	/*
		Проверяем, что каждый route действительно содержит
		символ, который вызвал simulation.

		Ожидаемый порядок:

		0-2 -> BTCUSDT
		3   -> ETHUSDT
		4-6 -> BTCUSDT
	*/

	expectedSymbols := []string{
		"BTCUSDT",
		"BTCUSDT",
		"BTCUSDT",

		"ETHUSDT",

		"BTCUSDT",
		"BTCUSDT",
		"BTCUSDT",
	}

	for i, route := range calls {
		expectedSymbol := expectedSymbols[i]

		found := false

		for _, edge := range route.Edges {
			if edge.Symbol == expectedSymbol {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf(
				"simulation call #%d: route %v does not contain expected symbol %s",
				i+1,
				route.Symbols(),
				expectedSymbol,
			)
		}
	}

	t.Log("TEST: all simulation calls belong to expected symbols")

	t.Log("TEST: stopping worker")

	cancel()

	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf(
				"worker stopped with unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}

	t.Log("TEST: multiple updates integration test finished")
}