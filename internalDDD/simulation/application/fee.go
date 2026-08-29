package application

import (
	"context"

	"github.com/shopspring/decimal"
)

type FeeConfig struct {
	Rate decimal.Decimal
}

type FeeProvider interface {
	Get(ctx context.Context, symbol string) (FeeConfig, bool)
}
