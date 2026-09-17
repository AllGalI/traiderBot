package domain

import "context"

type Adapter interface {
	Execute(ctx context.Context, request OrderRequest) (OrderResult, error)
	GetOrderByClientID(ctx context.Context, request OrderRequest) (OrderResult, error)
}