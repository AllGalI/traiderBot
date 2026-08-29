package tests

import (
	"testing"

	"traiderBot/internalDDD/market/domain"
	"traiderBot/internalDDD/market/infrastructure"
)

func TestInMemoryMarketStructureStoreGetBeforeSet(t *testing.T) {
	store := infrastructure.NewInMemoryMarketStructureStore()
	structure, ok := store.Get()

	if ok {
		t.Fatal("expected store to report that structure does not exist")
	}

	if structure == nil {
		t.Fatal("expected non-nil structure")
	}

	if len(structure.Symbols) != 0 {
		t.Fatalf(
			"expected empty structure, got %d symbols",
			len(structure.Symbols),
		)
	}
}

func TestInMemoryMarketStructureStoreSetAndGet(t *testing.T) {
	store := infrastructure.NewInMemoryMarketStructureStore()

	expected := &domain.MarketStructure{
		Symbols: []domain.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
			},
		},
	}

	store.Set(expected)

	got, ok := store.Get()

	if !ok {
		t.Fatal("expected structure to exist")
	}

	if got == nil {
		t.Fatal("expected non-nil structure")
	}

	if len(got.Symbols) != 1 {
		t.Fatalf(
			"expected 1 symbol, got %d",
			len(got.Symbols),
		)
	}

	if got.Symbols[0].Name != "BTCUSDT" {
		t.Fatalf(
			"expected BTCUSDT, got %s",
			got.Symbols[0].Name,
		)
	}
}

func TestInMemoryMarketStructureStoreSetReplacesValue(t *testing.T) {
	store := infrastructure.NewInMemoryMarketStructureStore()

	first := domain.MarketStructure{
		Symbols: []domain.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
			},
		},
	}

	second := domain.MarketStructure{
		Symbols: []domain.Symbol{
			{
				Name:       "ETHUSDT",
				BaseAsset:  "ETH",
				QuoteAsset: "USDT",
			},
		},
	}

	store.Set(&first)
	store.Set(&second)

	got, ok := store.Get()

	if !ok {
		t.Fatal("expected store hit")
	}

	if got == nil {
		t.Fatal("expected non-nil structure")
	}

	if len(got.Symbols) != 1 {
		t.Fatalf(
			"expected 1 symbol, got %d",
			len(got.Symbols),
		)
	}

	if got.Symbols[0].Name != "ETHUSDT" {
		t.Fatalf(
			"symbol name = %q, want ETHUSDT",
			got.Symbols[0].Name,
		)
	}
}
