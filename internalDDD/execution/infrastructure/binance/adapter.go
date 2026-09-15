package binance

import (
	"context"

	executionDomain "traiderBot/internalDDD/execution/domain"
)

type APIClient interface {
	Execute(
		ctx context.Context,
		request executionDomain.OrderRequest,
	) (executionDomain.OrderResult, error)

	GetOrderByClientID(
		ctx context.Context,
		request executionDomain.OrderRequest,
	) (executionDomain.OrderResult, error)
}

type Adapter struct {
	client APIClient
}


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
) (executionDomain.OrderResult, error) {
	return a.client.Execute(
		ctx,
		request,
	)
}

func (a *Adapter) GetOrderByClientID(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderResult, error) {
	return a.client.GetOrderByClientID(
		ctx,
		request,
	)
}