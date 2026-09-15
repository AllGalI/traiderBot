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
	maxSteps int,
) []recoveryDomain.RecoveryRoute {

	if maxSteps <= 0 {
		return nil
	}


	result := make([]recoveryDomain.RecoveryRoute,0)


	f.dfs(
		startAsset,
		startAsset,
		maxSteps,
		nil,
		&result,
	)


	return result
}



func (f RouteFinder) dfs(
	start string,
	current string,
	remaining int,

	path []marketDomain.GraphEdge,

	result *[]recoveryDomain.RecoveryRoute,
) {


	if len(path) > 0 {

		route :=
			recoveryDomain.NewRecoveryRoute(
				append(
					[]marketDomain.GraphEdge{},
					path...,
				),
			)


		*result = append(
			*result,
			route,
		)
	}



	if remaining == 0 {
		return
	}



	edges, ok := f.graph[current]

	if !ok {
		return
	}



	for _, edge := range edges {


		// запрещаем сразу вернуться назад
		if containsAsset(
			path,
			edge.To,
		) {
			continue
		}


		next :=
			append(
				path,
				edge,
			)


		f.dfs(
			start,
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