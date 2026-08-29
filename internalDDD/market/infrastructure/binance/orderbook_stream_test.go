package binance

import (
	"testing"
)

func TestSymbolFromStream(t *testing.T) {
	tests := []struct {
		name    string
		stream  string
		want    string
		wantErr bool
	}{
		{
			name:   "BTCUSDT depth stream",
			stream: "btcusdt@depth10@100ms",
			want:   "BTCUSDT",
		},
		{
			name:   "ETHUSDT depth stream",
			stream: "ethusdt@depth5@100ms",
			want:   "ETHUSDT",
		},
		{
			name:   "stream with additional parts",
			stream: "bnbusdt@depth20@100ms@something",
			want:   "BNBUSDT",
		},
		{
			name:    "empty stream",
			stream:  "",
			wantErr: true,
		},
		{
			name:    "invalid stream",
			stream:  "@depth10@100ms",
			wantErr: true,
		},
		{
			name:    "missing stream separator",
			stream:  "btcusdt",
			wantErr: true,
		},
		{
			name:    "missing symbol",
			stream:  "@",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := symbolFromStream(tt.stream)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf(
					"symbolFromStream() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}

}

func TestBuildWebSocketURL(t *testing.T) {
	tests := []struct {
		name    string
		symbols []string
		depth   int
		want    string
	}{
		{
			name: "multiple symbols",
			symbols: []string{
				"BTCUSDT",
				"ETHUSDT",
			},
			depth: 10,
			want:  "wss://stream.binance.com:9443/stream?streams=btcusdt@depth10@100ms/ethusdt@depth10@100ms",
		},
		{
			name:    "single symbol",
			symbols: []string{"BTCUSDT"},
			depth:   5,
			want:    "wss://stream.binance.com:9443/stream?streams=btcusdt@depth5@100ms",
		},
		{
			name: "lowercase symbols",
			symbols: []string{
				"btcusdt",
				"ethusdt",
			},
			depth: 20,
			want:  "wss://stream.binance.com:9443/stream?streams=btcusdt@depth20@100ms/ethusdt@depth20@100ms",
		},
		{
			name:    "zero depth",
			symbols: []string{"BTCUSDT"},
			depth:   0,
			want:    "wss://stream.binance.com:9443/stream?streams=btcusdt@depth0@100ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWebSocketURL(tt.symbols, tt.depth)

			if got != tt.want {
				t.Errorf(
					"buildWebSocketURL() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}

}

func TestBuildWebSocketURLEmptySymbols(t *testing.T) {
	got := buildWebSocketURL(nil, 10)

	want := "wss://stream.binance.com:9443/stream?streams="

	if got != want {
		t.Errorf(
			"buildWebSocketURL() = %q, want %q",
			got,
			want,
		)
	}

}
