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
}

func NewMarketStructureService(
	symbolProvider SymbolProvider,
) *MarketStructureService {
	return &MarketStructureService{
		symbolProvider: symbolProvider,
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

	fmt.Println(routes[0].Key)

	routeIndex := domain.BuildRouteIndex(routes)

	return domain.MarketStructure{
		Symbols:    symbols,
		Graph:      graph,
		Routes:     routes,
		RouteIndex: routeIndex,
	}, nil
}
