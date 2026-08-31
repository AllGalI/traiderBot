package tests

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestExecutionManagerFailsWhenExecutorReturnsError(t *testing.T) {
	opportunity := simulationDomain.Opportunity{
		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(100),
		ExecutedStartAmount:  decimal.NewFromInt(100),

		Steps: []simulationDomain.StepResult{
			{
				AmountIn:  decimal.NewFromInt(100),
				AmountOut: decimal.NewFromInt(101),
			},
		},
	}

	executor := &fakeExchangeExecutorError{
		err: context.DeadlineExceeded,
	}

	manager := executionApplication.NewExecutionManager(
		executor,
		nil,
		fakeExecutionPolicy{},
	)

	execution, err := manager.Execute(
		context.Background(),
		&opportunity,
	)

	if err == nil {
		t.Fatal("expected execution error, got nil")
	}

	if execution == nil {
		t.Fatal("expected execution, got nil")
	}

	if execution.Status != executionDomain.ExecutionFailed {
		t.Fatalf(
			"expected status %s, got %s",
			executionDomain.ExecutionFailed,
			execution.Status,
		)
	}

	if len(execution.Orders) != 0 {
		t.Fatalf(
			"expected 0 orders after executor failure, got %d",
			len(execution.Orders),
		)
	}
}

func TestExecutionManagerAbortsWhenOrderIsNotFilled(t *testing.T) {
	opportunity := simulationDomain.Opportunity{
		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(100),
		ExecutedStartAmount:  decimal.NewFromInt(100),

		Steps: []simulationDomain.StepResult{
			{
				AmountIn:  decimal.NewFromInt(100),
				AmountOut: decimal.NewFromFloat(0.0018),
			},
			{
				AmountIn:  decimal.NewFromFloat(0.0018),
				AmountOut: decimal.NewFromFloat(4.7),
			},
			{
				AmountIn:  decimal.NewFromFloat(4.7),
				AmountOut: decimal.NewFromFloat(101.2),
			},
		},
	}

	executor := &fakeExchangeExecutor{
		results: []decimal.Decimal{
			decimal.NewFromFloat(0.0018),
			decimal.NewFromFloat(4.7),
			decimal.NewFromFloat(101.2),
		},
		statuses: []executionDomain.OrderStatus{
			executionDomain.OrderFilled,
			executionDomain.OrderCancelled,
			executionDomain.OrderFilled,
		},
	}

	manager := executionApplication.NewExecutionManager(
		executor,
		nil,
		fakeExecutionPolicy{},
	)

	execution, err := manager.Execute(
		context.Background(),
		&opportunity,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.Status != executionDomain.ExecutionAborted {
		t.Fatalf(
			"expected execution aborted, got %s",
			execution.Status,
		)
	}

	if len(executor.amountsIn) != 2 {
		t.Fatalf(
			"expected 2 executor calls, got %d",
			len(executor.amountsIn),
		)
	}

	if len(execution.Orders) != 2 {
		t.Fatalf(
			"expected 2 orders, got %d",
			len(execution.Orders),
		)
	}

	if execution.Orders[0].Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected first order filled, got %s",
			execution.Orders[0].Status,
		)
	}

	if execution.Orders[1].Status != executionDomain.OrderCancelled {
		t.Fatalf(
			"expected second order cancelled, got %s",
			execution.Orders[1].Status,
		)
	}
}

func TestExecutionManagerFailsWhenExecutionAndReconciliationFail(t *testing.T) {
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
		err: context.DeadlineExceeded,
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

	if err == nil {
		t.Fatal("expected execution error, got nil")
	}

	if execution == nil {
		t.Fatal("expected execution, got nil")
	}

	if execution.Status != executionDomain.ExecutionFailed {
		t.Fatalf(
			"expected execution failed, got %s",
			execution.Status,
		)
	}

	if query.calls != 1 {
		t.Fatalf(
			"expected 1 reconciliation call, got %d",
			query.calls,
		)
	}

	if len(execution.Orders) != 0 {
		t.Fatalf(
			"expected 0 orders after failed reconciliation, got %d",
			len(execution.Orders),
		)
	}
}
