package infrastructure

import (
	"testing"

	"github.com/shopspring/decimal"

	"traiderBot/internalDDD/market/domain"
)

func TestOrderBookStore_SetAndGet(t *testing.T) {
	store := NewOrderBookStore()

	orderBook := domain.OrderBook{
		Symbol:       "BTCUSDT",
		LastUpdateID: 123,
		Bids: []domain.PriceLevel{
			{
				Price: decimal.RequireFromString("100000"),
				Qty:   decimal.RequireFromString("0.1"),
			},
		},
		Asks: []domain.PriceLevel{
			{
				Price: decimal.RequireFromString("100001"),
				Qty:   decimal.RequireFromString("0.2"),
			},
		},
	}

	store.Set(orderBook)

	got, ok := store.Get("BTCUSDT")
	if !ok {
		t.Fatal("expected order book to exist")
	}

	if got.Symbol != orderBook.Symbol {
		t.Fatalf(
			"expected symbol %q, got %q",
			orderBook.Symbol,
			got.Symbol,
		)
	}

	if got.LastUpdateID != orderBook.LastUpdateID {
		t.Fatalf(
			"expected update ID %d, got %d",
			orderBook.LastUpdateID,
			got.LastUpdateID,
		)
	}

	if len(got.Bids) != 1 {
		t.Fatalf("expected 1 bid, got %d", len(got.Bids))
	}

	if len(got.Asks) != 1 {
		t.Fatalf("expected 1 ask, got %d", len(got.Asks))
	}
}

func TestOrderBookStore_GetUnknownSymbol(t *testing.T) {
	store := NewOrderBookStore()

	_, ok := store.Get("BTCUSDT")
	if ok {
		t.Fatal("expected order book not to exist")
	}
}

func TestOrderBookStore_SetReplacesOrderBook(t *testing.T) {
	store := NewOrderBookStore()

	first := domain.OrderBook{
		Symbol:       "BTCUSDT",
		LastUpdateID: 100,
	}

	second := domain.OrderBook{
		Symbol:       "BTCUSDT",
		LastUpdateID: 200,
	}

	store.Set(first)
	store.Set(second)

	got, ok := store.Get("BTCUSDT")
	if !ok {
		t.Fatal("expected order book to exist")
	}

	if got.LastUpdateID != 200 {
		t.Fatalf(
			"expected latest update ID 200, got %d",
			got.LastUpdateID,
		)
	}
}

func TestOrderBookStore_SymbolIsCaseInsensitive(t *testing.T) {
	store := NewOrderBookStore()

	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
	}

	store.Set(orderBook)

	got, ok := store.Get("btcusdt")
	if !ok {
		t.Fatal("expected order book to exist")
	}

	if got.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %q",
			got.Symbol,
		)
	}
}