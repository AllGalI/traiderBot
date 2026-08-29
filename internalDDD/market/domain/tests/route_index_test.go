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

	index := domain.BuildRouteIndex([]domain.Route{
		route1,
		route2,
	})

	if len(index) != 3 {
		t.Fatalf(
			"expected 3 symbols in index, got %d",
			len(index),
		)
	}

	for _, symbol := range []string{
		"BTCUSDT",
		"ETHBTC",
		"ETHUSDT",
	} {
		routes := index.Get(symbol)

		if len(routes) != 2 {
			t.Fatalf(
				"expected 2 routes for %s, got %d",
				symbol,
				len(routes),
			)
		}
	}
}

func TestBuildRouteIndexDoesNotDuplicateSymbolWithinRoute(t *testing.T) {
	route := domain.NewRoute([]domain.GraphEdge{
		{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: domain.Buy,
		},
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
	})

	index := domain.BuildRouteIndex([]domain.Route{
		route,
	})

	if len(index.Get("BTCUSDT")) != 1 {
		t.Fatalf(
			"expected route to appear once for BTCUSDT, got %d",
			len(index.Get("BTCUSDT")),
		)
	}

	if len(index.Get("ETHUSDT")) != 1 {
		t.Fatalf(
			"expected route to appear once for ETHUSDT, got %d",
			len(index.Get("ETHUSDT")),
		)
	}
}

func TestRouteIndexGetMissingSymbol(t *testing.T) {
	index := domain.BuildRouteIndex(nil)

	routes := index.Get("BTCUSDT")

	if len(routes) != 0 {
		t.Fatalf(
			"expected no routes for missing symbol, got %d",
			len(routes),
		)
	}
}

func TestRouteIndexSymbols(t *testing.T) {
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

	index := domain.BuildRouteIndex([]domain.Route{
		route,
	})

	symbols := index.Symbols()

	if len(symbols) != 3 {
		t.Fatalf(
			"expected 3 symbols, got %d",
			len(symbols),
		)
	}

	found := make(map[string]bool)

	for _, symbol := range symbols {
		found[symbol] = true
	}

	for _, expected := range []string{
		"BTCUSDT",
		"ETHBTC",
		"ETHUSDT",
	} {
		if !found[expected] {
			t.Fatalf(
				"expected symbol %q in index",
				expected,
			)
		}
	}
}
