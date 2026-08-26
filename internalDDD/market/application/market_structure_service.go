package application

import (
	"context"
	"fmt"

	"traiderBot/internalDDD/market/domain"
)

type MarketStructure struct {
    Symbols    []domain.Symbol
    Graph      domain.Graph
    Routes     []domain.Route
    RouteIndex domain.RouteIndex
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
) (MarketStructure, error) {
	symbols, err := s.symbolProvider.GetSymbols(ctx)
	if err != nil {
		return MarketStructure{}, fmt.Errorf(
			"load symbols: %w",
			err,
		)
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		return MarketStructure{}, fmt.Errorf(
			"build graph: %w",
			err,
		)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		startAssets,
	)

	routeIndex := domain.BuildRouteIndex(routes)

	return MarketStructure{
		Symbols: symbols,
		Graph:   graph,
		Routes:  routes,
		RouteIndex: routeIndex,
	}, nil
}