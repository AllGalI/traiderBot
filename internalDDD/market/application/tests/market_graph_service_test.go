package tests

import (
	"testing"

	"traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/domain"
)

func TestMarketGraphServiceBuild(t *testing.T) {
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

	service := application.NewMarketGraphService()

	result, err := service.Build(
		symbols,
		map[string]struct{}{
			"USDT": {},
		},
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(result.Graph) == 0 {
		t.Fatal("expected graph to be built")
	}

	if len(result.Routes) != 2 {
		t.Fatalf(
			"expected 2 routes, got %d",
			len(result.Routes),
		)
	}

	if len(result.RouteIndex["BTCUSDT"]) != 2 {
		t.Fatalf(
			"expected 2 routes for BTCUSDT, got %d",
			len(result.RouteIndex["BTCUSDT"]),
		)
	}

	if len(result.RouteIndex["ETHBTC"]) != 2 {
		t.Fatalf(
			"expected 2 routes for ETHBTC, got %d",
			len(result.RouteIndex["ETHBTC"]),
		)
	}

	if len(result.RouteIndex["ETHUSDT"]) != 2 {
		t.Fatalf(
			"expected 2 routes for ETHUSDT, got %d",
			len(result.RouteIndex["ETHUSDT"]),
		)
	}
}

func TestMarketGraphServiceBuildInvalidSymbol(t *testing.T) {
	service := application.NewMarketGraphService()

	symbols := []domain.Symbol{
		{
			Name:       "INVALID",
			BaseAsset:  "",
			QuoteAsset: "USDT",
		},
	}

	_, err := service.Build(
		symbols,
		map[string]struct{}{
			"USDT": {},
		},
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMarketGraphServiceBuildNoRoutes(t *testing.T) {
	service := application.NewMarketGraphService()

	symbols := []domain.Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "USDT",
		},
	}

	result, err := service.Build(
		symbols,
		map[string]struct{}{
			"USDT": {},
		},
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(result.Routes) != 0 {
		t.Fatalf(
			"expected 0 routes, got %d",
			len(result.Routes),
		)
	}

	if len(result.RouteIndex) != 0 {
		t.Fatalf(
			"expected empty route index, got %d symbols",
			len(result.RouteIndex),
		)
	}
}

func TestMarketGraphServiceBuildStartAssets(t *testing.T) {
	service := application.NewMarketGraphService()

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

	result, err := service.Build(
		symbols,
		map[string]struct{}{
			"BTC": {},
		},
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(result.Routes) != 2 {
		t.Fatalf(
			"expected 2 routes, got %d",
			len(result.Routes),
		)
	}

	for _, route := range result.Routes {
		if len(route.Edges) != 3 {
			t.Fatalf(
				"expected triangle route, got %d edges",
				len(route.Edges),
			)
		}

		if route.Edges[0].From != "BTC" {
			t.Fatalf(
				"expected route to start from BTC, got %q",
				route.Edges[0].From,
			)
		}
	}
}