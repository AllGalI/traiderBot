package paper

import (
	"context"
	"fmt"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"

	"github.com/shopspring/decimal"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(
	ctx context.Context,
	request exchangeDomain.OrderRequest,
) (exchangeDomain.OrderResult, error) {
	if err := ctx.Err(); err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	if request.Edge.Symbol == "" {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"symbol must not be empty",
		)
	}

	if request.Amount.LessThanOrEqual(decimal.Zero) {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"amount must be greater than zero",
		)
	}

	if request.ClientOrderID == "" {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"client order ID must not be empty",
		)
	}

	return exchangeDomain.OrderResult{
		Status:          exchangeDomain.OrderStatusFilled,
		ExchangeOrderID: request.ClientOrderID,
		ClientOrderID:   request.ClientOrderID,
		ExecutedAmount:  request.Amount,
		ReceivedAmount:  request.Amount,
		Fee:             decimal.Zero,
		FeeAsset:        "",
	}, nil
}

func (e *Executor) GetOrderByClientID(
	ctx context.Context,
	request exchangeDomain.OrderRequest,
) (exchangeDomain.OrderResult, error) {
	if err := ctx.Err(); err != nil {
		return exchangeDomain.OrderResult{}, err
	}

	if request.ClientOrderID == "" {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"client order ID must not be empty",
		)
	}

	return exchangeDomain.OrderResult{
		Status:          exchangeDomain.OrderStatusFilled,
		ExchangeOrderID: request.ClientOrderID,
		ClientOrderID:   request.ClientOrderID,
		ExecutedAmount:  request.Amount,
		ReceivedAmount: request.Amount,
		Fee:             decimal.Zero,
		FeeAsset:        "",
	}, nil
}