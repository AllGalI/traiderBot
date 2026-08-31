package binance

import (
	"fmt"
	"strconv"
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

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

func millisToTime(
	value int64,
) time.Time {
	if value <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(value)
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
