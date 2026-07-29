package route

import (
	"strings"

	"traiderBot/internal/graph"
)

// Route — последовательность из ровно 3 рёбер, образующих цикл
type Route []graph.Edge

// FindTriangularRoutes находит все уникальные направленные треугольники
func FindTriangularRoutes(graph graph.Graph) []Route {
	var routes []Route
	seen := make(map[string]struct{})

	for start, edges1 := range graph {
		for _, e1 := range edges1 {
			mid := e1.To
			if mid == start {
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
					route := Route{e1, e2, e3}

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
		if r[i].From < r[minIdx].From {
			minIdx = i
		}
	}

	// Ротируем маршрут так, чтобы он начинался с минимального актива
	rotated := make(Route, 3)
	for i := 0; i < 3; i++ {
		rotated[i] = r[(minIdx+i)%3]
	}

	// Собираем ключ из последовательности From + направления
	var b strings.Builder
	for _, e := range rotated {
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
