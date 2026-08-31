# simulation — infrastructure

Source: `internalDDD/simulation/infrastructure/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/simulation/infrastructure/binance/constants.go

Package: `binance`

## internalDDD/simulation/infrastructure/binance/fee_provider.go

Package: `binance`

### Structs

```go
FeeProvider struct {
	client  *http.Client
	baseURL string
	apiKey  string
	secret  string
}
```

```go
commissionResponse struct {
	Symbol string `json:"symbol"`

	StandardCommission struct {
		Maker string `json:"maker"`
		Taker string `json:"taker"`
	} `json:"standardCommission"`
}
```

### Constructors

```go
func NewFeeProvider() *FeeProvider {
	return &FeeProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
		apiKey:  os.Getenv("API_KEY"),
		secret:  os.Getenv("SECRET_KEY"),
	}
}
```

### Methods

```go
func (p *FeeProvider) Get(
	ctx context.Context,
	symbol string,
) (simulationApplication.FeeConfig, error) {
	if p.apiKey == "" {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("binance api key is not configured")
	}

	if p.secret == "" {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("binance secret key is not configured")
	}

	params := url.Values{}

	params.Set("symbol", symbol)
	params.Set(
		"timestamp",
		strconv.FormatInt(time.Now().UnixMilli(), 10),
	)

	query := params.Encode()
	signature := p.sign(query)

	reqURL := fmt.Sprintf(
		"%s%s?%s&signature=%s",
		p.baseURL,
		accountCommissionPath,
		query,
		signature,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("create commission request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("request commission: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return simulationApplication.FeeConfig{},
			fmt.Errorf(
				"binance commission request returned status %d",
				resp.StatusCode,
			)
	}

	var dto commissionResponse

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("decode commission response: %w", err)
	}

	taker, err := decimal.NewFromString(
		dto.StandardCommission.Taker,
	)
	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("parse taker commission: %w", err)
	}

	return simulationApplication.FeeConfig{
		Rate: taker,
	}, nil
}
```

```go
func (p *FeeProvider) sign(query string) string {
	mac := hmac.New(
		sha256.New,
		[]byte(p.secret),
	)

	_, _ = mac.Write([]byte(query))

	return hex.EncodeToString(mac.Sum(nil))
}
```

## internalDDD/simulation/infrastructure/fee_config.go

Package: `infrastructure`

### Structs

```go
FeeConfigProvider struct {
	fees map[string]simulationApplication.FeeConfig
}
```

### Constructors

```go
func NewFeeConfigProvider(
	fees map[string]simulationApplication.FeeConfig,
) *FeeConfigProvider {
	copied := make(map[string]simulationApplication.FeeConfig, len(fees))

	for symbol, fee := range fees {
		copied[symbol] = fee
	}

	return &FeeConfigProvider{
		fees: copied,
	}
}
```

### Functions

```go
func DefaultFeeConfigProvider(
	symbols []string,
	rate decimal.Decimal,
) *FeeConfigProvider {
	fees := make(map[string]simulationApplication.FeeConfig, len(symbols))

	for _, symbol := range symbols {
		fees[symbol] = simulationApplication.FeeConfig{
			Rate: rate,
		}
	}

	return NewFeeConfigProvider(fees)
}
```

### Methods

```go
func (p *FeeConfigProvider) Get(
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	fee, ok := p.fees[symbol]
	return fee, ok
}
```

## internalDDD/simulation/infrastructure/static_fee_provider.go

Package: `infrastructure`

### Structs

```go
StaticFeeProvider struct {
	rate decimal.Decimal
}
```

### Constructors

```go
func NewStaticFeeProvider(
	rate decimal.Decimal,
) *StaticFeeProvider {
	return &StaticFeeProvider{
		rate: rate,
	}
}
```

### Methods

```go
func (p *StaticFeeProvider) Get(
	symbol string,
) (simulationApplication.FeeConfig, bool) {
	return simulationApplication.FeeConfig{
		Rate: p.rate,
	}, false
}
```

