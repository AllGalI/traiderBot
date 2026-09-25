package domain


type RouteIndex map[string][]Route

func BuildRouteIndex(routes []Route) RouteIndex {
	index := make(RouteIndex)

	for _, route := range routes {
		seenSymbols := make(map[string]struct{}, len(route.Edges))

		for _, edge := range route.Edges {
			if _, exists := seenSymbols[edge.Symbol]; exists {
				continue
			}

			seenSymbols[edge.Symbol] = struct{}{}
			index[edge.Symbol] = append(index[edge.Symbol], route)
		}
	}

	return index
}

func (index RouteIndex) Get(symbol string) []Route {
	return index[symbol]
}

func (index RouteIndex) Symbols() []string {
	symbols := make([]string, 0, len(index))

	for symbol := range index {
		symbols = append(symbols, symbol)
	}

	return symbols
}
