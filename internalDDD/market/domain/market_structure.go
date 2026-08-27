package domain

type MarketStructure struct {
	Symbols    []Symbol
	Graph      Graph
	Routes     []Route
	RouteIndex RouteIndex
}
