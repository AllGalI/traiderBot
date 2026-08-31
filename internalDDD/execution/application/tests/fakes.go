package tests

import (
	"context"
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"

	"github.com/shopspring/decimal"
)

type fakeExchangeExecutor struct {
	amountsIn  []decimal.Decimal
	amountsOut []decimal.Decimal

	results  []decimal.Decimal
	statuses []executionDomain.OrderStatus

	requests []executionDomain.OrderRequest
}

func (f *fakeExchangeExecutor) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	callIndex := len(f.amountsOut)

	f.amountsIn = append(
		f.amountsIn,
		request.Amount,
	)

	f.requests = append(
		f.requests,
		request,
	)

	result := f.results[callIndex]

	f.amountsOut = append(
		f.amountsOut,
		result,
	)

	status := executionDomain.OrderFilled

	if len(f.statuses) > callIndex {
		status = f.statuses[callIndex]
	}

	now := time.Now()

	return executionDomain.OrderExecutionResult{
		StepIndex: request.StepIndex,

		Symbol:    request.Edge.Symbol,
		Direction: request.Edge.Direction,

		ClientOrderID: request.ClientOrderID,

		RequestedAmount:   request.Amount,
		ExpectedAmountOut: request.ExpectedAmountOut,
		ExecutedAmount:    request.Amount,
		ReceivedAmount:    result,

		Status: status,

		CreatedAt:   now,
		SubmittedAt: now,
		FilledAt:    now,
	}, nil
}

type fakeExchangeExecutorError struct {
	err error
}

func (f *fakeExchangeExecutorError) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	return executionDomain.OrderExecutionResult{}, f.err
}

type fakeExecutionPolicy struct{}

func (fakeExecutionPolicy) Decide(
	execution *executionDomain.Execution,
) executionDomain.ExecutionDecision {
	return executionDomain.ExecutionDecision{
		Type: executionDomain.DecisionContinue,
	}
}

type fakeExecutionPolicyAbort struct{}

func (fakeExecutionPolicyAbort) Decide(
	execution *executionDomain.Execution,
) executionDomain.ExecutionDecision {
	return executionDomain.ExecutionDecision{
		Type:   executionDomain.DecisionAbort,
		Reason: "test abort",
	}
}

type fakeExchangeOrderQuery struct {
	result executionDomain.OrderExecutionResult
	err    error

	calls int
}

func (f *fakeExchangeOrderQuery) GetOrder(
	ctx context.Context,
	symbol string,
	orderID string,
) (executionDomain.OrderExecutionResult, error) {
	f.calls++

	if f.err != nil {
		return executionDomain.OrderExecutionResult{}, f.err
	}

	return f.result, nil
}
