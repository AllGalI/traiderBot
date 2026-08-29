package integration

import (
	"testing"

	marketDomain "traiderBot/internalDDD/market/domain"
	marketInfrastructure "traiderBot/internalDDD/market/infrastructure"
	simulationApplication "traiderBot/internalDDD/simulation/application"
)

func TestMarketSimulationMarketStructureSymbolProvider(t *testing.T) {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	structure := &marketDomain.MarketStructure{
		Symbols: []marketDomain.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
			},
			{
				Name:       "ETHUSDT",
				BaseAsset:  "ETH",
				QuoteAsset: "USDT",
			},
		},
	}

	store.Set(structure)

	provider := simulationApplication.NewMarketStructureSymbolProvider(store)

	t.Run("existing symbol", func(t *testing.T) {
		got, ok := provider.Get("BTCUSDT")

		if !ok {
			t.Fatal("expected symbol to be found")
		}

		if got.Name != "BTCUSDT" {
			t.Fatalf(
				"symbol name = %q, want BTCUSDT",
				got.Name,
			)
		}

		if got.BaseAsset != "BTC" {
			t.Fatalf(
				"base asset = %q, want BTC",
				got.BaseAsset,
			)
		}

		if got.QuoteAsset != "USDT" {
			t.Fatalf(
				"quote asset = %q, want USDT",
				got.QuoteAsset,
			)
		}
	})

	t.Run("unknown symbol", func(t *testing.T) {
		_, ok := provider.Get("BNBUSDT")

		if ok {
			t.Fatal("expected unknown symbol to return false")
		}
	})
}

func TestMarketSimulationMarketStructureSymbolProviderWithoutStructure(
	t *testing.T,
) {
	store := marketInfrastructure.NewInMemoryMarketStructureStore()

	provider := simulationApplication.NewMarketStructureSymbolProvider(store)

	_, ok := provider.Get("BTCUSDT")

	if ok {
		t.Fatal("expected lookup to fail when market structure is unavailable")
	}
}