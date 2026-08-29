package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestSimulateStep_Buy_FullFill(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.001"),
			},
			{
				Price: decimal.NewFromInt(100100),
				Qty:   decimal.RequireFromString("0.002"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Buy,
	}

	constraints := marketDomain.SymbolConstraints{}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.NewFromInt(150),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"BTC",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedAmountOut := decimal.RequireFromString(
		"0.0014995004995005",
	)

	if !result.AmountOut.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected amount out %s, got %s",
			expectedAmountOut,
			result.AmountOut,
		)
	}

	expectedFillRatio := decimal.NewFromInt(1)

	if !result.FillRatio.Equal(expectedFillRatio) {
		t.Fatalf(
			"expected fill ratio %s, got %s",
			expectedFillRatio,
			result.FillRatio,
		)
	}

	expectedAmountIn := decimal.NewFromInt(150)

	if !result.AmountIn.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected amount in %s, got %s",
			expectedAmountIn,
			result.AmountIn,
		)
	}
}

func TestSimulateStep_Sell_PartialFill_Accepted(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.0005"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "BTC",
		To:        "USDT",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Sell,
	}

	constraints := marketDomain.SymbolConstraints{}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.RequireFromString("0.001"),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"USDT",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedAmountOut := decimal.RequireFromString("50")

	if !result.AmountOut.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected amount out %s, got %s",
			expectedAmountOut,
			result.AmountOut,
		)
	}

	expectedAmountIn := decimal.RequireFromString("0.0005")

	if !result.AmountIn.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected amount in %s, got %s",
			expectedAmountIn,
			result.AmountIn,
		)
	}

	expectedFillRatio := decimal.RequireFromString("0.5")

	if !result.FillRatio.Equal(expectedFillRatio) {
		t.Fatalf(
			"expected fill ratio %s, got %s",
			expectedFillRatio,
			result.FillRatio,
		)
	}
}

func TestSimulateStep_Sell_PartialFill_Rejected(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.0002"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "BTC",
		To:        "USDT",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Sell,
	}

	constraints := marketDomain.SymbolConstraints{}

	_, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.RequireFromString("0.001"),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"USDT",
	)

	if err == nil {
		t.Fatal("expected insufficient liquidity error")
	}
}

func TestSimulateStep_Buy_PartialFill_Accepted(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.0005"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Buy,
	}

	constraints := marketDomain.SymbolConstraints{}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.NewFromInt(100),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"BTC",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedAmountIn := decimal.NewFromInt(50)

	if !result.AmountIn.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected amount in %s, got %s",
			expectedAmountIn,
			result.AmountIn,
		)
	}

	expectedAmountOut := decimal.RequireFromString("0.0005")

	if !result.AmountOut.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected amount out %s, got %s",
			expectedAmountOut,
			result.AmountOut,
		)
	}

	expectedFillRatio := decimal.RequireFromString("0.5")

	if !result.FillRatio.Equal(expectedFillRatio) {
		t.Fatalf(
			"expected fill ratio %s, got %s",
			expectedFillRatio,
			result.FillRatio,
		)
	}
}