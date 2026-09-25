package application

import (
	"context"
	"fmt"

	"traiderBot/internalDDD/market/domain"
)

type MarketStructureStore interface {
	Get() (*domain.MarketStructure, bool)
	Set(*domain.MarketStructure)
}

type MarketStructureService struct {
	symbolProvider SymbolProvider
	symbolsLimit   int
}

func NewMarketStructureService(
	symbolProvider SymbolProvider,
	symbolsLimit int,

) *MarketStructureService {
	return &MarketStructureService{
		symbolProvider: symbolProvider,
		symbolsLimit:   symbolsLimit,
	}
}

func (s *MarketStructureService) Build(
	ctx context.Context,
	startAssets map[string]struct{},

) (domain.MarketStructure, error) {
	symbols, err := s.symbolProvider.GetSymbols(ctx)
	if err != nil {
		return domain.MarketStructure{}, fmt.Errorf(
			"load symbols: %w",
			err,
		)
	}

	if s.symbolsLimit <= 0 {
		return domain.MarketStructure{}, fmt.Errorf(
			"symbols limit must be greater than zero",
			err,
		)
	}

	if s.symbolsLimit < len(symbols) {
		symbols = symbols[:s.symbolsLimit]
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		return domain.MarketStructure{}, fmt.Errorf(
			"build graph: %w",
			err,
		)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		startAssets,
	)

	routeIndex := domain.BuildRouteIndex(routes)

	return domain.MarketStructure{
		Symbols:    symbols,
		Graph:      graph,
		Routes:     routes,
		RouteIndex: routeIndex,
	}, nil
}
