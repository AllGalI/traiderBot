package tests

import (
	"context"
	"testing"
	"time"
	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

func TestExecutionManagerReconcilesOrderAfterExecutorError(t *testing.T) {
	opportunity := simulationDomain.Opportunity{
		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(100),
		ExecutedStartAmount:  decimal.NewFromInt(100),

		Steps: []simulationDomain.StepResult{
			{
				AmountIn:  decimal.NewFromInt(100),
				AmountOut: decimal.NewFromInt(101),
				Edge: marketDomain.GraphEdge{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: marketDomain.Buy,
				},
			},
		},
	}

	executor := &fakeExchangeExecutorError{
		err: context.DeadlineExceeded,
	}

	query := &fakeExchangeOrderQuery{
		result: executionDomain.OrderExecutionResult{
			StepIndex: 0,

			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,

			ClientOrderID:   "execution-1-order-0-2",
			ExchangeOrderID: "exchange-123",

			RequestedAmount:   decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.NewFromInt(101),
			ExecutedAmount:    decimal.NewFromInt(100),
			ReceivedAmount:    decimal.NewFromInt(101),

			Status: executionDomain.OrderFilled,

			CreatedAt:   time.Now(),
			SubmittedAt: time.Now(),
			FilledAt:    time.Now(),
		},
	}

	manager := executionApplication.NewExecutionManager(
		executor,
		query,
		fakeExecutionPolicy{},
	)

	execution, err := manager.Execute(
		context.Background(),
		&opportunity,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.calls != 1 {
		t.Fatalf(
			"expected 1 order query call, got %d",
			query.calls,
		)
	}

	if execution.Status != executionDomain.ExecutionCompleted {
		t.Fatalf(
			"expected completed execution, got %s",
			execution.Status,
		)
	}

	if len(execution.Orders) != 1 {
		t.Fatalf(
			"expected 1 order, got %d",
			len(execution.Orders),
		)
	}

	order := execution.Orders[0]

	if order.ExchangeOrderID != "exchange-123" {
		t.Fatalf(
			"expected exchange order ID exchange-123, got %s",
			order.ExchangeOrderID,
		)
	}

	if order.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected filled order, got %s",
			order.Status,
		)
	}
}

func TestExecutionManagerUsesStableClientOrderIDForReconciliation(t *testing.T) {
	opportunity := simulationDomain.Opportunity{
		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(100),
		ExecutedStartAmount:  decimal.NewFromInt(100),

		Steps: []simulationDomain.StepResult{
			{
				AmountIn:  decimal.NewFromInt(100),
				AmountOut: decimal.NewFromInt(101),
				Edge: marketDomain.GraphEdge{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: marketDomain.Buy,
				},
			},
		},
	}

	executor := &fakeExchangeExecutorError{
		err: context.DeadlineExceeded,
	}

	query := &fakeExecutionOrderQueryWithRequest{
		result: executionDomain.OrderExecutionResult{
			StepIndex:         0,
			Symbol:            "BTCUSDT",
			Direction:         marketDomain.Buy,
			ClientOrderID:     "execution-1-order-0-2",
			ExchangeOrderID:   "exchange-123",
			RequestedAmount:   decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.NewFromInt(101),
			ExecutedAmount:    decimal.NewFromInt(100),
			ReceivedAmount:    decimal.NewFromInt(101),
			Status:            executionDomain.OrderFilled,
			CreatedAt:         time.Now(),
			SubmittedAt:       time.Now(),
			FilledAt:          time.Now(),
		},
	}

	manager := executionApplication.NewExecutionManager(
		executor,
		query,
		fakeExecutionPolicy{},
	)

	execution, err := manager.Execute(
		context.Background(),
		&opportunity,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.calls != 1 {
		t.Fatalf(
			"expected 1 order query call, got %d",
			query.calls,
		)
	}

	if query.symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			query.symbol,
		)
	}

	if query.orderID == "" {
		t.Fatal("expected client order ID to be passed to reconciliation")
	}

	if query.orderID != execution.Orders[0].ClientOrderID {
		t.Fatalf(
			"expected query order ID %s, got %s",
			execution.Orders[0].ClientOrderID,
			query.orderID,
		)
	}
}
