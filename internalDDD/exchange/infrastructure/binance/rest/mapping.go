package rest

import (
	"encoding/json"
	"fmt"
	"strconv"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"net/url"

	"github.com/shopspring/decimal"
)

type createOrderRequest struct {
	Symbol           string
	Side             string
	Type             string
	Quantity         string
	QuoteOrderQty    string
	NewClientOrderID string
}

func buildCreateOrderParams(
	request exchangeDomain.OrderRequest,
) (
	url.Values,
	error,
) {

	mapped, err := mapCreateOrderRequest(
		request,
	)

	if err != nil {
		return nil, err
	}

	params := url.Values{}

	params.Set(
		"symbol",
		mapped.Symbol,
	)

	params.Set(
		"side",
		mapped.Side,
	)

	params.Set(
		"type",
		mapped.Type,
	)

	params.Set(
		"newClientOrderId",
		mapped.NewClientOrderID,
	)

	if mapped.Quantity != "" {

		params.Set(
			"quantity",
			mapped.Quantity,
		)
	}

	if mapped.QuoteOrderQty != "" {

		params.Set(
			"quoteOrderQty",
			mapped.QuoteOrderQty,
		)
	}

	return params, nil
}

func mapCreateOrderRequest(
	request exchangeDomain.OrderRequest,
) (
	createOrderRequest,
	error,
) {

	side, err := mapSide(
		request.Edge.Direction,
	)

	if err != nil {
		return createOrderRequest{}, err
	}

	result := createOrderRequest{

		Symbol: request.Edge.Symbol,

		Side: side,

		Type: "MARKET",

		NewClientOrderID: request.ClientOrderID,
	}

	switch request.Edge.Direction {

	case marketDomain.Buy:

		// Покупаем BTC за USDT
		// Binance принимает quoteOrderQty

		result.QuoteOrderQty =
			request.Amount.String()

	case marketDomain.Sell:

		// Продаём BTC
		// Binance принимает quantity

		result.Quantity =
			request.Amount.String()

	default:

		return createOrderRequest{},
			fmt.Errorf(
				"unsupported trade direction: %v",
				request.Edge.Direction,
			)
	}

	return result, nil
}

func mapSide(
	direction marketDomain.TradeDirection,
) (
	string,
	error,
) {

	switch direction {

	case marketDomain.Buy:

		return "BUY", nil

	case marketDomain.Sell:

		return "SELL", nil

	default:

		return "",
			fmt.Errorf(
				"unsupported trade direction: %v",
				direction,
			)
	}
}

func decodeOrderResponse(
	body []byte,
) (
	orderResponse,
	error,
) {

	var response orderResponse

	if err := json.Unmarshal(
		body,
		&response,
	); err != nil {

		return orderResponse{},
			fmt.Errorf(
				"decode order response: %w",
				err,
			)
	}

	return response, nil
}

func mapOrderResult(
	request exchangeDomain.OrderRequest,
	response orderResponse,
) (
	exchangeDomain.OrderResult,
	error,
) {

	executedQty, err := decimal.NewFromString(
		response.ExecutedQty,
	)

	if err != nil {
		return exchangeDomain.OrderResult{},
			fmt.Errorf(
				"parse executed qty: %w",
				err,
			)
	}

	quoteQty, err := decimal.NewFromString(
		response.CumulativeQuoteQty,
	)

	if err != nil {
		return exchangeDomain.OrderResult{},
			fmt.Errorf(
				"parse quote qty: %w",
				err,
			)
	}

	fee, feeAsset, err := parseFee(
		response.Fills,
	)

	if err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	var executedAmount decimal.Decimal
	var receivedAmount decimal.Decimal

	switch request.Edge.Direction {

	case marketDomain.Buy:

		executedAmount = quoteQty
		receivedAmount = executedQty

	case marketDomain.Sell:

		executedAmount = executedQty
		receivedAmount = quoteQty

	default:

		return exchangeDomain.OrderResult{},
			fmt.Errorf(
				"unsupported direction",
			)
	}

	return exchangeDomain.OrderResult{

		Status: mapStatus(
			response.Status,
		),

		ExchangeOrderID: strconv.FormatInt(
			response.OrderID,
			10,
		),

		ClientOrderID: response.ClientOrderID,

		ExecutedAmount: executedAmount,

		ReceivedAmount: receivedAmount,

		Fee: fee,

		FeeAsset: feeAsset,
	}, nil
}

func parseFee(
	fills []fill,
)(
	decimal.Decimal,
	string,
	error,
){

	total := decimal.Zero
	asset := ""


	for _, f := range fills {


		value, err :=
			decimal.NewFromString(
				f.Commission,
			)

		if err != nil {
			return decimal.Zero,
				"",
				err
		}


		if value.IsZero() {
			continue
		}


		if asset == "" {
			asset = f.CommissionAsset
		}


		total =
			total.Add(value)
	}


	return total,asset,nil
}

func mapStatus(
	status string,
)(
	exchangeDomain.OrderStatus,
){

	switch status {

	case "NEW":
		return exchangeDomain.OrderStatusPending

	case "PARTIALLY_FILLED":
		return exchangeDomain.OrderStatusPartiallyFilled

	case "FILLED":
		return exchangeDomain.OrderStatusFilled

	case "REJECTED":
		return exchangeDomain.OrderStatusRejected

	case "CANCELED":
		return exchangeDomain.OrderStatusCancelled

	default:
		return exchangeDomain.OrderStatusUnknown
	}
}