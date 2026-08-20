package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"traiderBot/internal/arbitrage"
)

type CreateOrderResponse struct {
	Symbol string `json:"symbol"`

	OrderID int64 `json:"orderId"`

	Status string `json:"status"`

	Side string `json:"side"`

	Price string `json:"price"`

	ExecutedQty string `json:"executedQty"`

	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
}

func (c *Client) PlaceLimitIOC(
	ctx context.Context,
	step arbitrage.StepResult,
) (*CreateOrderResponse, error) {

	values := url.Values{}

	values.Set("symbol", step.Symbol)

	values.Set("side", step.Side)

	values.Set("type", "LIMIT")

	values.Set("timeInForce", "IOC")

	values.Set("quantity", strconv.FormatFloat(
		step.Quantity,
		'f',
		-1,
		64,
	))

	values.Set("price", strconv.FormatFloat(
		step.AvgPrice,
		'f',
		-1,
		64,
	))

	var resp CreateOrderResponse

	err := c.do(
		ctx,
		http.MethodPost,
		"/api/v3/order",
		values,
		true,
		&resp,
	)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) PlaceMarket(
	ctx context.Context,
	symbol string,
	side string,
	quantity float64,
) (*CreateOrderResponse, error) {

	values := url.Values{}

	values.Set("symbol", symbol)

	values.Set("side", side)

	values.Set("type", "MARKET")

	values.Set("quantity", strconv.FormatFloat(
		quantity,
		'f',
		-1,
		64,
	))

	var resp CreateOrderResponse

	err := c.do(
		ctx,
		http.MethodPost,
		"/api/v3/order",
		values,
		true,
		&resp,
	)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}