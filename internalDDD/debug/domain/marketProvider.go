package domain

import "traiderBot/internalDDD/market/domain"

type MarketMetrics struct {
}

type MarketProvider struct {
	marketStructure *domain.MarketStructure
}

func NewMarketProvider(marketStructure *domain.MarketStructure) *MarketProvider {
	return &MarketProvider{
		marketStructure: marketStructure,
	}
}

func (p *MarketProvider) Name() string {
	return "MarketStructureProvider"
}

func (p *MarketProvider) Metrics() map[string]any {
	response := map[string]any{
		"symbolsCount":  len(p.marketStructure.Symbols),
		// "grapthMetrics": graphMetrics(&p.marketStructure.Graph),
		"routeIndex":    routeIndexMetrics(p.marketStructure.RouteIndex),
	}
	return response
}
