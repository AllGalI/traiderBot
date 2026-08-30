package tests

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestExecutionManagerAbortsWhenPolicyRejectsExecution(t *testing.T) {
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

	executor := &fakeExchangeExecutor{
		results: []decimal.Decimal{
			decimal.NewFromInt(101),
		},
	}

	manager := executionApplication.NewExecutionManager(
		executor,
		nil,
		fakeExecutionPolicyAbort{},
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

	if execution.Status != executionDomain.ExecutionAborted {
		t.Fatalf(
			"expected status %s, got %s",
			executionDomain.ExecutionAborted,
			execution.Status,
		)
	}

	if len(executor.amountsIn) != 0 {
		t.Fatalf(
			"expected 0 executor calls, got %d",
			len(executor.amountsIn),
		)
	}
}