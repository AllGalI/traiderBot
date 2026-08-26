package application

import (
	"testing"

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

	service := NewMarketGraphService()

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