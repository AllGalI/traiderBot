package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestSimulateStep_Buy_AppliesStepSize(t *testing.T) {
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

	constraints := marketDomain.SymbolConstraints{
		StepSize: decimal.RequireFromString("0.0001"),
	}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.RequireFromString("55"),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"BTC",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 55 / 100000 = 0.00055 BTC.
	//
	// StepSize = 0.0001.
	//
	// Поэтому фактически покупаем:
	// 0.0005 BTC = 50 USDT.

	expectedQuantity := decimal.RequireFromString("0.0005")
	expectedAmountIn := decimal.NewFromInt(50)
	expectedAmountOut := decimal.RequireFromString("0.0005")

	if !result.Quantity.Equal(expectedQuantity) {
		t.Fatalf(
			"expected quantity %s, got %s",
			expectedQuantity,
			result.Quantity,
		)
	}

	if !result.AmountIn.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected amount in %s, got %s",
			expectedAmountIn,
			result.AmountIn,
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

func TestSimulateStep_Sell_AppliesStepSize(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("1"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "BTC",
		To:        "USDT",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Sell,
	}

	constraints := marketDomain.SymbolConstraints{
		StepSize: decimal.RequireFromString("0.0001"),
	}

	result, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.RequireFromString("0.00055"),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"USDT",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedQuantity := decimal.RequireFromString("0.0005")
	expectedAmountIn := decimal.RequireFromString("0.0005")
	expectedAmountOut := decimal.NewFromInt(50)

	if !result.Quantity.Equal(expectedQuantity) {
		t.Fatalf(
			"expected quantity %s, got %s",
			expectedQuantity,
			result.Quantity,
		)
	}

	if !result.AmountIn.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected amount in %s, got %s",
			expectedAmountIn,
			result.AmountIn,
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

func TestSimulateStep_RejectsBelowMinQty(t *testing.T) {
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

	constraints := marketDomain.SymbolConstraints{
		MinQty: decimal.RequireFromString("0.001"),
	}

	_, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.NewFromInt(50),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"BTC",
	)

	if err == nil {
		t.Fatal("expected error for quantity below minimum")
	}
}

func TestSimulateStep_RejectsAboveMaxQty(t *testing.T) {
	orderBook := marketDomain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []marketDomain.PriceLevel{
			{
				Price: decimal.NewFromInt(100000),
				Qty:   decimal.RequireFromString("0.01"),
			},
		},
	}

	edge := marketDomain.GraphEdge{
		From:      "USDT",
		To:        "BTC",
		Symbol:    "BTCUSDT",
		Direction: marketDomain.Buy,
	}

	constraints := marketDomain.SymbolConstraints{
		MaxQty: decimal.RequireFromString("0.0005"),
	}

	_, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.NewFromInt(100),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"BTC",
	)

	if err == nil {
		t.Fatal("expected error for quantity above maximum")
	}
}

func TestSimulateStep_RejectsBelowMinNotional(t *testing.T) {
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

	constraints := marketDomain.SymbolConstraints{
		MinNotional: decimal.NewFromInt(100),
	}

	_, err := simulationDomain.SimulateStep(
		edge,
		orderBook,
		constraints,
		decimal.NewFromInt(50),
		decimal.RequireFromString("0.30"),
		decimal.Zero,
		"BTC",
	)

	if err == nil {
		t.Fatal("expected error for notional below minimum")
	}
}

func TestSimulateStep_AcceptsExactMinQty(t *testing.T) {
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

	constraints := marketDomain.SymbolConstraints{
		MinQty: decimal.RequireFromString("0.001"),
	}

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
		t.Fatalf("unexpected error at exact minimum quantity: %v", err)
	}

	expectedQuantity := decimal.RequireFromString("0.001")

	if !result.Quantity.Equal(expectedQuantity) {
		t.Fatalf(
			"expected quantity %s, got %s",
			expectedQuantity,
			result.Quantity,
		)
	}
}

func TestSimulateStep_AcceptsExactMinNotional(t *testing.T) {
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

	constraints := marketDomain.SymbolConstraints{
		MinNotional: decimal.NewFromInt(100),
	}

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
		t.Fatalf("unexpected error at exact minimum notional: %v", err)
	}

	expectedQuoteQty := decimal.NewFromInt(100)

	if !result.QuoteQty.Equal(expectedQuoteQty) {
		t.Fatalf(
			"expected quote qty %s, got %s",
			expectedQuoteQty,
			result.QuoteQty,
		)
	}
}
