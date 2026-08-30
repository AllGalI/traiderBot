package paper

import (
	"context"
	"fmt"
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"

	"github.com/shopspring/decimal"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	if request.Edge.Symbol == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"symbol must not be empty",
		)
	}

	if request.Amount.LessThanOrEqual(decimal.Zero) {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"amount must be greater than zero",
		)
	}

	if request.ExpectedAmountOut.LessThanOrEqual(decimal.Zero) {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"expected amount out must be greater than zero",
		)
	}

	if request.ClientOrderID == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"client order ID must not be empty",
		)
	}

	now := time.Now()

	return executionDomain.NewOrderExecutionResult(
		request,
		request.Amount,
		request.ExpectedAmountOut,
		executionDomain.OrderFilled,
		request.ClientOrderID,
		request.ClientOrderID,
		now,
		now,
		now,
	), nil
}