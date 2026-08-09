package route

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

type LifetimeStats struct {
	Median time.Duration `json:"mediad"`
	Values []time.Duration
}

type FreshnessStats struct {
	Symbols map[string]LifetimeStats `json:"symbols"`
}

func NewFreshnessStats() FreshnessStats {
	return FreshnessStats{Symbols: make(map[string]LifetimeStats)}
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

func (r *Route) UpdateFreshness(
	market *orderbook.Market,
	maxBookAge time.Duration,
	freshnessStats FreshnessStats,
	statsFoulderPath string,
) bool {
	// Очищаем причины предыдущего отказа.
	clear(r.OldSymbols)

	for _, edge := range r.Edges {
		now := time.Now()
		book, ok := market.GetOrderBook(edge.Symbol)

		if !ok {
			fmt.Printf(
				"FRESHNESS: symbol=%s BOOK NOT FOUND\n",
				edge.Symbol,
			)

			r.OldSymbols[edge.Symbol] = struct{}{}
			continue
		}

		lifetime := now.Sub(book.LastUpdateTime)

		stats, exists := freshnessStats.Symbols[edge.Symbol]

		if !exists {
			stats = LifetimeStats{
				Median: lifetime,
				Values: []time.Duration{lifetime},
			}

			freshnessStats.Symbols[edge.Symbol] = stats
		}

		// Обновляем статистику.
		stats.Median = medianDuration(stats.Values)
		appendRow(statsFoulderPath, edge.Symbol, lifetime, stats.Median, now)

		// Проверяем freshness.
		if lifetime > maxBookAge {
			r.OldSymbols[edge.Symbol] = struct{}{}
			continue
		}
	}

	return len(r.OldSymbols) == 0
}

func medianDuration(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sorted := append(
		[]time.Duration(nil),
		values...,
	)

	sort.Slice(
		sorted,
		func(i, j int) bool {
			return sorted[i] < sorted[j]
		},
	)

	n := len(sorted)

	if n%2 == 1 {
		return sorted[n/2]
	}

	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func appendRow(pathToFoulder, symbol string, age, median time.Duration, curTime time.Time) error {

	if err := os.MkdirAll(pathToFoulder, 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(
		filepath.Join(pathToFoulder, symbol+".csv"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat csv file: %w", err)
	}

	if info.Size() == 0 {
		if err := writer.Write([]string{
			"time", "age", "median",
		}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
	}

	if err := writer.Write([]string{
		curTime.Format(time.RFC3339Nano),
		strconv.FormatInt(age.Milliseconds(), 10),
		strconv.FormatInt(median.Milliseconds(), 10),
	}); err != nil {
		return fmt.Errorf("write csv row: %w", err)

	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flusth csv: %w", err)
	}

	return nil
}

type LifetimeStatsLog struct {
	Counter int       `json:"counter"`
	Age     string    `json:"ag3"`
	CurTime time.Time `json:"curTime"`
}

type FreshnessLogOutput struct {
	Symbols  map[string]LifetimeStatsLog `json:"symbols"`
	Counter  int                         `json:"counter"`
	Timemark time.Time                   `json:"time_mark"`
}
