package tests

import (
	"testing"

	"traiderBot/internalDDD/market/domain"
)

func TestNewRoute(t *testing.T) {
	edges := []domain.GraphEdge{
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: domain.Buy,
		},
		{
			From:      "BTC",
			To:        "ETH",
			Symbol:    "ETHBTC",
			Direction: domain.Buy,
		},
		{
			From:      "ETH",
			To:        "USDT",
			Symbol:    "ETHUSDT",
			Direction: domain.Sell,
		},
	}

	route := domain.NewRoute(edges)

	if len(route.Edges) != 3 {
		t.Fatalf(
			"expected 3 edges, got %d",
			len(route.Edges),
		)
	}

	if route.Edges[0].Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected first symbol BTCUSDT, got %q",
			route.Edges[0].Symbol,
		)
	}

	if route.Key == "" {
		t.Fatal("expected route key to be generated")
	}
}

func TestRouteSymbols(t *testing.T) {
	route := domain.NewRoute([]domain.GraphEdge{
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: domain.Buy,
		},
		{
			From:      "BTC",
			To:        "ETH",
			Symbol:    "ETHBTC",
			Direction: domain.Buy,
		},
		{
			From:      "ETH",
			To:        "USDT",
			Symbol:    "ETHUSDT",
			Direction: domain.Sell,
		},
	})

	symbols := route.Symbols()

	expected := []string{
		"BTCUSDT",
		"ETHBTC",
		"ETHUSDT",
	}

	if len(symbols) != len(expected) {
		t.Fatalf(
			"expected %d symbols, got %d",
			len(expected),
			len(symbols),
		)
	}

	for i := range expected {
		if symbols[i] != expected[i] {
			t.Fatalf(
				"expected symbol %d to be %q, got %q",
				i,
				expected[i],
				symbols[i],
			)
		}
	}
}

func TestRouteIsTriangle(t *testing.T) {
	tests := []struct {
		name  string
		edges []domain.GraphEdge
		want  bool
	}{
		{
			name: "valid triangle",
			edges: []domain.GraphEdge{
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
			},
			want: true,
		},
		{
			name: "wrong number of edges",
			edges: []domain.GraphEdge{
				{
					From: "USDT",
					To:   "BTC",
				},
				{
					From: "BTC",
					To:   "ETH",
				},
			},
			want: false,
		},
		{
			name: "broken connection",
			edges: []domain.GraphEdge{
				{
					From: "USDT",
					To:   "BTC",
				},
				{
					From: "ETH",
					To:   "BTC",
				},
				{
					From: "BTC",
					To:   "USDT",
				},
			},
			want: false,
		},
		{
			name: "does not return to start",
			edges: []domain.GraphEdge{
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
					To:   "BNB",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := domain.NewRoute(tt.edges)

			if got := route.IsTriangle(); got != tt.want {
				t.Fatalf(
					"IsTriangle() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestRouteKeyCanonicalizesRotation(t *testing.T) {
	route1 := domain.NewRoute([]domain.GraphEdge{
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: domain.Buy,
		},
		{
			From:      "BTC",
			To:        "ETH",
			Symbol:    "ETHBTC",
			Direction: domain.Buy,
		},
		{
			From:      "ETH",
			To:        "USDT",
			Symbol:    "ETHUSDT",
			Direction: domain.Sell,
		},
	})

	route2 := domain.NewRoute([]domain.GraphEdge{
		{
			From:      "BTC",
			To:        "ETH",
			Symbol:    "ETHBTC",
			Direction: domain.Buy,
		},
		{
			From:      "ETH",
			To:        "USDT",
			Symbol:    "ETHUSDT",
			Direction: domain.Sell,
		},
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: domain.Buy,
		},
	})

	if route1.RouteKey() != route2.RouteKey() {
		t.Fatalf(
			"expected rotated routes to have equal RouteKey, got %q and %q",
			route1.RouteKey(),
			route2.RouteKey(),
		)
	}
}

func TestRouteKeyIncludesDirection(t *testing.T) {
	buyRoute := domain.NewRoute([]domain.GraphEdge{
		{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: domain.Buy,
		},
		{
			From:      "USDT",
			To:        "ETH",
			Symbol:    "ETHUSDT",
			Direction: domain.Buy,
		},
		{
			From:      "ETH",
			To:        "BTC",
			Symbol:    "ETHBTC",
			Direction: domain.Sell,
		},
	})

	sellRoute := domain.NewRoute([]domain.GraphEdge{
		{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: domain.Sell,
		},
		{
			From:      "USDT",
			To:        "ETH",
			Symbol:    "ETHUSDT",
			Direction: domain.Buy,
		},
		{
			From:      "ETH",
			To:        "BTC",
			Symbol:    "ETHBTC",
			Direction: domain.Sell,
		},
	})

	if buyRoute.RouteKey() == sellRoute.RouteKey() {
		t.Fatal("expected routes with different directions to have different RouteKey")
	}
}
