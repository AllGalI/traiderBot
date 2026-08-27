package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"traiderBot/internalDDD/market/domain"
)

func TestMarketCacheGetNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "market.json")

	cache := NewMarketCache(path)

	structure, ok, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if ok {
		t.Fatal("expected cache miss")
	}

	if len(structure.Symbols) != 0 {
		t.Fatal("expected empty structure")
	}
}

func TestMarketCacheSetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "market.json")

	cache := NewMarketCache(path)

	expected := domain.MarketStructure{
		Symbols: []domain.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
				Constraints: domain.SymbolConstraints{
					MinQty:      decimal.RequireFromString("0.00001"),
					MaxQty:      decimal.RequireFromString("9000"),
					StepSize:    decimal.RequireFromString("0.00001"),
					MinNotional: decimal.RequireFromString("10"),
					PriceStep:   decimal.RequireFromString("0.01"),
				},
			},
		},
		Graph: domain.Graph{
			"USDT": {
				{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: domain.Buy,
				},
			},
		},
		Routes: []domain.Route{
			{
				Edges: []domain.GraphEdge{
					{
						From:      "USDT",
						To:        "BTC",
						Symbol:    "BTCUSDT",
						Direction: domain.Buy,
					},
				},
			},
		},
		RouteIndex: domain.RouteIndex{
			"BTCUSDT": {
				{
					Edges: []domain.GraphEdge{
						{
							From:      "USDT",
							To:        "BTC",
							Symbol:    "BTCUSDT",
							Direction: domain.Buy,
						},
					},
				},
			},
		},
	}

	if err := cache.Set(context.Background(), expected); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	actual, ok, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !ok {
		t.Fatal("expected cache hit")
	}

	if len(actual.Symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(actual.Symbols))
	}

	symbol := actual.Symbols[0]

	if symbol.Name != "BTCUSDT" {
		t.Errorf("symbol name = %q, want BTCUSDT", symbol.Name)
	}

	if !symbol.Constraints.StepSize.Equal(
		decimal.RequireFromString("0.00001"),
	) {
		t.Errorf(
			"step size = %s, want 0.00001",
			symbol.Constraints.StepSize,
		)
	}

	if len(actual.Graph["USDT"]) != 1 {
		t.Fatalf(
			"expected 1 graph edge from USDT, got %d",
			len(actual.Graph["USDT"]),
		)
	}

	if len(actual.Routes) != 1 {
		t.Fatalf(
			"expected 1 route, got %d",
			len(actual.Routes),
		)
	}

	if len(actual.RouteIndex["BTCUSDT"]) != 1 {
		t.Fatalf(
			"expected 1 route in BTCUSDT index, got %d",
			len(actual.RouteIndex["BTCUSDT"]),
		)
	}
}

func TestMarketCacheGetInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "market.json")

	if err := writeInvalidCacheFile(path); err != nil {
		t.Fatalf("write invalid cache: %v", err)
	}

	cache := NewMarketCache(path)

	_, ok, err := cache.Get(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	if ok {
		t.Fatal("expected cache miss on invalid cache")
	}
}

func writeInvalidCacheFile(path string) error {
	return os.WriteFile(
		path,
		[]byte("{invalid json"),
		0644,
	)
}
