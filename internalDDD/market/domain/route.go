package domain

import "strings"

type Route struct {
	Edges []GraphEdge
	Key   string
}

func NewRoute(edges []GraphEdge) Route {
	return Route{
		Edges: edges,
		Key:   key(edges),
	}
}

func (r Route) Symbols() []string {
	symbols := make([]string, 0, len(r.Edges))

	for _, edge := range r.Edges {
		symbols = append(symbols, edge.Symbol)
	}

	return symbols
}

func key(edges []GraphEdge) string {
	if len(edges) == 0 {
		return ""
	}

	parts := make([]string, 0, len(edges))

	for _, edge := range edges {
		direction := "S"
		if edge.Direction == Buy {
			direction = "B"
		}

		parts = append(
			parts,
			edge.From+":"+direction+":"+edge.To,
		)
	}

	return strings.Join(parts, "|")
}

func (r Route) IsTriangle() bool {
	if len(r.Edges) != 3 {
		return false
	}

	first := r.Edges[0]
	second := r.Edges[1]
	third := r.Edges[2]

	return first.To == second.From &&
		second.To == third.From &&
		third.To == first.From
}

func FindTriangularRoutes(
	graph Graph,
	startAssets map[string]struct{},
) []Route {
	routes := make([]Route, 0)

	seen := make(map[string]struct{})

	for startAsset := range startAssets {
		for _, first := range graph[startAsset] {
			middleAsset := first.To

			if middleAsset == startAsset {
				continue
			}

			for _, second := range graph[middleAsset] {
				endAsset := second.To

				if endAsset == startAsset ||
					endAsset == middleAsset {
					continue
				}

				for _, third := range graph[endAsset] {
					if third.To != startAsset {
						continue
					}

					route := NewRoute([]GraphEdge{
						first,
						second,
						third,
					})

					key := canonicalRouteKey(route)

					if _, exists := seen[key]; exists {
						continue
					}

					seen[key] = struct{}{}
					routes = append(routes, route)
				}
			}
		}
	}

	return routes
}

func canonicalRouteKey(route Route) string {
	if len(route.Edges) != 3 {
		return route.Key
	}

	minIndex := 0

	for i := 1; i < 3; i++ {
		if route.Edges[i].From < route.Edges[minIndex].From {
			minIndex = i
		}
	}

	parts := make([]string, 0, 3)

	for i := 0; i < 3; i++ {
		edge := route.Edges[(minIndex+i)%3]

		direction := "S"
		if edge.Direction == Buy {
			direction = "B"
		}

		parts = append(
			parts,
			edge.From+":"+direction+":"+edge.To,
		)
	}

	return strings.Join(parts, "|")
}

func (r *Route) RouteKey() string {
	// Находим индекс актива с минимальным именем
	minIdx := 0
	for i := 1; i < 3; i++ {
		if r.Edges[i].From < r.Edges[minIdx].From {
			minIdx = i
		}
	}

	// Ротируем маршрут так, чтобы он начинался с минимального актива
	rotated := NewRoute(make([]GraphEdge, 3))

	for i := 0; i < 3; i++ {
		rotated.Edges[i] = r.Edges[(minIdx+i)%3]
	}

	// Собираем ключ из последовательности From + направления
	var b strings.Builder
	for _, e := range rotated.Edges {
		b.WriteString(e.From)
		if e.IsBuy() {
			b.WriteByte('B')
		} else {
			b.WriteByte('S')
		}
		b.WriteByte('|')
	}
	return b.String()
}
