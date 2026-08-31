# debug — domain

Source: `internalDDD/debug/domain/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/debug/domain/graphProvider.go

Package: `domain`

### Structs

```go
GraphProvider struct {
	structureStore application.MarketStructureStore
}
```

### Constructors

```go
func NewGraphProvider(
	structureStore application.MarketStructureStore,
) *GraphProvider {
	return &GraphProvider{
		structureStore: structureStore,
	}
}
```

### Functions

```go
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
```

### Methods

```go
func (p *GraphProvider) Metrics() (any, error) {
	structure, ok := p.structureStore.Get()
	if !ok {
		return nil, fmt.Errorf(
			"market structure is not available",
		)
	}

	return graphMetrics(&structure.Graph), nil
}
```

```go
func (p *GraphProvider) Name() string {
	return "GraphProvider"
}
```

```go
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
```

## internalDDD/debug/domain/marketProvider.go

Package: `domain`

### Structs

```go
MarketMetrics struct {
}
```

```go
MarketProvider struct {
	marketStructure *domain.MarketStructure
}
```

### Constructors

```go
func NewMarketProvider(marketStructure *domain.MarketStructure) *MarketProvider {
	return &MarketProvider{
		marketStructure: marketStructure,
	}
}
```

### Methods

```go
func (p *MarketProvider) Metrics() map[string]any {
	response := map[string]any{
		"symbolsCount": len(p.marketStructure.Symbols),

		"routeIndex": routeIndexMetrics(p.marketStructure.RouteIndex),
	}
	return response
}
```

```go
func (p *MarketProvider) Name() string {
	return "MarketStructureProvider"
}
```

## internalDDD/debug/domain/orderBookProvider.go

Package: `domain`

### Interfaces

```go
OrderBookReader interface {
	Get(symbol string) (marketDomain.OrderBook, bool)
	Symbols() []string
}
```

### Structs

```go
OrderBookProvider struct {
	store OrderBookReader
}
```

### Constructors

```go
func NewOrderBookProvider(
	store OrderBookReader,
) *OrderBookProvider {
	return &OrderBookProvider{
		store: store,
	}
}
```

### Functions

```go
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(time.RFC3339Nano)
}
```

```go
func orderBookAge(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return time.Since(value).String()
}
```

### Methods

```go
func (p *OrderBookProvider) Metrics() (any, error) {
	symbols := p.store.Symbols()

	metrics := map[string]any{
		"symbols":    len(symbols),
		"orderBooks": len(symbols),
		"bySymbol":   make(map[string]any),
	}

	bySymbol := metrics["bySymbol"].(map[string]any)

	for _, symbol := range symbols {
		orderBook, ok := p.store.Get(symbol)
		if !ok {
			continue
		}

		bySymbol[symbol] = map[string]any{
			"lastUpdateID":   orderBook.LastUpdateID,
			"lastUpdateTime": formatTime(orderBook.LastUpdateTime),
			"bids":           len(orderBook.Bids),
			"asks":           len(orderBook.Asks),
			"age":            orderBookAge(orderBook.LastUpdateTime),
		}
	}

	return metrics, nil
}
```

```go
func (p *OrderBookProvider) Name() string {
	return "OrderBookProvider"
}
```

```go
func (p *OrderBookProvider) Snapshot() (string, error) {
	symbols := p.store.Symbols()

	if len(symbols) == 0 {
		return "No order books loaded.", nil
	}

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"Order books: %d\n\n",
		len(symbols),
	)

	for _, symbol := range symbols {
		orderBook, ok := p.store.Get(symbol)
		if !ok {
			continue
		}

		fmt.Fprintf(
			&builder,
			"%s\n",
			symbol,
		)

		fmt.Fprintf(
			&builder,
			"  LastUpdateID: %d\n",
			orderBook.LastUpdateID,
		)

		fmt.Fprintf(
			&builder,
			"  LastUpdateTime: %s\n",
			formatTime(orderBook.LastUpdateTime),
		)

		fmt.Fprintf(
			&builder,
			"  Bids: %d\n",
			len(orderBook.Bids),
		)

		fmt.Fprintf(
			&builder,
			"  Asks: %d\n",
			len(orderBook.Asks),
		)

		if len(orderBook.Bids) > 0 {
			fmt.Fprintf(
				&builder,
				"  BestBid: %s @ %s\n",
				orderBook.Bids[0].Qty,
				orderBook.Bids[0].Price,
			)
		}

		if len(orderBook.Asks) > 0 {
			fmt.Fprintf(
				&builder,
				"  BestAsk: %s @ %s\n",
				orderBook.Asks[0].Qty,
				orderBook.Asks[0].Price,
			)
		}

		fmt.Fprintln(&builder)
	}

	return builder.String(), nil
}
```

## internalDDD/debug/domain/provider.go

Package: `domain`

### Interfaces

```go
DomainProvider interface {
	Name() string
}
```

```go
MetricsProvider interface {
	DomainProvider
	Metrics() any
}
```

```go
SnapshotProvider interface {
	DomainProvider
	Snapshot() string
}
```

## internalDDD/debug/domain/routeIndexProvider.go

Package: `domain`

### Functions

```go
func routeIndexMetrics(index domain.RouteIndex) map[string]int {
	response := map[string]int{}

	for symbol, routes := range index {
		response[symbol] = len(routes)
	}

	return response
}
```

```go
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
```

## internalDDD/debug/domain/simulationProvider.go

Package: `domain`

### Structs

```go
SimulationProvider struct {
	reader interface {
		Snapshot() simulationApplication.WorkerMetricsSnapshot
	}
}
```

### Constructors

```go
func NewSimulationProvider(
	reader interface {
		Snapshot() simulationApplication.WorkerMetricsSnapshot
	},
) *SimulationProvider {
	return &SimulationProvider{
		reader: reader,
	}
}
```

### Methods

```go
func (p *SimulationProvider) Metrics() (any, error) {
	metrics := p.reader.Snapshot()

	response := map[string]any{
		"updatesReceived":     metrics.UpdatesReceived,
		"routesSimulated":     metrics.RoutesSimulated,
		"opportunities":       metrics.Opportunities,
		"totalSimulationTime": metrics.TotalSimulationTime,
		"lastUpdateAt":        metrics.LastUpdateAt,
		"lastSimulationAt":    metrics.LastSimulationAt,
	}

	if metrics.HasProfitRange {
		response["minProfitPct"] = metrics.MinProfitPct
		response["maxProfitPct"] = metrics.MaxProfitPct
	} else {
		response["minProfitPct"] = nil
		response["maxProfitPct"] = nil
	}

	if metrics.RoutesSimulated > 0 {
		response["averageSimulationTime"] =
			metrics.TotalSimulationTime /
				time.Duration(metrics.RoutesSimulated)
	} else {
		response["averageSimulationTime"] = time.Duration(0)
	}

	return response, nil
}
```

```go
func (p *SimulationProvider) Name() string {
	return "SimulationProvider"
}
```

```go
func (p *SimulationProvider) Snapshot() (string, error) {
	metrics := p.reader.Snapshot()

	var builder strings.Builder

	fmt.Fprintln(&builder, "Simulation Worker")
	fmt.Fprintln(&builder, "-----------------")

	fmt.Fprintf(
		&builder,
		"Updates received: %d\n",
		metrics.UpdatesReceived,
	)

	fmt.Fprintf(
		&builder,
		"Routes simulated: %d\n",
		metrics.RoutesSimulated,
	)

	fmt.Fprintf(
		&builder,
		"Opportunities: %d\n",
		metrics.Opportunities,
	)

	fmt.Fprintf(
		&builder,
		"Last update: %s\n",
		formatTime(metrics.LastUpdateAt),
	)

	fmt.Fprintf(
		&builder,
		"Last simulation: %s\n",
		formatTime(metrics.LastSimulationAt),
	)

	if metrics.HasProfitRange {
		fmt.Fprintf(
			&builder,
			"Min profit: %s%%\n",
			metrics.MinProfitPct.Mul(decimal.NewFromInt(100)),
		)

		fmt.Fprintf(
			&builder,
			"Max profit: %s%%\n",
			metrics.MaxProfitPct.Mul(decimal.NewFromInt(100)),
		)
	} else {
		fmt.Fprintln(
			&builder,
			"Profit range: no opportunities",
		)
	}

	return builder.String(), nil
}
```

