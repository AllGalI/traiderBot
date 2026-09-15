package binance

import (
	"fmt"
	"traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
)

type createOrderRequest struct {
	Symbol           string
	Side             string
	Type             string
	Quantity         string
	QuoteOrderQty    string
	NewClientOrderID string
}

func toCreateOrderRequest(
	request domain.OrderRequest,
) (createOrderRequest, error) {
	side, err := binanceSide(request.Edge.Direction)
	if err != nil {
		return createOrderRequest{}, err
	}

	result := createOrderRequest{
		Symbol:           request.Edge.Symbol,
		Side:             side,
		Type:             "MARKET",
		NewClientOrderID: request.ClientOrderID,
	}

	switch request.Edge.Direction {
	case marketDomain.Buy:
		result.QuoteOrderQty = request.Amount.String()

	case marketDomain.Sell:
		result.Quantity = request.Amount.String()

	default:
		return createOrderRequest{}, fmt.Errorf(
			"unsupported trade direction: %v",
			request.Edge.Direction,
		)
	}

	return result, nil
}

func binanceSide(direction marketDomain.TradeDirection) (string, error) {
	switch direction {
	case marketDomain.Buy:
		return "BUY", nil
	case marketDomain.Sell:
		return "SELL", nil
	default:
		return "", fmt.Errorf("unsupported trade direction: %v", direction)
	}
}
