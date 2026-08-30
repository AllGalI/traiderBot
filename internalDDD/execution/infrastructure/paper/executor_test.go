package paper_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	paper "traiderBot/internalDDD/execution/infrastructure/paper"
)

func TestExecutorReturnsPaperExecutionAmount(t *testing.T) {
	request := executionDomain.OrderRequest{
		StepIndex: 0,

		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},

		Amount:            decimal.NewFromInt(100),
		ExpectedAmountOut: decimal.NewFromFloat(0.0018),

		ClientOrderID: "test-order-1",
	}

	executor := paper.NewExecutor()

	order, err := executor.Execute(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedAmountOut := decimal.NewFromFloat(0.0018)
	expectedAmountIn := decimal.NewFromInt(100)

	if !order.RequestedAmount.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected requested amount %s, got %s",
			expectedAmountIn,
			order.RequestedAmount,
		)
	}

	if !order.ExecutedAmount.Equal(expectedAmountIn) {
		t.Fatalf(
			"expected executed amount %s, got %s",
			expectedAmountIn,
			order.ExecutedAmount,
		)
	}

	if !order.ReceivedAmount.Equal(expectedAmountOut) {
		t.Fatalf(
			"expected received amount %s, got %s",
			expectedAmountOut,
			order.ReceivedAmount,
		)
	}

	if order.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			order.Symbol,
		)
	}

	if order.Direction != marketDomain.Buy {
		t.Fatalf(
			"expected direction %v, got %v",
			marketDomain.Buy,
			order.Direction,
		)
	}

	if order.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected filled order, got %s",
			order.Status,
		)
	}

	if order.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}

	if order.SubmittedAt.IsZero() {
		t.Fatal("SubmittedAt is zero")
	}

	if order.FilledAt.IsZero() {
		t.Fatal("FilledAt is zero")
	}
}

func TestExecutorRejectsEmptySymbol(t *testing.T) {
	executor := paper.NewExecutor()

	_, err := executor.Execute(
		context.Background(),
		executionDomain.OrderRequest{
			Amount:            decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.NewFromInt(101),
		},
	)

	if err == nil {
		t.Fatal("expected error for empty symbol")
	}
}

func TestExecutorRejectsInvalidExpectedAmountOut(t *testing.T) {
	executor := paper.NewExecutor()

	_, err := executor.Execute(
		context.Background(),
		executionDomain.OrderRequest{
			Edge: marketDomain.GraphEdge{
				Symbol: "BTCUSDT",
			},
			Amount:            decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.Zero,
		},
	)

	if err == nil {
		t.Fatal("expected error for invalid expected amount out")
	}
}

func TestExecutorRejectsInvalidAmountIn(t *testing.T) {
	executor := paper.NewExecutor()

	_, err := executor.Execute(
		context.Background(),
		executionDomain.OrderRequest{
			Edge: marketDomain.GraphEdge{
				Symbol: "BTCUSDT",
			},
			Amount:            decimal.Zero,
			ExpectedAmountOut: decimal.NewFromFloat(0.0018),
		},
	)

	if err == nil {
		t.Fatal("expected error for zero amount in")
	}
}

func TestExecutorStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	executor := paper.NewExecutor()

	_, err := executor.Execute(
		ctx,
		executionDomain.OrderRequest{
			Edge: marketDomain.GraphEdge{
				Symbol: "BTCUSDT",
			},
			Amount:            decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.NewFromFloat(0.0018),
		},
	)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if err != context.Canceled {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}
