package domain

import "testing"

func TestNewGraph(t *testing.T) {
	symbols := []Symbol{
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
	}

	graph, err := NewGraph(symbols)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}

	t.Run("USDT -> BTC", func(t *testing.T) {
		edges := graph["USDT"]

		if len(edges) != 1 {
			t.Fatalf("expected 1 edge, got %d", len(edges))
		}

		edge := edges[0]

		if edge.From != "USDT" {
			t.Errorf("From = %q, want USDT", edge.From)
		}

		if edge.To != "BTC" {
			t.Errorf("To = %q, want BTC", edge.To)
		}

		if edge.Symbol != "BTCUSDT" {
			t.Errorf("Symbol = %q, want BTCUSDT", edge.Symbol)
		}

		if edge.Direction != Buy {
			t.Errorf("Direction = %v, want Buy", edge.Direction)
		}
	})

	t.Run("BTC -> USDT", func(t *testing.T) {
		var found bool

		for _, edge := range graph["BTC"] {
			if edge.Symbol == "BTCUSDT" &&
				edge.From == "BTC" &&
				edge.To == "USDT" &&
				edge.Direction == Sell {
				found = true
				break
			}
		}

		if !found {
			t.Fatal("BTC -> USDT edge not found")
		}
	})

	t.Run("BTC -> ETH", func(t *testing.T) {
		var found bool

		for _, edge := range graph["BTC"] {
			if edge.Symbol == "ETHBTC" &&
				edge.From == "BTC" &&
				edge.To == "ETH" &&
				edge.Direction == Buy {
				found = true
				break
			}
		}

		if !found {
			t.Fatal("BTC -> ETH edge not found")
		}
	})
}

func TestNewGraphRejectsInvalidSymbol(t *testing.T) {
	_, err := NewGraph([]Symbol{
		{
			Name:       "BTCUSDT",
			BaseAsset:  "BTC",
			QuoteAsset: "",
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}
}
