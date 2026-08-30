package binance

import (
	"context"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
)

type APIClient interface {
	Execute(
		ctx context.Context,
		request executionDomain.OrderRequest,
	) (executionDomain.OrderExecutionResult, error)

	GetOrder(
		ctx context.Context,
		symbol string,
		orderID string,
	) (executionDomain.OrderExecutionResult, error)
}

type Adapter struct {
	client APIClient
}

var (
	_ executionApplication.ExchangeExecutor   = (*Adapter)(nil)
	_ executionApplication.ExchangeOrderQuery = (*Adapter)(nil)
)

func NewAdapter(client APIClient) (*Adapter, error) {
	if client == nil {
		return nil, ErrClientNotConfigured
	}

	return &Adapter{
		client: client,
	}, nil
}

func (a *Adapter) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	return a.client.Execute(
		ctx,
		request,
	)
}

func (a *Adapter) GetOrder(
	ctx context.Context,
	symbol string,
	orderID string,
) (executionDomain.OrderExecutionResult, error) {
	return a.client.GetOrder(
		ctx,
		symbol,
		orderID,
	)
}
