# market — domain

Source: `internalDDD/market/domain/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/market/domain/fee_policy.go

Package: `domain`

### Structs

```go
FeeRate struct {
	Maker float64
	Taker float64
}
```

## internalDDD/market/domain/graph.go

Package: `domain`

### Structs

```go
GraphEdge struct {
	From      string
	To        string
	Symbol    string
	Direction TradeDirection
}
```

### Constructors

```go
func NewGraph(symbols []Symbol) (Graph, error) {
	graph := make(Graph)

	for _, symbol := range symbols {
		if symbol.BaseAsset == "" {
			return nil, fmt.Errorf(
				"symbol %q has empty base asset",
				symbol.Name,
			)
		}

		if symbol.QuoteAsset == "" {
			return nil, fmt.Errorf(
				"symbol %q has empty quote asset",
				symbol.Name,
			)
		}

		graph[symbol.QuoteAsset] = append(
			graph[symbol.QuoteAsset],
			GraphEdge{
				From:      symbol.QuoteAsset,
				To:        symbol.BaseAsset,
				Symbol:    symbol.Name,
				Direction: Buy,
			},
		)

		graph[symbol.BaseAsset] = append(
			graph[symbol.BaseAsset],
			GraphEdge{
				From:      symbol.BaseAsset,
				To:        symbol.QuoteAsset,
				Symbol:    symbol.Name,
				Direction: Sell,
			},
		)
	}

	return graph, nil
}
```

### Methods

```go
func (e *GraphEdge) IsBuy() bool {
	return e.Direction == Buy
}
```

## internalDDD/market/domain/market_structure.go

Package: `domain`

### Structs

```go
MarketStructure struct {
	Symbols    []Symbol
	Graph      Graph
	Routes     []Route
	RouteIndex RouteIndex
}
```

## internalDDD/market/domain/orderbook.go

Package: `domain`

### Structs

```go
OrderBook struct {
	Symbol         string
	LastUpdateID   int64
	LastUpdateTime time.Time
	Asks           []PriceLevel
	Bids           []PriceLevel
}
```

```go
PriceLevel struct {
	Price decimal.Decimal
	Qty   decimal.Decimal
}
```

### Methods

```go
func (ob *OrderBook) GetBestAsk() (PriceLevel, bool) {
	if len(ob.Asks) == 0 {
		return PriceLevel{}, false
	}
	return ob.Asks[0], true
}
```

```go
func (ob *OrderBook) GetBestBid() (PriceLevel, bool) {
	if len(ob.Bids) == 0 {
		return PriceLevel{}, false
	}
	return ob.Bids[0], true
}
```

## internalDDD/market/domain/route.go

Package: `domain`

### Structs

```go
Route struct {
	Edges []GraphEdge
	Key   string
}
```

### Constructors

```go
func NewRoute(edges []GraphEdge) Route {
	return Route{
		Edges: edges,
		Key:   key(edges),
	}
}
```

### Functions

```go
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
```

```go
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
```

```go
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
```

### Methods

```go
func (r *Route) RouteKey() string {

	minIdx := 0
	for i := 1; i < 3; i++ {
		if r.Edges[i].From < r.Edges[minIdx].From {
			minIdx = i
		}
	}

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
```

```go
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
```

```go
func (r Route) Symbols() []string {
	symbols := make([]string, 0, len(r.Edges))

	for _, edge := range r.Edges {
		symbols = append(symbols, edge.Symbol)
	}

	return symbols
}
```

## internalDDD/market/domain/route_index.go

Package: `domain`

### Functions

```go
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
```

### Methods

```go
func (index RouteIndex) Get(symbol string) []Route {
	return index[symbol]
}
```

```go
func (index RouteIndex) Symbols() []string {
	symbols := make([]string, 0, len(index))

	for symbol := range index {
		symbols = append(symbols, symbol)
	}

	return symbols
}
```

## internalDDD/market/domain/symbol.go

Package: `domain`

### Structs

```go
Symbol struct {
	Name        string
	BaseAsset   string
	QuoteAsset  string
	Constraints SymbolConstraints
}
```

```go
SymbolConstraints struct {
	MinQty      decimal.Decimal
	MaxQty      decimal.Decimal
	StepSize    decimal.Decimal
	MinNotional decimal.Decimal
	PriceStep   decimal.Decimal
}
```

