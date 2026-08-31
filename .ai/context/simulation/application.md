# simulation — application

Source: `internalDDD/simulation/application/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/simulation/application/fee.go

Package: `application`

### Interfaces

```go
FeeProvider interface {
	Get(ctx context.Context, symbol string) (FeeConfig, bool)
}
```

### Structs

```go
FeeConfig struct {
	Rate decimal.Decimal
}
```

## internalDDD/simulation/application/fee_cache.go

Package: `application`

### Structs

```go
FeeCache struct {
	mu   sync.RWMutex
	fees map[string]FeeConfig
}
```

### Constructors

```go
func NewFeeCache() *FeeCache {
	return &FeeCache{
		fees: make(map[string]FeeConfig),
	}
}
```

### Methods

```go
func (c *FeeCache) Get(
	ctx context.Context,
	symbol string,
) (FeeConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fee, ok := c.fees[symbol]

	return fee, ok
}
```

```go
func (c *FeeCache) Set(
	symbol string,
	fee FeeConfig,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.fees[symbol] = fee
}
```

## internalDDD/simulation/application/fee_loader.go

Package: `application`

### Interfaces

```go
FeeLoader interface {
	Get(ctx context.Context, symbol string) (FeeConfig, error)
}
```

## internalDDD/simulation/application/fee_refresh_worker.go

Package: `application`

### Structs

```go
FeeRefreshWorker struct {
	loader          FeeLoader
	cache           *FeeCache
	marketStructure MarketStructureReader
	interval        time.Duration
}
```

### Constructors

```go
func NewFeeRefreshWorker(
	loader FeeLoader,
	cache *FeeCache,
	marketStructure MarketStructureReader,
	interval time.Duration,
) *FeeRefreshWorker {
	return &FeeRefreshWorker{
		loader:          loader,
		cache:           cache,
		marketStructure: marketStructure,
		interval:        interval,
	}
}
```

### Methods

```go
func (w *FeeRefreshWorker) Run(ctx context.Context) error {
	if err := w.refresh(ctx); err != nil {
		log.Printf("initial fee refresh failed: %v", err)
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := w.refresh(ctx); err != nil {
				log.Printf("fee refresh failed: %v", err)
			}
		}
	}
}
```

```go
func (w *FeeRefreshWorker) refresh(
	ctx context.Context,
) error {
	structure, ok := w.marketStructure.Get()
	if !ok {
		return nil
	}

	for _, symbol := range structure.Symbols {
		if err := ctx.Err(); err != nil {
			return err
		}

		fee, err := w.loader.Get(
			ctx,
			symbol.Name,
		)
		if err != nil {
			log.Printf(
				"fee refresh failed for %s: %v",
				symbol.Name,
				err,
			)

			continue
		}

		w.cache.Set(
			symbol.Name,
			fee,
		)
	}

	return nil
}
```

## internalDDD/simulation/application/freshness.go

Package: `application`

### Interfaces

```go
FreshnessPolicy interface {
	IsFresh(symbol string, updatedAt time.Time, now time.Time) bool
}
```

### Structs

```go
FixedTTLFreshnessPolicy struct {
	ttls map[string]time.Duration
}
```

### Constructors

```go
func NewFixedTTLFreshnessPolicy(
	ttls map[string]time.Duration,
) *FixedTTLFreshnessPolicy {
	copied := make(map[string]time.Duration, len(ttls))

	for symbol, ttl := range ttls {
		copied[symbol] = ttl
	}

	return &FixedTTLFreshnessPolicy{
		ttls: copied,
	}
}
```

### Methods

```go
func (p *FixedTTLFreshnessPolicy) IsFresh(
	symbol string,
	updatedAt time.Time,
	now time.Time,
) bool {
	ttl, ok := p.ttls[symbol]
	if !ok {
		return false
	}

	if updatedAt.IsZero() {
		return false
	}

	if now.Before(updatedAt) {
		return false
	}

	return now.Sub(updatedAt) <= ttl
}
```

## internalDDD/simulation/application/freshness_config.go

Package: `application`

### Structs

```go
FreshnessConfig struct {
	GeneratedAt  time.Time                  `json:"generated_at"`
	DefaultTTLMS int64                      `json:"default_ttl_ms"`
	Symbols      map[string]FreshnessSymbol `json:"symbols"`
}
```

```go
FreshnessSymbol struct {
	TTLMS        int64  `json:"ttl_ms"`
	Percentile   string `json:"percentile"`
	PercentileMS int64  `json:"percentile_ms"`
	SampleCount  int64  `json:"sample_count"`
}
```

### Methods

```go
func (c *FreshnessConfig) TTLs() map[string]time.Duration {
	ttls := make(map[string]time.Duration, len(c.Symbols))

	for symbol, item := range c.Symbols {
		ttls[symbol] = time.Duration(item.TTLMS) * time.Millisecond
	}

	return ttls
}
```

## internalDDD/simulation/application/freshness_config_loader.go

Package: `application`

### Functions

```go
func LoadFreshnessConfig(
	path string,
) (FreshnessConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FreshnessConfig{}, fmt.Errorf(
			"read freshness config: %w",
			err,
		)
	}

	var config FreshnessConfig

	if err := json.Unmarshal(data, &config); err != nil {
		return FreshnessConfig{}, fmt.Errorf(
			"decode freshness config: %w",
			err,
		)
	}

	return config, nil
}
```

### Methods

```go
func (c FreshnessConfig) NewPolicy() *FixedTTLFreshnessPolicy {
	ttls := make(map[string]time.Duration, len(c.Symbols))

	for symbol, config := range c.Symbols {
		ttls[symbol] = time.Duration(config.TTLMS) * time.Millisecond
	}

	return NewFixedTTLFreshnessPolicy(ttls)
}
```

## internalDDD/simulation/application/market_structure_reader.go

Package: `application`

### Interfaces

```go
MarketStructureReader interface {
	Get() (*marketDomain.MarketStructure, bool)
}
```

## internalDDD/simulation/application/simulation_service.go

Package: `application`

### Interfaces

```go
OrderBookFreshness interface {
	IsFresh(symbol string, updatedAt time.Time, now time.Time) bool
}
```

```go
OrderBookReader interface {
	Get(symbol string) (marketDomain.OrderBook, bool)
}
```

```go
SymbolProvider interface {
	Get(symbol string) (marketDomain.Symbol, bool)
}
```

### Structs

```go
SimulationService struct {
	orderBooks            OrderBookReader
	feeProvider           FeeProvider
	symbols               SymbolProvider
	freshness             OrderBookFreshness
	minLiquidityFillRatio decimal.Decimal
	minProfitPct          decimal.Decimal
}
```

### Constructors

```go
func NewSimulationService(
	orderBooks OrderBookReader,
	feeProvider FeeProvider,
	symbols SymbolProvider,
	freshness OrderBookFreshness,
	minLiquidityFillRatio decimal.Decimal,
	minProfitPct decimal.Decimal,
) *SimulationService {
	return &SimulationService{
		orderBooks:            orderBooks,
		feeProvider:           feeProvider,
		symbols:               symbols,
		freshness:             freshness,
		minLiquidityFillRatio: minLiquidityFillRatio,
		minProfitPct:          minProfitPct,
	}
}
```

### Functions

```go
func startAsset(route marketDomain.Route) string {
	if len(route.Edges) == 0 {
		return ""
	}

	return route.Edges[0].From
}
```

### Methods

```go
func (s *SimulationService) Simulate(
	ctx context.Context,
	route marketDomain.Route,
	startAmount decimal.Decimal,
) simulationDomain.SimulationResult {
	if !route.IsTriangle() {
		return simulationDomain.SimulationResult{
			Reason: simulationDomain.SimulationInvalidRoute,
		}
	}

	if startAmount.LessThanOrEqual(decimal.Zero) {
		return simulationDomain.SimulationResult{
			Reason: simulationDomain.SimulationInvalidStartAmount,
		}
	}

	currentAmount := startAmount
	routeFillRatio := decimal.NewFromInt(1)

	steps := make(
		[]simulationDomain.StepResult,
		0,
		len(route.Edges),
	)

	if err := ctx.Err(); err != nil {
		return simulationDomain.SimulationResult{
			Reason: simulationDomain.SimulationInvalidStartAmount,
		}
	}

	for _, edge := range route.Edges {
		orderBook, ok := s.orderBooks.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationOrderBookNotFound,
			}
		}

		if s.freshness != nil &&
			!s.freshness.IsFresh(
				edge.Symbol,
				orderBook.LastUpdateTime,
				time.Now(),
			) {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationOrderBookStale,
			}
		}

		symbol, ok := s.symbols.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationFiltersNotFound,
			}
		}

		fee, ok := s.feeProvider.Get(ctx, edge.Symbol)
		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationFeeNotFound,
			}
		}

		step, err := simulationDomain.SimulateStep(
			edge,
			orderBook,
			symbol.Constraints,
			currentAmount,
			s.minLiquidityFillRatio,
			fee.Rate,
			edge.To,
		)

		if err != nil {
			fmt.Printf(
				"DEBUG SimulateStep: symbol=%s from=%s to=%s direction=%d amountIn=%s error=%v\n",
				edge.Symbol,
				edge.From,
				edge.To,
				edge.Direction,
				currentAmount,
				err,
			)

			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationInsufficientLiquidity,
			}
		}

		steps = append(steps, step)

		routeFillRatio = routeFillRatio.Mul(step.FillRatio)

		currentAmount = step.AmountOut
	}

	executedStartAmount := startAmount.Mul(routeFillRatio)

	profitAbs := currentAmount.Sub(executedStartAmount)

	profitPct := decimal.Zero

	if !executedStartAmount.IsZero() {
		profitPct = profitAbs.Div(executedStartAmount)
	}

	if profitPct.LessThan(s.minProfitPct) {
		return simulationDomain.SimulationResult{
			Reason:         simulationDomain.SimulationProfitTooLow,
			RouteFillRatio: routeFillRatio,
		}
	}

	return simulationDomain.SimulationResult{
		Reason:         simulationDomain.SimulationOK,
		RouteFillRatio: routeFillRatio,

		Opportunity: &simulationDomain.Opportunity{
			Route: route,
			Steps: steps,

			StartAsset: startAsset(route),

			RequestedStartAmount: startAmount,
			ExecutedStartAmount:  executedStartAmount,
			EndAmount:            currentAmount,

			FillRatio: routeFillRatio,

			ProfitAbs: profitAbs,
			ProfitPct: profitPct,

			Timestamp: time.Now(),
		},
	}
}
```

## internalDDD/simulation/application/symbol_provider.go

Package: `application`

### Interfaces

```go
MarketStructureStore interface {
	Get() (*marketDomain.MarketStructure, bool)
}
```

### Structs

```go
MarketStructureSymbolProvider struct {
	store MarketStructureStore
}
```

### Constructors

```go
func NewMarketStructureSymbolProvider(
	store MarketStructureStore,
) *MarketStructureSymbolProvider {
	return &MarketStructureSymbolProvider{
		store: store,
	}
}
```

### Methods

```go
func (p *MarketStructureSymbolProvider) Get(
	symbol string,
) (marketDomain.Symbol, bool) {
	structure, ok := p.store.Get()
	if !ok {
		return marketDomain.Symbol{}, false
	}

	for _, item := range structure.Symbols {
		if item.Name == symbol {
			return item, true
		}
	}

	return marketDomain.Symbol{}, false
}
```

## internalDDD/simulation/application/symbol_reader.go

Package: `application`

### Structs

```go
SymbolReader struct {
	structure MarketStructureReader
}
```

### Constructors

```go
func NewSymbolReader(
	structure MarketStructureReader,
) *SymbolReader {
	return &SymbolReader{
		structure: structure,
	}
}
```

### Methods

```go
func (r *SymbolReader) Get(
	symbol string,
) (marketDomain.Symbol, bool) {
	structure, ok := r.structure.Get()
	if !ok {
		return marketDomain.Symbol{}, false
	}

	for _, item := range structure.Symbols {
		if item.Name == symbol {
			return item, true
		}
	}

	return marketDomain.Symbol{}, false
}
```

## internalDDD/simulation/application/tests/freshness_stub.go

Package: `tests`

### Structs

```go
alwaysFreshPolicy struct{}
```

### Methods

```go
func (alwaysFreshPolicy) IsFresh(
	symbol string,
	updatedAt time.Time,
	now time.Time,
) bool {
	return true
}
```

## internalDDD/simulation/application/worker.go

Package: `application`

### Interfaces

```go
OpportunityHandler interface {
	Handle(*simulationDomain.Opportunity)
}
```

```go
Simulator interface {
	Simulate(
		ctx context.Context,
		route marketDomain.Route,
		startAmount decimal.Decimal,
	) simulationDomain.SimulationResult
}
```

### Structs

```go
Worker struct {
	marketStructure MarketStructureReader
	updates         <-chan string
	simulation      Simulator
	opportunities   OpportunityHandler

	metrics *WorkerMetrics

	startAmount decimal.Decimal
}
```

### Constructors

```go
func NewWorker(
	marketStructure MarketStructureReader,
	updates <-chan string,
	simulation Simulator,
	opportunities OpportunityHandler,
	startAmount decimal.Decimal,
) *Worker {
	return &Worker{
		marketStructure: marketStructure,
		updates:         updates,
		simulation:      simulation,
		opportunities:   opportunities,
		metrics:         NewWorkerMetrics(),
		startAmount:     startAmount,
	}
}
```

### Methods

```go
func (w *Worker) Metrics() *WorkerMetrics {
	return w.metrics
}
```

```go
func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case symbol, ok := <-w.updates:
			if !ok {
				return nil
			}

			if err := w.runUpdate(ctx, symbol); err != nil {
				return err
			}
		}
	}
}
```

```go
func (w *Worker) runUpdate(
	ctx context.Context,
	symbol string,
) error {
	w.metrics.RecordUpdate()

	structure, ok := w.marketStructure.Get()
	if !ok {
		return nil
	}

	routes := structure.RouteIndex.Get(symbol)

	for _, route := range routes {
		if err := ctx.Err(); err != nil {
			return err
		}

		started := time.Now()

		result := w.simulation.Simulate(
			ctx,
			route,
			w.startAmount,
		)

		w.metrics.RecordSimulation(
			time.Since(started),
		)

		if result.Opportunity == nil {
			continue
		}

		w.metrics.RecordOpportunity(
			result.Opportunity,
		)

		w.opportunities.Handle(
			result.Opportunity,
		)
	}

	return nil
}
```

## internalDDD/simulation/application/worker_metrics.go

Package: `application`

### Structs

```go
WorkerMetrics struct {
	mu sync.RWMutex

	updatesReceived uint64
	routesSimulated uint64
	opportunities   uint64

	lastUpdateAt     time.Time
	lastSimulationAt time.Time

	totalSimulationTime time.Duration

	minProfitPct   decimal.Decimal
	maxProfitPct   decimal.Decimal
	hasProfitRange bool
}
```

```go
WorkerMetricsSnapshot struct {
	UpdatesReceived     uint64
	RoutesSimulated     uint64
	Opportunities       uint64
	TotalSimulationTime time.Duration
	LastUpdateAt        time.Time
	LastSimulationAt    time.Time

	MinProfitPct   decimal.Decimal
	MaxProfitPct   decimal.Decimal
	HasProfitRange bool
}
```

### Constructors

```go
func NewWorkerMetrics() *WorkerMetrics {
	return &WorkerMetrics{}
}
```

### Methods

```go
func (m *WorkerMetrics) RecordOpportunity(
	opportunity *simulationDomain.Opportunity,
) {
	if opportunity == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.opportunities++

	profitPct := opportunity.ProfitPct

	if !m.hasProfitRange {
		m.minProfitPct = profitPct
		m.maxProfitPct = profitPct
		m.hasProfitRange = true

		return
	}

	if profitPct.LessThan(m.minProfitPct) {
		m.minProfitPct = profitPct
	}

	if profitPct.GreaterThan(m.maxProfitPct) {
		m.maxProfitPct = profitPct
	}
}
```

```go
func (m *WorkerMetrics) RecordSimulation(
	duration time.Duration,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.routesSimulated++
	m.lastSimulationAt = time.Now()
	m.totalSimulationTime += duration
}
```

```go
func (m *WorkerMetrics) RecordUpdate() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updatesReceived++
	m.lastUpdateAt = time.Now()
}
```

```go
func (m *WorkerMetrics) Snapshot() WorkerMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return WorkerMetricsSnapshot{
		UpdatesReceived:     m.updatesReceived,
		RoutesSimulated:     m.routesSimulated,
		Opportunities:       m.opportunities,
		TotalSimulationTime: m.totalSimulationTime,
		LastUpdateAt:        m.lastUpdateAt,
		LastSimulationAt:    m.lastSimulationAt,

		MinProfitPct:   m.minProfitPct,
		MaxProfitPct:   m.maxProfitPct,
		HasProfitRange: m.hasProfitRange,
	}
}
```

