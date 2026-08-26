package domain

import "testing"

func TestFindTriangularRoutes(t *testing.T) {
	symbols := []Symbol{
		{
			Name:      "BTCUSDT",
			BaseAsset: "BTC",
			QuoteAsset: "USDT",
		},
		{
			Name:      "ETHBTC",
			BaseAsset: "ETH",
			QuoteAsset: "BTC",
		},
		{
			Name:      "ETHUSDT",
			BaseAsset: "ETH",
			QuoteAsset: "USDT",
		},
	}

	graph, err := NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	routes := FindTriangularRoutes(
		graph,
		map[string]struct{}{
			"USDT": {},
		},
	)

	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	foundForward := false
	foundReverse := false

	for _, route := range routes {
		if len(route.Edges) != 3 {
			t.Fatalf("expected 3 edges, got %d", len(route.Edges))
		}

		if !route.IsTriangle() {
			t.Fatalf("route is not a triangle: %v", route)
		}

		if route.Edges[0].From != "USDT" {
			t.Errorf(
				"route does not start from USDT: %v",
				route.Symbols(),
			)
		}

		if route.Edges[0].Symbol == "BTCUSDT" &&
			route.Edges[1].Symbol == "ETHBTC" &&
			route.Edges[2].Symbol == "ETHUSDT" {

			foundForward = true
		}

		if route.Edges[0].Symbol == "ETHUSDT" &&
			route.Edges[1].Symbol == "ETHBTC" &&
			route.Edges[2].Symbol == "BTCUSDT" {

			foundReverse = true
		}
	}

	if !foundForward {
		t.Error("forward triangle was not found")
	}

	if !foundReverse {
		t.Error("reverse triangle was not found")
	}
}

func TestRouteKey(t *testing.T) {
	route := NewRoute([]GraphEdge{
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: Buy,
		},
		{
			From:      "BTC",
			To:        "ETH",
			Symbol:    "ETHBTC",
			Direction: Buy,
		},
		{
			From:      "ETH",
			To:        "USDT",
			Symbol:    "ETHUSDT",
			Direction: Sell,
		},
	})

	if route.Key() == "" {
		t.Fatal("expected non-empty route key")
	}
}

func TestRouteIsTriangle(t *testing.T) {
	route := NewRoute([]GraphEdge{
		{
			From: "USDT",
			To:   "BTC",
		},
		{
			From: "BTC",
			To:   "ETH",
		},
		{
			From: "ETH",
			To:   "USDT",
		},
	})

	if !route.IsTriangle() {
		t.Fatal("expected triangle")
	}
}