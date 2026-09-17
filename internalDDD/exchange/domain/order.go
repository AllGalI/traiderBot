package domain

import (
	"fmt"

	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "Pending"
	OrderStatusPartiallyFilled OrderStatus = "PartiallyFilled"
	OrderStatusFilled          OrderStatus = "Filled"
	OrderStatusRejected        OrderStatus = "Rejected"
	OrderStatusCancelled       OrderStatus = "Cancelled"
	OrderStatusUnknown         OrderStatus = "Unknown"
)

type OrderRequest struct {
	Edge          marketDomain.GraphEdge
	Amount        decimal.Decimal
	ClientOrderID string
}

func NewOrderRequest(
	edge marketDomain.GraphEdge,
	amount decimal.Decimal,
	clientOrderID string,
) (OrderRequest, error) {
	if edge.Symbol == "" {
		return OrderRequest{}, fmt.Errorf("symbol is required")
	}

	if amount.IsZero() || amount.IsNegative() {
		return OrderRequest{}, fmt.Errorf("amount must be greater than zero")
	}

	if clientOrderID == "" {
		return OrderRequest{}, fmt.Errorf("client order ID is required")
	}

	return OrderRequest{
		Edge:          edge,
		Amount:        amount,
		ClientOrderID: clientOrderID,
	}, nil
}

type OrderResult struct {
	Status          OrderStatus
	ExchangeOrderID string
	ClientOrderID   string
	ExecutedAmount  decimal.Decimal
	ReceivedAmount  decimal.Decimal
	Fee             decimal.Decimal
	FeeAsset        string
}