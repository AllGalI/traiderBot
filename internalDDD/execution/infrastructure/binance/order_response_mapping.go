package binance

import (
	"fmt"
	"strconv"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type orderResponse struct {
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

type orderFill struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}

func toOrderResult(
	request executionDomain.OrderRequest,
	response orderResponse,
) (executionDomain.OrderResult, error) {
	executedQty, err := decimal.NewFromString(response.ExecutedQty)
	if err != nil {
		return executionDomain.OrderResult{}, fmt.Errorf(
			"parse executed qty: %w",
			err,
		)
	}

	quoteQty, err := decimal.NewFromString(response.CumulativeQuoteQty)
	if err != nil {
		return executionDomain.OrderResult{}, fmt.Errorf(
			"parse cumulative quote qty: %w",
			err,
		)
	}

	status, err := mapOrderStatus(response.Status)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	fee, feeAsset, err := parseOrderFee(response.Fills)
	if err != nil {
		return executionDomain.OrderResult{}, err
	}

	var (
		executedAmount decimal.Decimal
		receivedAmount decimal.Decimal
	)

	switch request.Edge.Direction {
	case marketDomain.Buy:
		executedAmount = quoteQty
		receivedAmount = executedQty

	case marketDomain.Sell:
		executedAmount = executedQty
		receivedAmount = quoteQty

	default:
		return executionDomain.OrderResult{}, fmt.Errorf(
			"unsupported trade direction: %v",
			request.Edge.Direction,
		)
	}

	fmt.Printf(
		"[BINANCE ORDER] symbol=%s side=%s status=%s executedQty=%s cumulativeQuoteQty=%s\n",
		response.Symbol,
		response.Side,
		response.Status,
		response.ExecutedQty,
		response.CumulativeQuoteQty,
	)

	return executionDomain.OrderResult{
		Status:          status,
		ExchangeOrderID: strconv.FormatInt(response.OrderID, 10),
		ClientOrderID:   response.ClientOrderID,
		ExecutedAmount:  executedAmount,
		ReceivedAmount:  receivedAmount,
		Fee:             fee,
		FeeAsset:        feeAsset,
	}, nil
}

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

func mapOrderStatus(
	status string,
) (executionDomain.OrderStatus, error) {
	switch status {
	case "NEW":
		return executionDomain.OrderStatusPending, nil

	case "PARTIALLY_FILLED":
		return executionDomain.OrderStatusPartiallyFilled, nil

	case "FILLED":
		return executionDomain.OrderStatusFilled, nil

	case "CANCELED", "EXPIRED", "PENDING_CANCEL":
		return executionDomain.OrderStatusCancelled, nil

	case "REJECTED":
		return executionDomain.OrderStatusRejected, nil

	default:
		return executionDomain.OrderStatusUnknown, fmt.Errorf(
			"unsupported binance order status: %s",
			status,
		)
	}
}
