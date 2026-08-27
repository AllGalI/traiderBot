package domain

import "testing"

func TestBuildRouteIndex(t *testing.T) {
	route1 := NewRoute([]GraphEdge{
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

	route2 := NewRoute([]GraphEdge{
		{
			From:      "USDT",
			To:        "ETH",
			Symbol:    "ETHUSDT",
			Direction: Buy,
		},
		{
			From:      "ETH",
			To:        "BTC",
			Symbol:    "ETHBTC",
			Direction: Sell,
		},
		{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: Sell,
		},
	})

	index := BuildRouteIndex([]Route{route1, route2})

	if len(index["BTCUSDT"]) != 2 {
		t.Fatalf(
			"expected 2 routes for BTCUSDT, got %d",
			len(index["BTCUSDT"]),
		)
	}

	if len(index["ETHBTC"]) != 2 {
		t.Fatalf(
			"expected 2 routes for ETHBTC, got %d",
			len(index["ETHBTC"]),
		)
	}

	if len(index["ETHUSDT"]) != 2 {
		t.Fatalf(
			"expected 2 routes for ETHUSDT, got %d",
			len(index["ETHUSDT"]),
		)
	}
}

func TestBuildRouteIndexDoesNotDuplicateSymbolInRoute(t *testing.T) {
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
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: Sell,
		},
	})

	index := BuildRouteIndex([]Route{route})

	if len(index["BTCUSDT"]) != 1 {
		t.Fatalf(
			"expected 1 route for BTCUSDT, got %d",
			len(index["BTCUSDT"]),
		)
	}
}

func TestRouteIndexGetUnknownSymbol(t *testing.T) {
	index := BuildRouteIndex(nil)

	routes := index.Get("UNKNOWN")

	if routes != nil {
		t.Fatalf("expected nil routes, got %v", routes)
	}
}
