package binance

import (
	"context"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
)

type APIClient interface {
	Execute(
		ctx context.Context,
		request exchangeDomain.OrderRequest,
	) (exchangeDomain.OrderResult, error)

	GetOrderByClientID(
		ctx context.Context,
		request exchangeDomain.OrderRequest,
	) (exchangeDomain.OrderResult, error)
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
	request exchangeDomain.OrderRequest,
) (exchangeDomain.OrderResult, error) {
	return a.client.Execute(
		ctx,
		request,
	)
}

func (a *Adapter) GetOrderByClientID(
	ctx context.Context,
	request exchangeDomain.OrderRequest,
) (exchangeDomain.OrderResult, error) {
	return a.client.GetOrderByClientID(
		ctx,
		request,
	)
}