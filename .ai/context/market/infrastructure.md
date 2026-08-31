# market — infrastructure

Source: `internalDDD/market/infrastructure/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/market/infrastructure/binance/constants.go

Package: `binance`

## internalDDD/market/infrastructure/binance/exchange.go

Package: `binance`

### Structs

```go
exchangeInfoDTO struct {
	Symbols []symbolDTO `json:"symbols"`
}
```

```go
filterDTO struct {
	FilterType  string `json:"filterType"`
	MinQty      string `json:"minQty,omitempty"`
	MaxQty      string `json:"maxQty,omitempty"`
	StepSize    string `json:"stepSize,omitempty"`
	MinNotional string `json:"minNotional,omitempty"`
	MinPrice    string `json:"minPrice,omitempty"`
	MaxPrice    string `json:"maxPrice,omitempty"`
	TickSize    string `json:"tickSize,omitempty"`
}
```

```go
symbolDTO struct {
	Symbol     string      `json:"symbol"`
	BaseAsset  string      `json:"baseAsset"`
	QuoteAsset string      `json:"quoteAsset"`
	Status     string      `json:"status"`
	Filters    []filterDTO `json:"filters"`
}
```

## internalDDD/market/infrastructure/binance/exchange_mapper.go

Package: `binance`

### Functions

```go
func mapSymbol(dto symbolDTO) (domain.Symbol, error) {
	if dto.Symbol == "" {
		return domain.Symbol{}, fmt.Errorf("symbol is empty")
	}

	if dto.BaseAsset == "" {
		return domain.Symbol{}, fmt.Errorf(
			"symbol %q has empty base asset",
			dto.Symbol,
		)
	}

	if dto.QuoteAsset == "" {
		return domain.Symbol{}, fmt.Errorf(
			"symbol %q has empty quote asset",
			dto.Symbol,
		)
	}

	constraints, err := mapSymbolConstraints(dto.Filters)
	if err != nil {
		return domain.Symbol{}, fmt.Errorf(
			"map symbol %q constraints: %w",
			dto.Symbol,
			err,
		)
	}

	return domain.Symbol{
		Name:        dto.Symbol,
		BaseAsset:   dto.BaseAsset,
		QuoteAsset:  dto.QuoteAsset,
		Constraints: constraints,
	}, nil
}
```

```go
func mapSymbolConstraints(
	filters []filterDTO,
) (domain.SymbolConstraints, error) {
	var constraints domain.SymbolConstraints

	for _, filter := range filters {
		switch filter.FilterType {
		case "LOT_SIZE":
			minQty, err := parseDecimal(filter.MinQty)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse minQty: %w",
					err,
				)
			}

			maxQty, err := parseDecimal(filter.MaxQty)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse maxQty: %w",
					err,
				)
			}

			stepSize, err := parseDecimal(filter.StepSize)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse stepSize: %w",
					err,
				)
			}

			constraints.MinQty = minQty
			constraints.MaxQty = maxQty
			constraints.StepSize = stepSize

		case "MIN_NOTIONAL":
			minNotional, err := parseDecimal(filter.MinNotional)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse minNotional: %w",
					err,
				)
			}

			constraints.MinNotional = minNotional

		case "PRICE_FILTER":
			tickSize, err := parseDecimal(filter.TickSize)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse tickSize: %w",
					err,
				)
			}

			constraints.PriceStep = tickSize
		}
	}

	return constraints, nil
}
```

```go
func parseDecimal(value string) (decimal.Decimal, error) {
	if value == "" {
		return decimal.Zero, nil
	}

	result, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf(
			"parse %q as decimal: %w",
			value,
			err,
		)
	}

	return result, nil
}
```

## internalDDD/market/infrastructure/binance/exchange_provider.go

Package: `binance`

### Structs

```go
ExchangeProvider struct {
	client  *http.Client
	baseURL string
}
```

### Constructors

```go
func NewExchangeProvider() *ExchangeProvider {
	return &ExchangeProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
	}
}
```

### Methods

```go
func (p *ExchangeProvider) GetSymbols(
	ctx context.Context,
) ([]domain.Symbol, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		p.baseURL+exchangeInfoPath,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create exchange info request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto exchangeInfoDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf("decode exchange info response: %w", err)
	}

	symbols := make([]domain.Symbol, 0, len(dto.Symbols))

	for _, symbolDTO := range dto.Symbols {
		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbol, err := mapSymbol(symbolDTO)
		if err != nil {

			continue
		}

		symbols = append(symbols, symbol)
	}

	return symbols, nil
}
```

## internalDDD/market/infrastructure/binance/orderbook_dto.go

Package: `binance`

### Structs

```go
orderBookDTO struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}
```

## internalDDD/market/infrastructure/binance/orderbook_mapper.go

Package: `binance`

### Functions

```go
func mapOrderBook(dto orderBookDTO, symbol string) (domain.OrderBook, error) {
	bids, err := mapPriceLevels(dto.Bids)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("map bids: %w", err)
	}

	asks, err := mapPriceLevels(dto.Asks)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("map asks: %w", err)
	}

	return domain.OrderBook{
		Symbol:         symbol,
		LastUpdateID:   dto.LastUpdateID,
		LastUpdateTime: time.Now(),
		Bids:           bids,
		Asks:           asks,
	}, nil
}
```

```go
func mapPriceLevels(levels [][]string) ([]domain.PriceLevel, error) {
	priceLevels := make([]domain.PriceLevel, 0, len(levels))

	for i, level := range levels {
		if len(level) != 2 {
			return nil, fmt.Errorf(
				"invalid price level at index %d: expected 2 values, got %d",
				i,
				len(level),
			)
		}

		price, err := decimal.NewFromString(level[0])
		if err != nil {
			return nil, fmt.Errorf(
				"parse price at index %d: %w",
				i,
				err,
			)
		}

		qty, err := decimal.NewFromString(level[1])
		if err != nil {
			return nil, fmt.Errorf(
				"parse quantity at index %d: %w",
				i,
				err,
			)
		}

		priceLevels = append(priceLevels, domain.PriceLevel{
			Price: price,
			Qty:   qty,
		})
	}

	return priceLevels, nil
}
```

## internalDDD/market/infrastructure/binance/orderbook_provider.go

Package: `binance`

### Structs

```go
OrderBookProvider struct {
	client  *http.Client
	baseURL string
}
```

### Constructors

```go
func NewOrderBookProvider() *OrderBookProvider {
	return &OrderBookProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
	}
}
```

### Methods

```go
func (p *OrderBookProvider) Get(
	ctx context.Context,
	symbol string,
	depth int,
) (domain.OrderBook, error) {
	if depth != 5 && depth != 10 && depth != 20 {
		return domain.OrderBook{}, fmt.Errorf(
			"unsupported order book depth: %d",
			depth,
		)
	}

	reqURL := fmt.Sprintf(
		"%s/api/v3/depth?symbol=%s&limit=%d",
		p.baseURL,
		url.QueryEscape(symbol),
		depth,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"create request: %w",
			err,
		)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"request order book: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.OrderBook{}, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto orderBookDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"decode order book response: %w",
			err,
		)
	}

	orderBook, err := mapOrderBook(dto, symbol)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"map order book: %w",
			err,
		)
	}

	return orderBook, nil
}
```

## internalDDD/market/infrastructure/binance/orderbook_stream.go

Package: `binance`

### Structs

```go
OrderBookStream struct {
}
```

```go
combinedEventDTO struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}
```

### Constructors

```go
func NewOrderBookStream() *OrderBookStream {
	return &OrderBookStream{}
}
```

### Functions

```go
func buildWebSocketURL(symbols []string, depth int) string {
	streams := make([]string, 0, len(symbols))

	for _, symbol := range symbols {
		streams = append(
			streams,
			fmt.Sprintf(
				"%s@depth%d@100ms",
				strings.ToLower(symbol),
				depth,
			),
		)
	}

	return fmt.Sprintf(
		"%s/stream?streams=%s",
		wsBaseURL,
		strings.Join(streams, "/"),
	)
}
```

```go
func symbolFromStream(stream string) (string, error) {
	if stream == "" {
		return "", fmt.Errorf("empty stream")
	}

	parts := strings.Split(stream, "@")

	if len(parts) < 2 || parts[0] == "" {
		return "", fmt.Errorf("invalid stream: %q", stream)
	}

	return strings.ToUpper(parts[0]), nil
}
```

### Methods

```go
func (s *OrderBookStream) Subscribe(
	ctx context.Context,
	symbols []string,
	depth int,
	handler application.OrderBookUpdateHandler,
) error {
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	if handler == nil {
		return fmt.Errorf("update handler is nil")
	}

	if depth != 5 && depth != 10 && depth != 20 {
		return fmt.Errorf("unsupported order book depth: %d", depth)
	}

	url := buildWebSocketURL(symbols, depth)

	return s.connectAndListen(ctx, url, handler)
}
```

```go
func (s *OrderBookStream) connectAndListen(
	ctx context.Context,
	url string,
	handler application.OrderBookUpdateHandler,
) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	log.Printf("WebSocket: dialing %s", url)

	conn, _, err := dialer.DialContext(ctx, url, nil)

	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	log.Println("WebSocket: connected")

	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read websocket message: %w", err)
		}

		var event combinedEventDTO

		if err := json.Unmarshal(message, &event); err != nil {
			return fmt.Errorf("decode combined event: %w", err)
		}

		symbol, err := symbolFromStream(event.Stream)
		if err != nil {
			return fmt.Errorf("parse stream symbol: %w", err)
		}

		var dto orderBookDTO

		if err := json.Unmarshal(event.Data, &dto); err != nil {
			return fmt.Errorf(
				"decode order book update for %s: %w",
				symbol,
				err,
			)
		}

		orderBook, err := mapOrderBook(dto, symbol)
		if err != nil {
			return fmt.Errorf(
				"map order book update for %s: %w",
				symbol,
				err,
			)
		}

		handler(orderBook)
	}
}
```

## internalDDD/market/infrastructure/binance/symbol_provider.go

Package: `binance`

### Structs

```go
SymbolProvider struct {
	client  *http.Client
	baseURL string
}
```

### Constructors

```go
func NewSymbolProvider() *SymbolProvider {
	return &SymbolProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
	}
}
```

### Methods

```go
func (p *SymbolProvider) Get(
	ctx context.Context,
	symbols []string,
) ([]domain.Symbol, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols provided")
	}

	requested := make(map[string]struct{}, len(symbols))

	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))

		if symbol == "" {
			continue
		}

		requested[symbol] = struct{}{}
	}

	if len(requested) == 0 {
		return nil, fmt.Errorf("no valid symbols provided")
	}

	symbolList := make([]string, 0, len(requested))

	for symbol := range requested {
		symbolList = append(symbolList, symbol)
	}

	symbolsJSON, err := json.Marshal(symbolList)
	if err != nil {
		return nil, fmt.Errorf("encode symbols: %w", err)
	}

	reqURL := fmt.Sprintf(
		"%s%s?symbols=%s",
		p.baseURL,
		exchangeInfoPath,
		url.QueryEscape(string(symbolsJSON)),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto exchangeInfoDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf(
			"decode exchange info response: %w",
			err,
		)
	}

	result := make([]domain.Symbol, 0, len(dto.Symbols))

	for _, symbolDTO := range dto.Symbols {
		symbolName := strings.ToUpper(symbolDTO.Symbol)

		if _, ok := requested[symbolName]; !ok {
			continue
		}

		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbol, err := mapSymbol(symbolDTO)
		if err != nil {
			return nil, fmt.Errorf(
				"map symbol %q: %w",
				symbolDTO.Symbol,
				err,
			)
		}

		result = append(result, symbol)
	}

	return result, nil
}
```

```go
func (p *SymbolProvider) GetSymbols(
	ctx context.Context,
) ([]domain.Symbol, error) {
	reqURL := fmt.Sprintf(
		"%s%s",
		p.baseURL,
		exchangeInfoPath,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto exchangeInfoDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf(
			"decode exchange info response: %w",
			err,
		)
	}

	result := make([]domain.Symbol, 0, len(dto.Symbols))

	for _, symbolDTO := range dto.Symbols {
		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbol, err := mapSymbol(symbolDTO)
		if err != nil {
			return nil, fmt.Errorf(
				"map symbol %q: %w",
				symbolDTO.Symbol,
				err,
			)
		}

		result = append(result, symbol)
	}

	return result, nil
}
```

## internalDDD/market/infrastructure/cache/market_cache.go

Package: `cache`

### Structs

```go
MarketCache struct {
	path string
}
```

### Constructors

```go
func NewMarketCache(path string) *MarketCache {
	return &MarketCache{
		path: path,
	}
}
```

### Methods

```go
func (c *MarketCache) Get(
	ctx context.Context,
) (domain.MarketStructure, bool, error) {
	select {
	case <-ctx.Done():
		return domain.MarketStructure{}, false, ctx.Err()
	default:
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.MarketStructure{}, false, nil
		}

		return domain.MarketStructure{}, false, fmt.Errorf(
			"read cache file: %w",
			err,
		)
	}

	var structure domain.MarketStructure

	if err := json.Unmarshal(data, &structure); err != nil {
		return domain.MarketStructure{}, false, fmt.Errorf(
			"decode market structure cache: %w",
			err,
		)
	}

	return structure, true, nil
}
```

```go
func (c *MarketCache) Set(
	ctx context.Context,
	structure domain.MarketStructure,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := json.MarshalIndent(structure, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"encode market structure cache: %w",
			err,
		)
	}

	if err := os.WriteFile(
		c.path,
		data,
		0644,
	); err != nil {
		return fmt.Errorf(
			"write cache file: %w",
			err,
		)
	}

	return nil
}
```

## internalDDD/market/infrastructure/market_structure_store.go

Package: `infrastructure`

### Structs

```go
InMemoryMarketStructureStore struct {
	mu        sync.RWMutex
	structure domain.MarketStructure
	exists    bool
}
```

### Constructors

```go
func NewInMemoryMarketStructureStore() *InMemoryMarketStructureStore {
	return &InMemoryMarketStructureStore{}
}
```

### Methods

```go
func (s *InMemoryMarketStructureStore) Get() (*domain.MarketStructure, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.exists {
		return &domain.MarketStructure{}, false
	}

	return &s.structure, true
}
```

```go
func (s *InMemoryMarketStructureStore) Set(
	structure *domain.MarketStructure,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.structure = *structure
	s.exists = true
}
```

## internalDDD/market/infrastructure/orderbook_store.go

Package: `infrastructure`

### Structs

```go
OrderBookStore struct {
	mu         sync.RWMutex
	orderBooks map[string]domain.OrderBook
}
```

### Constructors

```go
func NewOrderBookStore() *OrderBookStore {
	return &OrderBookStore{
		orderBooks: make(map[string]domain.OrderBook),
	}
}
```

### Methods

```go
func (s *OrderBookStore) Get(
	symbol string,
) (domain.OrderBook, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderBook, ok := s.orderBooks[strings.ToUpper(symbol)]

	return orderBook, ok
}
```

```go
func (s *OrderBookStore) Set(
	orderBook domain.OrderBook,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orderBooks[strings.ToUpper(orderBook.Symbol)] = orderBook
}
```

```go
func (s *OrderBookStore) Symbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]string, 0, len(s.orderBooks))

	for symbol := range s.orderBooks {
		symbols = append(symbols, symbol)
	}

	sort.Strings(symbols)

	return symbols
}
```

