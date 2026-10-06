package domain

import (
	"fmt"

	marketApplication "traiderBot/internalDDD/market/application"
	marketDomain "traiderBot/internalDDD/market/domain"
)

type GraphProvider struct {
	structureStore marketApplication.MarketStructureStore
}

func NewGraphProvider(
	structureStore marketApplication.MarketStructureStore,
) *GraphProvider {
	return &GraphProvider{
		structureStore: structureStore,
	}
}

func (p *GraphProvider) Name() string {
	return "GraphProvider"
}

func (p *GraphProvider) Snapshot() (any, error) {
	structure, ok := p.structureStore.Get()

	if !ok {
		return nil, fmt.Errorf(
			"market structure is not available",
		)
	}

	return map[string]any{
		"assets": structure.Graph,
	}, nil
}

func (p *GraphProvider) Metrics() (any, error) {
	structure, ok := p.structureStore.Get()

	if !ok {
		return nil, fmt.Errorf(
			"market structure is not available",
		)
	}

	return graphMetrics(&structure.Graph), nil
}

func graphMetrics(
	graph *marketDomain.Graph,
) map[string]int {
	edgesCount := 0

	for _, edges := range *graph {
		edgesCount += len(edges)
	}

	return map[string]int{
		"edgesCount":  edgesCount,
		"assetsCount": len(*graph),
	}
}
