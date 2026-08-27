package domain

import "traiderBot/internalDDD/market/domain"

func routeIndexMetrics(index domain.RouteIndex) map[string]int {
	response := map[string]int{}

	for symbol, routes := range index {
		response[symbol] = len(routes)
	}

	return response
}

func symgolSnapshotRoutes(symbolIndex string, index domain.RouteIndex) map[string]any {
	response := map[string]any{}

	for symbol, routes := range index {
		if symbolIndex != symbol {
			continue
		}
		keys := []string{}
		for _, route := range routes {
			keys = append(keys, route.Key)
		}
		response[symbol] = keys
	}

	return response
}
