package em

import (
	"context"
	"fmt"
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"
	recoveryApplication "traiderBot/internalDDD/recovery/application"

	"github.com/shopspring/decimal"
)

func (m *ExecutionManager) executeStep(
	ctx context.Context,
	step executionDomain.ExecutionStep,
	requestAmount decimal.Decimal,
	marketContext recoveryApplication.MarketContext,
) (
	executionDomain.ActualStepResult,
	executionDomain.OrderResult,
	error,
) {
	requestAmount, err := m.normalizeOrderAmount(
		requestAmount,
		step.Edge,
		marketContext,
	)
	if err != nil {
		return executionDomain.ActualStepResult{},
			executionDomain.OrderResult{},
			fmt.Errorf(
				"normalize amount for step %d: %w",
				step.ID,
				err,
			)
	}

	request, err := m.buildOrderRequest(
		step,
		requestAmount,
	)
	if err != nil {
		return executionDomain.ActualStepResult{},
			executionDomain.OrderResult{},
			err
	}

	startedAt := time.Now()

	result, err := m.exchangeAdapter.Execute(
		ctx,
		request,
	)
	if err != nil {
		return executionDomain.ActualStepResult{},
			executionDomain.OrderResult{},
			fmt.Errorf(
				"execute step %d: %w",
				step.ID,
				err,
			)
	}

	completedAt := time.Now()

	if err := m.validateOrderResult(result); err != nil {
		return executionDomain.ActualStepResult{},
			executionDomain.OrderResult{},
			fmt.Errorf(
				"step %d returned invalid order result: %w",
				step.ID,
				err,
			)
	}

	result, completedAt, err = m.reconcileIfRequired(
		ctx,
		step,
		request,
		result,
		completedAt,
	)
	if err != nil {
		return executionDomain.ActualStepResult{},
			executionDomain.OrderResult{},
			err
	}

	actualResult, err := m.resultMapper.Map(
		step,
		request,
		result,
		startedAt,
		completedAt,
	)
	if err != nil {
		return executionDomain.ActualStepResult{},
			executionDomain.OrderResult{},
			fmt.Errorf(
				"map actual result for step %d: %w",
				step.ID,
				err,
			)
	}

	return actualResult, result, nil
}

func (m *ExecutionManager) buildOrderRequest(
	step executionDomain.ExecutionStep,
	amount decimal.Decimal,
) (executionDomain.OrderRequest, error) {
	clientOrderID := m.orderIDGenerator.Next(step.ID)

	return executionDomain.NewOrderRequest(
		step.Edge,
		amount,
		clientOrderID,
	)
}
