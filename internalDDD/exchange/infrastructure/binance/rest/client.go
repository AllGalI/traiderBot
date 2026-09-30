package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	binance "traiderBot/internalDDD/exchange/infrastructure/binance"
)

type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client
}

func NewClient(
	cfg binance.Config,
	httpClient *http.Client,
) (*Client, error) {

	if cfg.APIKey == "" {
		return nil, fmt.Errorf(
			"binance api key is empty",
		)
	}

	if cfg.APISecret == "" {
		return nil, fmt.Errorf(
			"binance api secret is empty",
		)
	}

	if cfg.RESTBaseURL == "" {
		return nil, fmt.Errorf(
			"binance base url is empty",
		)
	}

	if httpClient == nil {
		return nil, fmt.Errorf(
			"http client is nil",
		)
	}

	return &Client{
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		baseURL:   cfg.RESTBaseURL,
		http:      httpClient,
	}, nil
}

func (c *Client) GetAccount(
	ctx context.Context,
) ([]byte, error) {

	return c.SignedRequest(
		ctx,
		http.MethodGet,
		"/api/v3/account",
		nil,
	)
}

func (c *Client) Execute(
	ctx context.Context,
	request exchangeDomain.OrderRequest,
) (
	exchangeDomain.OrderResult,
	error,
) {

	params, err := buildCreateOrderParams(
		request,
	)

	if err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	body, err := c.SignedRequest(
		ctx,
		http.MethodPost,
		"/api/v3/order",
		params,
	)

	if err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	response, err := decodeOrderResponse(body)

	if err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	return mapOrderResult(
		request,
		response,
	)
}

func (c *Client) GetOrderByClientID(
	ctx context.Context,
	request exchangeDomain.OrderRequest,
) (
	exchangeDomain.OrderResult,
	error,
) {

	params := url.Values{}

	params.Set(
		"symbol",
		request.Edge.Symbol,
	)

	params.Set(
		"origClientOrderId",
		request.ClientOrderID,
	)

	body, err := c.SignedRequest(
		ctx,
		http.MethodGet,
		"/api/v3/order",
		params,
	)

	if err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	response, err := decodeOrderResponse(body)

	if err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	return mapOrderResult(
		request,
		response,
	)
}

func (c *Client) GetAccountCommission(
	ctx context.Context,
	symbol string,
) ([]byte, error) {

	params := url.Values{}

	params.Set(
		"symbol",
		symbol,
	)

	return c.SignedRequest(
		ctx,
		http.MethodGet,
		"/api/v3/account/commission",
		params,
	)
}