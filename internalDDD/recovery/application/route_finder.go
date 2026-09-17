package application

import (
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
)

type RouteFinder struct {
	graph marketDomain.Graph
}

func NewRouteFinder(
	graph marketDomain.Graph,
) RouteFinder {
	return RouteFinder{
		graph: graph,
	}
}

func (f RouteFinder) Find(
	startAsset string,
	targetAsset string,
	maxSteps int,
) []recoveryDomain.RecoveryRoute {
	if startAsset == "" || targetAsset == "" {
		return nil
	}

	if startAsset == targetAsset {
		return nil
	}

	if maxSteps <= 0 {
		return nil
	}

	result := make(
		[]recoveryDomain.RecoveryRoute,
		0,
	)

	f.dfs(
		targetAsset,
		startAsset,
		maxSteps,
		nil,
		&result,
	)

	return result
}

func (f RouteFinder) dfs(
	targetAsset string,
	current string,
	remaining int,
	path []marketDomain.GraphEdge,
	result *[]recoveryDomain.RecoveryRoute,
) {
	if remaining == 0 {
		return
	}

	edges, ok := f.graph[current]
	if !ok {
		return
	}

	for _, edge := range edges {
		// Не допускаем повторное посещение актива.
		if containsAsset(path, edge.To) {
			continue
		}

		next := append(
			append(
				[]marketDomain.GraphEdge{},
				path...,
			),
			edge,
		)

		// Маршрут подходит только тогда,
		// когда он дошёл до целевого актива.
		if edge.To == targetAsset {
			route := recoveryDomain.NewRecoveryRoute(next)

			*result = append(
				*result,
				route,
			)

			continue
		}

		f.dfs(
			targetAsset,
			edge.To,
			remaining-1,
			next,
			result,
		)
	}
}

func containsAsset(
	path []marketDomain.GraphEdge,
	asset string,
) bool {
	for _, edge := range path {
		if edge.From == asset ||
			edge.To == asset {
			return true
		}
	}

	return false
}