package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	executionDomain "traiderBot/internalDDD/execution/domain"
)

const defaultBaseURL = "https://api.binance.com"

type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client
}

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

func (c *Client) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderResult, error) {
	mappedRequest, err := toCreateOrderRequest(request)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	params := url.Values{}

	params.Set("symbol", mappedRequest.Symbol)
	params.Set("side", mappedRequest.Side)
	params.Set("type", mappedRequest.Type)
	params.Set("newClientOrderId", mappedRequest.NewClientOrderID)

	if mappedRequest.Quantity != "" {
		params.Set("quantity", mappedRequest.Quantity)
	}

	if mappedRequest.QuoteOrderQty != "" {
		params.Set("quoteOrderQty", mappedRequest.QuoteOrderQty)
	}

	body, err := c.signedRequest(
		ctx,
		http.MethodPost,
		"/api/v3/order",
		params,
	)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	response, err := decodeJSON[orderResponse](body)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	return toOrderResult(request, response)
}

func (c *Client) GetOrderByClientID(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderResult, error) {
	if request.Edge.Symbol == "" {
		return executionDomain.OrderResult{}, fmt.Errorf(
			"symbol is empty",
		)
	}

	if request.ClientOrderID == "" {
		return executionDomain.OrderResult{}, fmt.Errorf(
			"client order id is empty",
		)
	}

	params := url.Values{}

	params.Set("symbol", request.Edge.Symbol)
	params.Set("origClientOrderId", request.ClientOrderID)

	body, err := c.signedRequest(
		ctx,
		http.MethodGet,
		"/api/v3/order",
		params,
	)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	response, err := decodeJSON[orderResponse](body)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	return toOrderResult(request, response)
}
