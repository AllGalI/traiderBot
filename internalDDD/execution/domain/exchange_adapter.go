package domain

import "context"

type ExchangeAdapter interface {
	Execute(
		ctx context.Context,
		request OrderRequest,
	) (OrderResult, error)

	GetOrderByClientID(
		ctx context.Context,
		request OrderRequest,
	) (OrderResult, error)
}
