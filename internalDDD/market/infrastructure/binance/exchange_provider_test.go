package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestExchangeProviderGetSymbols(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.URL.Path != exchangeInfoPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, exchangeInfoPath)
		}

		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`{
			"symbols": [
				{
					"symbol": "BTCUSDT",
					"baseAsset": "BTC",
					"quoteAsset": "USDT",
					"status": "TRADING",
					"filters": [
						{
							"filterType": "LOT_SIZE",
							"minQty": "0.00001000",
							"maxQty": "9000.00000000",
							"stepSize": "0.00001000"
						},
						{
							"filterType": "MIN_NOTIONAL",
							"minNotional": "5.00000000"
						},
						{
							"filterType": "PRICE_FILTER",
							"minPrice": "0.01000000",
							"maxPrice": "1000000.00000000",
							"tickSize": "0.01000000"
						}
					]
				},
				{
					"symbol": "ETHUSDT",
					"baseAsset": "ETH",
					"quoteAsset": "USDT",
					"status": "BREAK",
					"filters": []
				},
				{
					"symbol": "BROKENUSDT",
					"baseAsset": "",
					"quoteAsset": "USDT",
					"status": "TRADING",
					"filters": []
				}
			]
		}`))
	}))
	defer server.Close()

	provider := &ExchangeProvider{
		client:  server.Client(),
		baseURL: server.URL,
	}

	symbols, err := provider.GetSymbols(context.Background())
	if err != nil {
		t.Fatalf("GetSymbols() error = %v", err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}

	symbol := symbols[0]

	if symbol.Name != "BTCUSDT" {
		t.Errorf("Name = %q, want BTCUSDT", symbol.Name)
	}

	if symbol.BaseAsset != "BTC" {
		t.Errorf("BaseAsset = %q, want BTC", symbol.BaseAsset)
	}

	if symbol.QuoteAsset != "USDT" {
		t.Errorf("QuoteAsset = %q, want USDT", symbol.QuoteAsset)
	}

	if !symbol.Constraints.StepSize.Equal(decimal.RequireFromString("0.00001000")) {
		t.Errorf(
			"StepSize = %s, want 0.00001000",
			symbol.Constraints.StepSize,
		)
	}

	if !symbol.Constraints.PriceStep.Equal(decimal.RequireFromString("0.01000000")) {
		t.Errorf(
			"PriceStep = %s, want 0.01000000",
			symbol.Constraints.PriceStep,
		)
	}

	if !symbol.Constraints.MinNotional.Equal(
		decimal.RequireFromString("5"),
	) {
		t.Errorf(
			"MinNotional = %s, want 5",
			symbol.Constraints.MinNotional,
		)
	}
}

func TestExchangeProviderGetSymbolsReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := &ExchangeProvider{
		client:  server.Client(),
		baseURL: server.URL,
	}

	_, err := provider.GetSymbols(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
