package paper

import (
	"context"
	"fmt"
	"sync"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

type OrderBookReader interface {
	Get(symbol string) (marketDomain.OrderBook, bool)
}

type SymbolProvider interface {
	Get(symbol string) (marketDomain.Symbol, bool)
}

type FeeProvider interface {
	Get(
		ctx context.Context,
		symbol string,
	) (simulationApplication.FeeConfig, bool)
}

type Executor struct {
	orderBooks OrderBookReader
	symbols    SymbolProvider
	fees       FeeProvider

	minLiquidityFillRatio decimal.Decimal

	mu      sync.RWMutex
	results map[string]exchangeDomain.OrderResult
}

func NewExecutor(
	orderBooks OrderBookReader,
	symbols SymbolProvider,
	fees FeeProvider,
	minLiquidityFillRatio decimal.Decimal,
) *Executor {
	return &Executor{
		orderBooks:            orderBooks,
		symbols:               symbols,
		fees:                  fees,
		minLiquidityFillRatio: minLiquidityFillRatio,

		results: make(map[string]exchangeDomain.OrderResult),
	}
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

	if request.Edge.From == "" {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"source asset must not be empty",
		)
	}

	if request.Edge.To == "" {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"destination asset must not be empty",
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

	if e.orderBooks == nil {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"order book reader is not configured",
		)
	}

	if e.symbols == nil {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"symbol provider is not configured",
		)
	}

	if e.fees == nil {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"fee provider is not configured",
		)
	}

	if e.minLiquidityFillRatio.LessThanOrEqual(decimal.Zero) ||
		e.minLiquidityFillRatio.GreaterThan(decimal.NewFromInt(1)) {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"min liquidity fill ratio must be greater than 0 and less than or equal to 1",
		)
	}

	orderBook, ok := e.orderBooks.Get(request.Edge.Symbol)
	if !ok {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"order book not found for symbol %s",
			request.Edge.Symbol,
		)
	}

	symbol, ok := e.symbols.Get(request.Edge.Symbol)
	if !ok {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"symbol not found for %s",
			request.Edge.Symbol,
		)
	}

	fee, ok := e.fees.Get(ctx, request.Edge.Symbol)
	if !ok {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"fee not found for symbol %s",
			request.Edge.Symbol,
		)
	}

	step, err := simulationDomain.SimulateStep(
		request.Edge,
		orderBook,
		symbol.Constraints,
		request.Amount,
		e.minLiquidityFillRatio,
		fee.Rate,
		request.Edge.To,
	)
	if err != nil {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"simulate paper execution for %s: %w",
			request.Edge.Symbol,
			err,
		)
	}

	result := exchangeDomain.OrderResult{
		Status:          exchangeDomain.OrderStatusFilled,
		ExchangeOrderID: request.ClientOrderID,
		ClientOrderID:   request.ClientOrderID,

		ExecutedAmount: step.AmountIn,
		ReceivedAmount: step.AmountOut,

		Fee:      step.Fee,
		FeeAsset: step.FeeAsset,
	}

	if step.AmountIn.LessThan(request.Amount) {
		result.Status = exchangeDomain.OrderStatusPartiallyFilled
	}

	e.mu.Lock()
	e.results[request.ClientOrderID] = result
	e.mu.Unlock()

	return result, nil
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

	e.mu.RLock()
	result, ok := e.results[request.ClientOrderID]
	e.mu.RUnlock()

	if !ok {
		return exchangeDomain.OrderResult{}, fmt.Errorf(
			"order %s not found",
			request.ClientOrderID,
		)
	}

	return result, nil
}