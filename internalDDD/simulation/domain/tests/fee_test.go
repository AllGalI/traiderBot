package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestSimulateStep_Buy_WithFee(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.001"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Buy,
	}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		marketDomain.SymbolConstraints{},
		decimal.NewFromInt(100),
		decimal.RequireFromString("0.3"),
		decimal.RequireFromString("0.001"),
		"BTC",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFee := decimal.RequireFromString("0.000001")
	expectedAmountOut := decimal.RequireFromString("0.000999")

	if !result.Fee.Equal(expectedFee) {
		t.Fatalf(
			"expected fee %s, got %s",
			expectedFee,
			result.Fee,
		)
	}

	if result.FeeAsset != "BTC" {
		t.Fatalf(
			"expected fee asset BTC, got %s",
			result.FeeAsset,
		)
	}

	if !result.AmountOut.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected amount out %s, got %s",
			expectedAmountOut,
			result.AmountOut,
		)
	}
}

func TestSimulateStep_Sell_WithFee(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.001"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "BTC",
		To:        "USDT",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Sell,
	}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		marketDomain.SymbolConstraints{},
		decimal.RequireFromString("0.001"),
		decimal.RequireFromString("0.3"),
		decimal.RequireFromString("0.001"),
		"USDT",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFee := decimal.RequireFromString("0.1")
	expectedAmountOut := decimal.RequireFromString("99.9")

	if !result.Fee.Equal(expectedFee) {
		t.Fatalf(
			"expected fee %s, got %s",
			expectedFee,
			result.Fee,
		)
	}

	if result.FeeAsset != "USDT" {
		t.Fatalf(
			"expected fee asset USDT, got %s",
			result.FeeAsset,
		)
	}

	if !result.AmountOut.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected amount out %s, got %s",
			expectedAmountOut,
			result.AmountOut,
		)
	}
}

func TestSimulateStep_ZeroFee(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.001"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Buy,
	}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		marketDomain.SymbolConstraints{},
		decimal.NewFromInt(100),
		decimal.RequireFromString("0.3"),
		decimal.Zero,
		"BTC",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedAmountOut := decimal.RequireFromString("0.001")

	if !result.Fee.IsZero() {
		t.Fatalf(
			"expected zero fee, got %s",
			result.Fee,
		)
	}

	if !result.AmountOut.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected amount out %s, got %s",
			expectedAmountOut,
			result.AmountOut,
		)
	}
}

func TestSimulateStep_RejectsWrongFeeAsset(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.001"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Buy,
	}

	_, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		marketDomain.SymbolConstraints{},
		decimal.NewFromInt(100),
		decimal.RequireFromString("0.3"),
		decimal.RequireFromString("0.001"),
		"USDT",
	)

	if err == nil {
		t.Fatal("expected error for wrong fee asset")
	}
}