# execution — infrastructure

Source: `internalDDD/execution/infrastructure/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/execution/infrastructure/binance/adapter.go

Package: `binance`

### Interfaces

```go
APIClient interface {
	Execute(
		ctx context.Context,
		request executionDomain.OrderRequest,
	) (executionDomain.OrderExecutionResult, error)

	GetOrder(
		ctx context.Context,
		symbol string,
		orderID string,
	) (executionDomain.OrderExecutionResult, error)
}
```

### Structs

```go
Adapter struct {
	client APIClient
}
```

### Constructors

```go
func NewAdapter(client APIClient) (*Adapter, error) {
	if client == nil {
		return nil, ErrClientNotConfigured
	}

	return &Adapter{
		client: client,
	}, nil
}
```

### Methods

```go
func (a *Adapter) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	return a.client.Execute(
		ctx,
		request,
	)
}
```

```go
func (a *Adapter) GetOrder(
	ctx context.Context,
	symbol string,
	orderID string,
) (executionDomain.OrderExecutionResult, error) {
	return a.client.GetOrder(
		ctx,
		symbol,
		orderID,
	)
}
```

## internalDDD/execution/infrastructure/binance/client.go

Package: `binance`

### Structs

```go
Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client
}
```

### Constructors

```go
func NewClient(
	cfg Config,
	httpClient *http.Client,
) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("binance api key is empty")
	}

	if cfg.APISecret == "" {
		return nil, fmt.Errorf("binance api secret is empty")
	}

	if httpClient == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	return &Client{
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		baseURL:   cfg.BaseURL,
		http:      httpClient,
	}, nil
}
```

## internalDDD/execution/infrastructure/binance/config.go

Package: `binance`

### Structs

```go
Config struct {
	APIKey    string
	APISecret string
	BaseURL   string
}
```

### Functions

```go
func LoadConfig(useTestnet bool) (Config, error) {
	_ = godotenv.Load()

	config := Config{}

	if useTestnet {
		config.APIKey = os.Getenv("TESTNET_API_KEY")
		config.APISecret = os.Getenv("TESTNET_SECRET_KEY")
		config.BaseURL = "https://testnet.binance.vision"
	} else {
		config.APIKey = os.Getenv("API_KEY")
		config.APISecret = os.Getenv("SECRET_KEY")
		config.BaseURL = "https://api.binance.com"
	}

	if config.APIKey == "" {
		return Config{}, fmt.Errorf(
			"binance api key is not configured",
		)
	}

	if config.APISecret == "" {
		return Config{}, fmt.Errorf(
			"binance secret key is not configured",
		)
	}

	return config, nil
}
```

## internalDDD/execution/infrastructure/binance/errors.go

Package: `binance`

## internalDDD/execution/infrastructure/binance/http_client.go

Package: `binance`

### Constructors

```go
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}
```

## internalDDD/execution/infrastructure/binance/json.go

Package: `binance`

### Functions

```go
func decodeJSON[T any](
	body []byte,
) (T, error) {
	var result T

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf(
			"decode binance response: %w",
			err,
		)
	}

	return result, nil
}
```

## internalDDD/execution/infrastructure/binance/order.go

Package: `binance`

### Methods

```go
func (c *Client) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if request.Edge.Symbol == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"symbol is empty",
		)
	}

	if request.Amount.IsZero() || request.Amount.IsNegative() {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"amount must be positive",
		)
	}

	if request.ClientOrderID == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"client order id is empty",
		)
	}

	side, err := binanceSide(request.Edge.Direction)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	params := url.Values{}

	params.Set("symbol", request.Edge.Symbol)
	params.Set("side", side)
	params.Set("type", "MARKET")
	params.Set("newClientOrderId", request.ClientOrderID)

	if request.Edge.Direction == marketDomain.Buy {
		params.Set(
			"quoteOrderQty",
			request.Amount.String(),
		)
	} else {
		params.Set(
			"quantity",
			request.Amount.String(),
		)
	}

	body, err := c.signedRequest(
		ctx,
		http.MethodPost,
		"/api/v3/order",
		params,
	)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	response, err := decodeJSON[orderResponse](body)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	return c.toExecutionResult(
		request,
		response,
	)
}
```

```go
func (c *Client) GetOrder(
	ctx context.Context,
	symbol string,
	clientOrderID string,
) (executionDomain.OrderExecutionResult, error) {
	if symbol == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"symbol is empty",
		)
	}

	if clientOrderID == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"client order id is empty",
		)
	}

	params := url.Values{}

	params.Set("symbol", symbol)
	params.Set("origClientOrderId", clientOrderID)

	body, err := c.signedRequest(
		ctx,
		http.MethodGet,
		"/api/v3/order",
		params,
	)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	response, err := decodeJSON[orderResponse](body)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	request := executionDomain.OrderRequest{
		Edge: marketDomain.GraphEdge{
			Symbol: symbol,
		},
		ClientOrderID: clientOrderID,
	}

	return c.toExecutionResult(
		request,
		response,
	)
}
```

## internalDDD/execution/infrastructure/binance/order_mapping.go

Package: `binance`

### Functions

```go
func binanceSide(
	direction marketDomain.TradeDirection,
) (string, error) {
	switch direction {
	case marketDomain.Buy:
		return "BUY", nil

	case marketDomain.Sell:
		return "SELL", nil

	default:
		return "", fmt.Errorf(
			"unsupported trade direction: %v",
			direction,
		)
	}
}
```

```go
func mapOrderDirection(
	side string,
) (marketDomain.TradeDirection, error) {
	switch side {
	case "BUY":
		return marketDomain.Buy, nil

	case "SELL":
		return marketDomain.Sell, nil

	default:
		return 0, fmt.Errorf(
			"unknown Binance order side: %s",
			side,
		)
	}
}
```

```go
func mapOrderStatus(
	status string,
) (executionDomain.OrderStatus, error) {
	switch status {
	case "NEW":
		return executionDomain.OrderSubmitted, nil

	case "PARTIALLY_FILLED":
		return executionDomain.OrderPartiallyFilled, nil

	case "FILLED":
		return executionDomain.OrderFilled, nil

	case "CANCELED", "EXPIRED", "PENDING_CANCEL":
		return executionDomain.OrderCancelled, nil

	case "REJECTED":
		return executionDomain.OrderFailed, nil

	default:
		return executionDomain.OrderFailed, fmt.Errorf(
			"unsupported binance order status: %s",
			status,
		)
	}
}
```

```go
func millisToTime(
	value int64,
) time.Time {
	if value <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(value)
}
```

```go
func parseOrderFee(
	fills []orderFill,
) (decimal.Decimal, string, error) {
	totalFee := decimal.Zero
	feeAsset := ""

	for _, fill := range fills {
		commission, err := decimal.NewFromString(
			fill.Commission,
		)
		if err != nil {
			return decimal.Zero,
				"",
				fmt.Errorf(
					"parse commission %q: %w",
					fill.Commission,
					err,
				)
		}

		if commission.IsNegative() {
			return decimal.Zero,
				"",
				fmt.Errorf(
					"commission must not be negative: %s",
					commission,
				)
		}

		if commission.IsZero() {
			continue
		}

		if fill.CommissionAsset == "" {
			return decimal.Zero,
				"",
				fmt.Errorf(
					"commission asset is empty",
				)
		}

		if feeAsset == "" {
			feeAsset = fill.CommissionAsset
		} else if feeAsset != fill.CommissionAsset {
			return decimal.Zero,
				"",
				fmt.Errorf(
					"multiple commission assets are not supported: %s and %s",
					feeAsset,
					fill.CommissionAsset,
				)
		}

		totalFee = totalFee.Add(commission)
	}

	return totalFee, feeAsset, nil
}
```

### Methods

```go
func (c *Client) toExecutionResult(
	request executionDomain.OrderRequest,
	response orderResponse,
) (executionDomain.OrderExecutionResult, error) {
	executedQty, err := decimal.NewFromString(
		response.ExecutedQty,
	)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"parse executed qty: %w",
			err,
		)
	}

	quoteQty, err := decimal.NewFromString(
		response.CumulativeQuoteQty,
	)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"parse cumulative quote qty: %w",
			err,
		)
	}

	status, err := mapOrderStatus(response.Status)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	direction, err := mapOrderDirection(response.Side)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	fee, feeAsset, err := parseOrderFee(response.Fills)
	if err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	var (
		executedAmount decimal.Decimal
		receivedAmount decimal.Decimal
	)

	switch direction {
	case marketDomain.Buy:
		executedAmount = quoteQty
		receivedAmount = executedQty

	case marketDomain.Sell:
		executedAmount = executedQty
		receivedAmount = quoteQty

	default:
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"unsupported trade direction: %v",
			direction,
		)
	}

	createdAt := millisToTime(
		response.TransactTime,
	)

	filledAt := time.Time{}

	if status == executionDomain.OrderFilled {
		filledAt = millisToTime(
			response.UpdateTime,
		)
	}

	return executionDomain.OrderExecutionResult{
		StepIndex: request.StepIndex,

		Symbol:    response.Symbol,
		Direction: direction,

		ClientOrderID:   response.ClientOrderID,
		ExchangeOrderID: strconv.FormatInt(response.OrderID, 10),

		RequestedAmount:   request.Amount,
		ExpectedAmountOut: request.ExpectedAmountOut,
		ExecutedAmount:    executedAmount,
		ReceivedAmount:    receivedAmount,

		Fee:      fee,
		FeeAsset: feeAsset,

		Status: status,

		CreatedAt:   createdAt,
		SubmittedAt: createdAt,
		FilledAt:    filledAt,
	}, nil
}
```

## internalDDD/execution/infrastructure/binance/order_response.go

Package: `binance`

### Structs

```go
orderFill struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}
```

```go
orderResponse struct {
	Symbol             string `json:"symbol"`
	OrderID            int64  `json:"orderId"`
	ClientOrderID      string `json:"clientOrderId"`
	TransactTime       int64  `json:"transactTime"`
	UpdateTime         int64  `json:"updateTime"`
	Status             string `json:"status"`
	Side               string `json:"side"`
	ExecutedQty        string `json:"executedQty"`
	CumulativeQuoteQty string `json:"cummulativeQuoteQty"`

	Fills []orderFill `json:"fills"`
}
```

## internalDDD/execution/infrastructure/binance/requests.go

Package: `binance`

### Functions

```go
func sign(
	payload string,
	secret string,
) string {
	mac := hmac.New(
		sha256.New,
		[]byte(secret),
	)

	_, _ = mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}
```

### Methods

```go
func (c *Client) signedRequest(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}

	params.Set(
		"timestamp",
		strconv.FormatInt(time.Now().UnixMilli(), 10),
	)

	query := params.Encode()

	signature := sign(
		query,
		c.apiSecret,
	)

	query += "&signature=" + signature

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		c.baseURL+path+"?"+query,
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"X-MBX-APIKEY",
		c.apiKey,
	)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"binance api error: status=%d body=%s",
			resp.StatusCode,
			string(body),
		)
	}

	return body, nil
}
```

## internalDDD/execution/infrastructure/paper/executor.go

Package: `paper`

### Structs

```go
Executor struct{}
```

### Constructors

```go
func NewExecutor() *Executor {
	return &Executor{}
}
```

### Methods

```go
func (e *Executor) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	if request.Edge.Symbol == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"symbol must not be empty",
		)
	}

	if request.Amount.LessThanOrEqual(decimal.Zero) {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"amount must be greater than zero",
		)
	}

	if request.ExpectedAmountOut.LessThanOrEqual(decimal.Zero) {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"expected amount out must be greater than zero",
		)
	}

	if request.ClientOrderID == "" {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"client order ID must not be empty",
		)
	}

	now := time.Now()

	return executionDomain.NewOrderExecutionResult(
		request,
		request.Amount,
		request.ExpectedAmountOut,
		executionDomain.OrderFilled,
		request.ClientOrderID,
		request.ClientOrderID,
		now,
		now,
		now,
	), nil
}
```

