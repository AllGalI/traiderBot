package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
)

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
