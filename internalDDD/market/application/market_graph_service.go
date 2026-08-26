package application

import "traiderBot/internalDDD/market/domain"

type MarketGraph struct {
	Graph      domain.Graph
	Routes     []domain.Route
	RouteIndex domain.RouteIndex
}

type MarketGraphService struct{}

func NewMarketGraphService() *MarketGraphService {
	return &MarketGraphService{}
}

func (s *MarketGraphService) Build(
	symbols []domain.Symbol,
	startAssets map[string]struct{},
) (MarketGraph, error) {
	graph, err := domain.NewGraph(symbols)
	if err != nil {
		return MarketGraph{}, err
	}

	routes := domain.FindTriangularRoutes(
		graph,
		startAssets,
	)

	routeIndex := domain.BuildRouteIndex(routes)

	return MarketGraph{
		Graph:      graph,
		Routes:     routes,
		RouteIndex: routeIndex,
	}, nil
}
