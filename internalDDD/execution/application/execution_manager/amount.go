package em

import (
	"fmt"

	marketDomain "traiderBot/internalDDD/market/domain"
	recoveryApplication "traiderBot/internalDDD/recovery/application"

	"github.com/shopspring/decimal"
)

func (m *ExecutionManager) normalizeOrderAmount(
	amount decimal.Decimal,
	edge marketDomain.GraphEdge,
	marketContext recoveryApplication.MarketContext,
) (decimal.Decimal, error) {
	if amount.IsZero() || amount.IsNegative() {
		return decimal.Zero, fmt.Errorf(
			"order amount must be greater than zero: %s",
			amount,
		)
	}

	symbol, ok := marketContext.Symbols[edge.Symbol]
	if !ok {
		return decimal.Zero, fmt.Errorf(
			"symbol %s not found in market context",
			edge.Symbol,
		)
	}

	// SELL sends base asset quantity.
	// Example: ETHUSDT -> quantity=0.0364 ETH.
	if edge.Direction == marketDomain.Sell {
		return normalizeSellAmount(
			amount,
			symbol,
		)
	}

	// BUY MARKET sends quote asset amount.
	// Example: ETHUSDT -> quoteOrderQty=99.98 USDT.
	return normalizeBuyAmount(
		amount,
		symbol,
	)
}

func normalizeBuyAmount(
	amount decimal.Decimal,
	symbol marketDomain.Symbol,
) (decimal.Decimal, error) {
	// For BUY quoteOrderQty we currently don't have a dedicated
	// quote-order step in SymbolConstraints.
	//
	// Therefore do not use StepSize here:
	// StepSize belongs to the base asset quantity.
	//
	// We only validate that the amount itself is positive.
	if amount.IsZero() || amount.IsNegative() {
		return decimal.Zero, fmt.Errorf(
			"BUY quote order amount must be greater than zero: %s",
			amount,
		)
	}

	// MinNotional is a lower bound for the order value.
	if !symbol.Constraints.MinNotional.IsZero() &&
		amount.LessThan(symbol.Constraints.MinNotional) {
		return decimal.Zero, fmt.Errorf(
			"BUY quote order amount %s is below minNotional %s for %s",
			amount.String(),
			symbol.Constraints.MinNotional.String(),
			symbol.Name,
		)
	}

	return amount, nil
}

func normalizeSellAmount(
	amount decimal.Decimal,
	symbol marketDomain.Symbol,
) (decimal.Decimal, error) {
	constraints := symbol.Constraints

	// SELL quantity is base asset quantity.
	// Binance LOT_SIZE/MARKET_LOT_SIZE step is applied here.
	normalized := roundDownToStep(
		amount,
		constraints.StepSize,
	)

	if normalized.IsZero() || normalized.IsNegative() {
		return decimal.Zero, fmt.Errorf(
			"normalized SELL quantity is not positive: original=%s stepSize=%s",
			amount.String(),
			constraints.StepSize.String(),
		)
	}

	if !constraints.MinQty.IsZero() &&
		normalized.LessThan(constraints.MinQty) {
		return decimal.Zero, fmt.Errorf(
			"SELL quantity %s is below minQty %s for %s",
			normalized.String(),
			constraints.MinQty.String(),
			symbol.Name,
		)
	}

	if !constraints.MaxQty.IsZero() &&
		normalized.GreaterThan(constraints.MaxQty) {
		return decimal.Zero, fmt.Errorf(
			"SELL quantity %s exceeds maxQty %s for %s",
			normalized.String(),
			constraints.MaxQty.String(),
			symbol.Name,
		)
	}

	return normalized, nil
}

func roundDownToStep(
	amount decimal.Decimal,
	step decimal.Decimal,
) decimal.Decimal {
	if step.IsZero() || step.IsNegative() {
		return amount
	}

	return amount.
		Div(step).
		Floor().
		Mul(step)
}