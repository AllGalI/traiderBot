package application

import (
	"context"

	executionDomain "traiderBot/internalDDD/execution/domain"
)

type ExchangeExecutor interface {
	Execute(
		ctx context.Context,
		request executionDomain.OrderRequest,
	) (executionDomain.OrderExecutionResult, error)
}

type ExchangeOrderQuery interface {
	GetOrder(
		ctx context.Context,
		symbol string,
		orderID string,
	) (executionDomain.OrderExecutionResult, error)
}
