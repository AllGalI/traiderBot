# market — application

Source: `internalDDD/market/application/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/market/application/market_cache.go

Package: `application`

### Interfaces

```go
MarketCache interface {
	Get(ctx context.Context) (domain.MarketStructure, bool, error)

	Set(ctx context.Context, structure domain.MarketStructure) error
}
```

## internalDDD/market/application/market_data_service.go

Package: `application`

### Interfaces

```go
OrderBookProvider interface {
	Get(
		ctx context.Context,
		symbol string,
		depth int,
	) (domain.OrderBook, error)
}
```

```go
OrderBookStore interface {
	Get(symbol string) (domain.OrderBook, bool)
	Set(orderBook domain.OrderBook)
	Symbols() []string
}
```

### Structs

```go
MarketDataService struct {
	provider OrderBookProvider
	stream   OrderBookStream
	store    OrderBookStore

	updates chan string
}
```

### Constructors

```go
func NewMarketDataService(
	provider OrderBookProvider,
	stream OrderBookStream,
	store OrderBookStore,
) *MarketDataService {
	return &MarketDataService{
		provider: provider,
		stream:   stream,
		store:    store,
		updates:  make(chan string, 1024),
	}
}
```

### Methods

```go
func (s *MarketDataService) GetOrderBook(
	symbol string,
) (domain.OrderBook, bool) {
	return s.store.Get(symbol)
}
```

```go
func (s *MarketDataService) Start(
	ctx context.Context,
	symbols []string,
	depth int,
) error {
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	for _, symbol := range symbols {
		log.Printf("MarketData: requesting snapshot %s", symbol)

		started := time.Now()

		orderBook, err := s.provider.Get(
			ctx,
			symbol,
			depth,
		)

		log.Printf(
			"MarketData: snapshot %s finished in %s",
			symbol,
			time.Since(started),
		)

		if err != nil {
			return fmt.Errorf(
				"load order book %s: %w",
				symbol,
				err,
			)
		}

		s.store.Set(orderBook)
	}

	log.Printf(
		"MarketData: starting websocket for %d symbols",
		len(symbols),
	)

	return s.stream.Subscribe(
		ctx,
		symbols,
		depth,
		func(orderBook domain.OrderBook) {
			s.store.Set(orderBook)
			s.publishUpdate(orderBook.Symbol)
		},
	)

}
```

```go
func (s *MarketDataService) Updates() <-chan string {
	return s.updates
}
```

```go
func (s *MarketDataService) publishUpdate(symbol string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if symbol == "" {
		return
	}

	select {
	case s.updates <- symbol:
	default:
		log.Printf(
			"MarketData: updates channel full, dropping update %s", symbol,
		)
	}

}
```

## internalDDD/market/application/market_graph_service.go

Package: `application`

### Structs

```go
MarketGraph struct {
	Graph      domain.Graph
	Routes     []domain.Route
	RouteIndex domain.RouteIndex
}
```

```go
MarketGraphService struct{}
```

### Constructors

```go
func NewMarketGraphService() *MarketGraphService {
	return &MarketGraphService{}
}
```

### Methods

```go
func (s *MarketGraphService) Build(
	symbols []domain.Symbol,
	startAssets map[string]struct{},
) (MarketGraph, error) {
	graph, err := domain.NewGraph(symbols)
	if err != nil {
		return MarketGraph{}, err
	}

	routes := domain.FindTriangularRoutes(
		graph,
		startAssets,
	)

	routeIndex := domain.BuildRouteIndex(routes)

	return MarketGraph{
		Graph:      graph,
		Routes:     routes,
		RouteIndex: routeIndex,
	}, nil
}
```

## internalDDD/market/application/market_orcestrator_service.go

Package: `application`

### Interfaces

```go
MarketDataStarter interface {
	Start(
		ctx context.Context,
		symbols []string,
		depth int,
	) error
}
```

```go
MarketStructureBuilder interface {
	Build(
		ctx context.Context,
		startAssets map[string]struct{},
	) (domain.MarketStructure, error)
}
```

### Structs

```go
MarketService struct {
	cache             MarketCache
	structureService  MarketStructureBuilder
	marketDataService MarketDataStarter
	structureStore    MarketStructureStore

	ready chan struct{}
}
```

### Constructors

```go
func NewMarketService(
	cache MarketCache,
	structureService MarketStructureBuilder,
	marketDataService MarketDataStarter,
	structureStore MarketStructureStore,
) *MarketService {
	return &MarketService{
		cache:             cache,
		structureService:  structureService,
		marketDataService: marketDataService,
		structureStore:    structureStore,
		ready:             make(chan struct{}),
	}
}
```

### Methods

```go
func (s *MarketService) Ready() <-chan struct{} {
	return s.ready
}
```

```go
func (s *MarketService) Start(
	ctx context.Context,
	startAssets map[string]struct{},
	depth, symbolsLimit int,
) error {
	log.Println("MarketService: START")

	structure, ok, err := s.cache.Get(ctx)
	if err != nil {
		return fmt.Errorf(
			"get market structure from cache: %w",
			err,
		)
	}

	log.Printf("MarketService: cache hit=%v", ok)

	if !ok {
		log.Println("MarketService: building structure")

		structure, err = s.structureService.Build(
			ctx,
			startAssets,
		)
		if err != nil {
			return fmt.Errorf(
				"build market structure: %w",
				err,
			)
		}

		log.Println("MarketService: structure built")

		if err := s.cache.Set(ctx, structure); err != nil {
			return fmt.Errorf(
				"cache market structure: %w",
				err,
			)
		}

		log.Println("MarketService: structure cached")
	}
	s.structureStore.Set(&structure)
	close(s.ready)

	symbols := structure.RouteIndex.Symbols()

	log.Printf(
		"MarketService: total symbols=%d, limit=%d",
		len(symbols),
		symbolsLimit,
	)

	if symbolsLimit <= 0 {
		return fmt.Errorf("symbols limit must be greater than zero")
	}

	if symbolsLimit < len(symbols) {
		symbols = symbols[:symbolsLimit]
	}

	log.Printf(
		"MarketService: selected symbols=%v",
		symbols,
	)

	if len(symbols) == 0 {
		return fmt.Errorf("no symbols found in market routes")
	}

	log.Println("MarketService: starting market data")

	if err := s.marketDataService.Start(
		ctx,
		symbols,
		depth,
	); err != nil {
		return fmt.Errorf(
			"start market data service: %w",
			err,
		)
	}

	log.Println("MarketService: market data stopped")

	return nil
}
```

## internalDDD/market/application/market_structure_service.go

Package: `application`

### Interfaces

```go
MarketStructureStore interface {
	Get() (*domain.MarketStructure, bool)
	Set(*domain.MarketStructure)
}
```

### Structs

```go
MarketStructureService struct {
	symbolProvider SymbolProvider
}
```

### Constructors

```go
func NewMarketStructureService(
	symbolProvider SymbolProvider,
) *MarketStructureService {
	return &MarketStructureService{
		symbolProvider: symbolProvider,
	}
}
```

### Methods

```go
func (s *MarketStructureService) Build(
	ctx context.Context,
	startAssets map[string]struct{},
) (domain.MarketStructure, error) {
	symbols, err := s.symbolProvider.GetSymbols(ctx)
	if err != nil {
		return domain.MarketStructure{}, fmt.Errorf(
			"load symbols: %w",
			err,
		)
	}

	graph, err := domain.NewGraph(symbols)
	if err != nil {
		return domain.MarketStructure{}, fmt.Errorf(
			"build graph: %w",
			err,
		)
	}

	routes := domain.FindTriangularRoutes(
		graph,
		startAssets,
	)

	routeIndex := domain.BuildRouteIndex(routes)

	return domain.MarketStructure{
		Symbols:    symbols,
		Graph:      graph,
		Routes:     routes,
		RouteIndex: routeIndex,
	}, nil
}
```

## internalDDD/market/application/orderbook_stream.go

Package: `application`

### Interfaces

```go
OrderBookStream interface {
	Subscribe(
		ctx context.Context,
		symbols []string,
		depth int,
		handler OrderBookUpdateHandler,
	) error
}
```

## internalDDD/market/application/symbol_provider.go

Package: `application`

### Interfaces

```go
SymbolProvider interface {
	Get(
		ctx context.Context,
		symbols []string,
	) ([]domain.Symbol, error)

	GetSymbols(ctx context.Context) ([]domain.Symbol, error)
}
```

