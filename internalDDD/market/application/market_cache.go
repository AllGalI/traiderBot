package application

import (
	"context"
)

type MarketCache interface {
	Get(ctx context.Context) (MarketStructure, bool, error)
	Set(ctx context.Context, structure MarketStructure) error
}