package route

import (
	"strings"
	"time"

	"traiderBot/internal/graphb"
	"traiderBot/internal/orderbook"
)

// Route — последовательность из ровно 3 рёбер, образующих цикл
type Route struct {
	Edges      []graphb.Edge
	OldSymbols map[string]struct{}
}

func NewRoute(edges []graphb.Edge) Route {
	return Route{
		Edges:      edges,
		OldSymbols: make(map[string]struct{}, 0),
	}
}

func (r *Route) UpdateFreshness(
	market *orderbook.Market,
	maxBookAge time.Duration,
) bool {

	// очищаем причины предыдущего отказа
	clear(r.OldSymbols)

	now := time.Now()

	for _, edge := range r.Edges {
		book, ok := market.GetOrderBook(edge.Symbol)

		if !ok {
			r.OldSymbols[edge.Symbol] = struct{}{}
		}

		if now.Sub(book.LastUpdateTime) > maxBookAge {
			r.OldSymbols[edge.Symbol] = struct{}{}
		}
	}

	return len(r.OldSymbols) == 0
}

// FindTriangularRoutes находит все уникальные направленные треугольники
func FindTriangularRoutes(graph graphb.Graph, startAmounts map[string]float64) []Route {
	var routes []Route
	seen := make(map[string]struct{})

	for start, edges1 := range graph {
		for _, e1 := range edges1 {
			mid := e1.To
			if mid == start {
				continue
			}
			if _, ok := startAmounts[start]; !ok {
				continue
			}

			for _, e2 := range graph[mid] {
				end := e2.To
				if end == start || end == mid {
					continue
				}

				// Ищем ребро, которое замыкает цикл: end → start
				for _, e3 := range graph[end] {
					if e3.To != start {
						continue
					}

					// Нашли цикл: start → mid → end → start
					route := NewRoute([]graphb.Edge{e1, e2, e3})

					// Нормализуем ключ, чтобы убрать ротации одного и того же направленного цикла
					key := routeKey(route)
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

// routeKey делает ключ уникальным для направленного цикла
// (ротации одного цикла дают одинаковый ключ, обратный цикл — другой)
func routeKey(r Route) string {
	// Находим индекс актива с минимальным именем
	minIdx := 0
	for i := 1; i < 3; i++ {
		if r.Edges[i].From < r.Edges[minIdx].From {
			minIdx = i
		}
	}

	// Ротируем маршрут так, чтобы он начинался с минимального актива
	rotated := NewRoute(make([]graphb.Edge, 3))

	for i := 0; i < 3; i++ {
		rotated.Edges[i] = r.Edges[(minIdx+i)%3]
	}

	// Собираем ключ из последовательности From + направления
	var b strings.Builder
	for _, e := range rotated.Edges {
		b.WriteString(e.From)
		if e.IsBuy {
			b.WriteByte('B')
		} else {
			b.WriteByte('S')
		}
		b.WriteByte('|')
	}
	return b.String()
}

func BuildRouteIndex(routes []Route) map[string][]Route {
	index := make(map[string][]Route)

	for _, r := range routes {
		// каждый маршрут содержит 3 ребра → 3 символа
		seenInRoute := make(map[string]struct{}, 3) // на случай, если вдруг один символ встретится дважды (маловероятно)

		for _, edge := range r.Edges {
			sym := edge.Symbol
			if _, ok := seenInRoute[sym]; ok {
				continue
			}
			seenInRoute[sym] = struct{}{}

			index[sym] = append(index[sym], r)
		}
	}

	return index
}

// RouteKey возвращает человекочитаемый ключ маршрута.
// Пример: "BTCUSDT→ETHBTC→ETHUSDT"
func RouteKey(route Route) string {
	if len(route.Edges) == 0 {
		return ""
	}

	parts := make([]string, 0, len(route.Edges))
	for _, edge := range route.Edges {
		parts = append(parts, edge.Symbol)
	}
	return strings.Join(parts, "→")
}
