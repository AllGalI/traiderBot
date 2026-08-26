package application

import (
	"context"

	"traiderBot/internalDDD/market/domain"
)

type SymbolProvider interface {
	Get(
		ctx context.Context,
		symbols []string,
	) ([]domain.Symbol, error)

	GetSymbols(ctx context.Context) ([]domain.Symbol, error)
}
