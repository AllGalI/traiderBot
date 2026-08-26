package binance

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMapOrderBook(t *testing.T) {
	tests := []struct {
		name      string
		dto       orderBookDTO
		symbol    string
		wantError bool
	}{
		{
			name: "valid order book",
			dto: orderBookDTO{
				LastUpdateID: 12345,
				Bids: [][]string{
					{"100.10", "2.5"},
					{"100.00", "1.2"},
				},
				Asks: [][]string{
					{"100.20", "1.5"},
					{"100.30", "3.0"},
				},
			},
			symbol: "BTCUSDT",
		},
		{
			name: "invalid bid price",
			dto: orderBookDTO{
				LastUpdateID: 12345,
				Bids: [][]string{
					{"invalid", "2.5"},
				},
				Asks: [][]string{
					{"100.20", "1.5"},
				},
			},
			symbol:    "BTCUSDT",
			wantError: true,
		},
		{
			name: "invalid ask quantity",
			dto: orderBookDTO{
				LastUpdateID: 12345,
				Bids: [][]string{
					{"100.10", "2.5"},
				},
				Asks: [][]string{
					{"100.20", "invalid"},
				},
			},
			symbol:    "BTCUSDT",
			wantError: true,
		},
		{
			name: "invalid price level length",
			dto: orderBookDTO{
				LastUpdateID: 12345,
				Bids: [][]string{
					{"100.10"},
				},
			},
			symbol:    "BTCUSDT",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapOrderBook(tt.dto, tt.symbol)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Symbol != tt.symbol {
				t.Errorf(
					"Symbol = %q, want %q",
					got.Symbol,
					tt.symbol,
				)
			}

			if got.LastUpdateID != tt.dto.LastUpdateID {
				t.Errorf(
					"LastUpdateID = %d, want %d",
					got.LastUpdateID,
					tt.dto.LastUpdateID,
				)
			}

			if len(got.Bids) != 2 {
				t.Fatalf(
					"len(Bids) = %d, want 2",
					len(got.Bids),
				)
			}

			if len(got.Asks) != 2 {
				t.Fatalf(
					"len(Asks) = %d, want 2",
					len(got.Asks),
				)
			}

			wantBidPrice := decimal.RequireFromString("100.10")
			wantBidQty := decimal.RequireFromString("2.5")

			if !got.Bids[0].Price.Equal(wantBidPrice) {
				t.Errorf(
					"Bid price = %s, want %s",
					got.Bids[0].Price,
					wantBidPrice,
				)
			}

			if !got.Bids[0].Qty.Equal(wantBidQty) {
				t.Errorf(
					"Bid qty = %s, want %s",
					got.Bids[0].Qty,
					wantBidQty,
				)
			}

			wantAskPrice := decimal.RequireFromString("100.20")
			wantAskQty := decimal.RequireFromString("1.5")

			if !got.Asks[0].Price.Equal(wantAskPrice) {
				t.Errorf(
					"Ask price = %s, want %s",
					got.Asks[0].Price,
					wantAskPrice,
				)
			}

			if !got.Asks[0].Qty.Equal(wantAskQty) {
				t.Errorf(
					"Ask qty = %s, want %s",
					got.Asks[0].Qty,
					wantAskQty,
				)
			}
		})
	}
}
