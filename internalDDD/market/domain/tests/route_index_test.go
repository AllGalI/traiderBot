package tests

import (
	"testing"
	"traiderBot/internalDDD/market/domain"
)

func TestBuildRouteIndex(t *testing.T) {
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
		{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: domain.Sell,
		},
	})

	index := domain.BuildRouteIndex([]domain.Route{route1, route2})

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
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: domain.Sell,
		},
	})

	index := domain.BuildRouteIndex([]domain.Route{route})

	if len(index["BTCUSDT"]) != 1 {
		t.Fatalf(
			"expected 1 route for BTCUSDT, got %d",
			len(index["BTCUSDT"]),
		)
	}
}

func TestRouteIndexGetUnknownSymbol(t *testing.T) {
	index := domain.BuildRouteIndex(nil)

	routes := index.Get("UNKNOWN")

	if routes != nil {
		t.Fatalf("expected nil routes, got %v", routes)
	}
}
