package domain

import (
	"encoding/json"
	"fmt"

	"traiderBot/internalDDD/market/application"
	marketDomain "traiderBot/internalDDD/market/domain"
)

type GraphProvider struct {
	structureStore application.MarketStructureStore
}

func NewGraphProvider(
	structureStore application.MarketStructureStore,
) *GraphProvider {
	return &GraphProvider{
		structureStore: structureStore,
	}
}

func (p *GraphProvider) Name() string {
	return "GraphProvider"
}

func (p *GraphProvider) Snapshot() (string, error) {
	structure, ok := p.structureStore.Get()
	if !ok {
		return "", fmt.Errorf(
			"market structure is not available",
		)
	}

	data, err := json.MarshalIndent(
		structure.Graph,
		"",
		"    ",
	)
	if err != nil {
		return "", fmt.Errorf(
			"encode graph: %w",
			err,
		)
	}

	return string(data), nil
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
		"EdgesCount":  edgesCount,
		"AssetsCount": len(*graph),
	}
}