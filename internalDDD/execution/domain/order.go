package domain

import (
	"time"

	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type OrderStatus uint8

const (
	OrderPending OrderStatus = iota
	OrderSubmitted
	OrderPartiallyFilled
	OrderFilled
	OrderCancelled
	OrderFailed
)

func (s OrderStatus) String() string {
	switch s {
	case OrderPending:
		return "pending"

	case OrderSubmitted:
		return "submitted"

	case OrderPartiallyFilled:
		return "partially_filled"

	case OrderFilled:
		return "filled"

	case OrderCancelled:
		return "cancelled"

	case OrderFailed:
		return "failed"

	default:
		return "unknown"
	}
}

type Order struct {
	StepIndex int

	Symbol    string
	Direction marketDomain.TradeDirection

	ClientOrderID   string
	ExchangeOrderID string

	RequestedAmount   decimal.Decimal
	ExpectedAmountOut decimal.Decimal
	ExecutedAmount    decimal.Decimal
	ReceivedAmount    decimal.Decimal

	Fee      decimal.Decimal
	FeeAsset string

	Status OrderStatus

	CreatedAt   time.Time
	SubmittedAt time.Time
	FilledAt    time.Time
}

func NewOrder(
	stepIndex int,
	result OrderExecutionResult,
) Order {
	return Order{
		StepIndex: stepIndex,

		Symbol:    result.Symbol,
		Direction: result.Direction,

		ClientOrderID:   result.ClientOrderID,
		ExchangeOrderID: result.ExchangeOrderID,

		RequestedAmount:   result.RequestedAmount,
		ExpectedAmountOut: result.ExpectedAmountOut,
		ExecutedAmount:    result.ExecutedAmount,
		ReceivedAmount:    result.ReceivedAmount,

		Fee: result.Fee,
		FeeAsset: result.FeeAsset,

		Status: result.Status,

		CreatedAt:   result.CreatedAt,
		SubmittedAt: result.SubmittedAt,
		FilledAt:    result.FilledAt,
	}
}