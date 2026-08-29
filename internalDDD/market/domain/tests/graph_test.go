package tests

import (
	"strings"
	"testing"

	"traiderBot/internalDDD/market/domain"
)

func TestNewGraph(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	if len(graph) != 2 {
		t.Fatalf(
			"expected 2 assets in graph, got %d",
			len(graph),
		)
	}

	usdtEdges := graph["USDT"]

	if len(usdtEdges) != 1 {
		t.Fatalf(
			"expected 1 edge from USDT, got %d",
			len(usdtEdges),
		)
	}

	if usdtEdges[0].From != "USDT" {
		t.Fatalf(
			"expected edge From=USDT, got %q",
			usdtEdges[0].From,
		)
	}

	if usdtEdges[0].To != "BTC" {
		t.Fatalf(
			"expected edge To=BTC, got %q",
			usdtEdges[0].To,
		)
	}

	if usdtEdges[0].Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %q",
			usdtEdges[0].Symbol,
		)
	}

	if usdtEdges[0].Direction != domain.Buy {
		t.Fatalf(
			"expected Buy direction, got %v",
			usdtEdges[0].Direction,
		)

	}

	btcEdges := graph["BTC"]

	if len(btcEdges) != 1 {
		t.Fatalf(
			"expected 1 edge from BTC, got %d",
			len(btcEdges),
		)
	}

	if btcEdges[0].From != "BTC" {
		t.Fatalf(
			"expected edge From=BTC, got %q",
			btcEdges[0].From,
		)
	}

	if btcEdges[0].To != "USDT" {
		t.Fatalf(
			"expected edge To=USDT, got %q",
			btcEdges[0].To,
		)
	}

	if btcEdges[0].Direction != domain.Sell {
		t.Fatalf(
			"expected Sell direction, got %v",
			btcEdges[0].Direction,
		)
	}
}

func TestNewGraphEmptyBaseAsset(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "",
			QuoteAsset: "USDT",
		},
	}

	_, err := domain.NewGraph(symbols)

	if err == nil {
		t.Fatal("expected error for empty base asset")
	}

	if !strings.Contains(err.Error(), "empty base asset") {
		t.Fatalf(
			"expected empty base asset error, got %v",
			err,
		)
	}
}

func TestNewGraphEmptyQuoteAsset(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "",
		},
	}

	_, err := domain.NewGraph(symbols)

	if err == nil {
		t.Fatal("expected error for empty quote asset")
	}

	if !strings.Contains(err.Error(), "empty quote asset") {
		t.Fatalf(
			"expected empty quote asset error, got %v",
			err,
		)
	}
}

func TestGraphEdgeIsBuy(t *testing.T) {
	buyEdge := domain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: domain.Buy,
	}

	if !buyEdge.IsBuy() {
		t.Fatal("expected Buy edge to return true")
	}

	sellEdge := domain.GraphEdge{
		From:      "BTC",
		To:        "USDT",
		Symbol:    "BTCUSDT",
		Direction: domain.Sell,
	}

	if sellEdge.IsBuy() {
		t.Fatal("expected Sell edge to return false")
	}
}

func TestFindTriangularRoutes(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
		{
			Name:       "ETHBTC",
			BaseAsset:  "ETH",
			QuoteAsset: "BTC",
		},
		{
			Name:       "ETHUSDT",
			BaseAsset:  "ETH",
			QuoteAsset: "USDT",
		},
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		map[string]struct{}{
			"USDT": {},
		},
	)

	if len(routes) != 2 {
		t.Fatalf(
			"expected 2 triangular routes, got %d",
			len(routes),
		)
	}

	for _, route := range routes {
		if !route.IsTriangle() {
			t.Fatalf(
				"expected triangle route, got %v",
				route.Symbols(),
			)
		}

		if route.Edges[0].From != "USDT" {
			t.Fatalf(
				"expected route to start from USDT, got %q",
				route.Edges[0].From,
			)
		}
	}
}

func TestFindTriangularRoutesRespectsStartAssets(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
		{
			Name:       "ETHBTC",
			BaseAsset:  "ETH",
			QuoteAsset: "BTC",
		},
		{
			Name:       "ETHUSDT",
			BaseAsset:  "ETH",
			QuoteAsset: "USDT",
		},
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		map[string]struct{}{
			"BTC": {},
		},
	)

	if len(routes) != 2 {
		t.Fatalf(
			"expected 2 triangular routes, got %d",
			len(routes),
		)
	}

	for _, route := range routes {
		if route.Edges[0].From != "BTC" {
			t.Fatalf(
				"expected route to start from BTC, got %q",
				route.Edges[0].From,
			)
		}
	}
}

func TestFindTriangularRoutesNoStartAsset(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
		{
			Name:       "ETHBTC",
			BaseAsset:  "ETH",
			QuoteAsset: "BTC",
		},
		{
			Name:       "ETHUSDT",
			BaseAsset:  "ETH",
			QuoteAsset: "USDT",
		},
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		map[string]struct{}{
			"BNB": {},
		},
	)

	if len(routes) != 0 {
		t.Fatalf(
			"expected no routes, got %d",
			len(routes),
		)
	}
}

func TestFindTriangularRoutesDoesNotDuplicateRoutes(t *testing.T) {
	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
		{
			Name:       "ETHBTC",
			BaseAsset:  "ETH",
			QuoteAsset: "BTC",
		},
		{
			Name:       "ETHUSDT",
			BaseAsset:  "ETH",
			QuoteAsset: "USDT",
		},
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		map[string]struct{}{
			"USDT": {},
			"BTC":  {},
			"ETH":  {},
		},
	)

	keys := make(map[string]struct{})

	for _, route := range routes {
		key := route.RouteKey()

		if _, exists := keys[key]; exists {
			t.Fatalf(
				"duplicate route found with key %q",
				key,
			)
		}

		keys[key] = struct{}{}
	}
}
