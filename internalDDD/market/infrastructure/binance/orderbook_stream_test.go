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
	got := buildWebSocketURL(
		[]string{
			"BTCUSDT",
			"ETHUSDT",
		},
		10,
	)

	want := "wss://stream.binance.com:9443/stream?streams=btcusdt@depth10@100ms/ethusdt@depth10@100ms"

	if got != want {
		t.Errorf(
			"buildWebSocketURL() = %q, want %q",
			got,
			want,
		)
	}
}