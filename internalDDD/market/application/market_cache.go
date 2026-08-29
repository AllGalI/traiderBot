package application

import (
	"context"
	"traiderBot/internalDDD/market/domain"
)

type MarketCache interface {
	Get(ctx context.Context) (domain.MarketStructure, bool, error)

	Set(ctx context.Context, structure domain.MarketStructure) error
}