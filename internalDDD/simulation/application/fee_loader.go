package application

import "context"

type FeeLoader interface {
	Get(ctx context.Context, symbol string) (FeeConfig, error)
}
