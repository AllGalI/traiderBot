package domain

import (
	"fmt"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
)

type StepResult struct {
	Edge marketDomain.GraphEdge

	AvgPrice decimal.Decimal
	Quantity decimal.Decimal
	QuoteQty decimal.Decimal

	AmountIn  decimal.Decimal
	AmountOut decimal.Decimal

	// Доля реально исполненного входного объёма.
	//
	// 1.0 = полное заполнение.
	// 0.75 = исполнено 75%.
	FillRatio decimal.Decimal

	// Комиссия.
	Fee      decimal.Decimal
	FeeAsset string
}

func SimulateStep(
	edge marketDomain.GraphEdge,
	orderBook marketDomain.OrderBook,
	constraints marketDomain.SymbolConstraints,
	amountIn decimal.Decimal,
	minFillRatio decimal.Decimal,
	feeRate decimal.Decimal,
	feeAsset string,
) (StepResult, error) {
	if amountIn.LessThanOrEqual(decimal.Zero) {
		return StepResult{}, fmt.Errorf(
			"amount in must be greater than zero",
		)
	}

	if minFillRatio.LessThanOrEqual(decimal.Zero) ||
		minFillRatio.GreaterThan(decimal.NewFromInt(1)) {
		return StepResult{}, fmt.Errorf(
			"min fill ratio must be greater than 0 and less than or equal to 1",
		)
	}

	if feeRate.LessThan(decimal.Zero) ||
		feeRate.GreaterThan(decimal.NewFromInt(1)) {
		return StepResult{}, fmt.Errorf(
			"fee rate must be between 0 and 1",
		)
	}

	if feeAsset == "" {
		return StepResult{}, fmt.Errorf(
			"fee asset must not be empty",
		)
	}

	var levels []marketDomain.PriceLevel

	if edge.Direction == marketDomain.Buy {
		levels = orderBook.Asks
	} else {
		levels = orderBook.Bids
	}

	if len(levels) == 0 {
		return StepResult{}, fmt.Errorf(
			"order book %s has no levels",
			orderBook.Symbol,
		)
	}

	avgPrice, amountOut, quantity, filledInput, err := walkBook(
		edge,
		levels,
		amountIn,
		constraints,
	)

	if err != nil {
		return StepResult{}, err
	}

	// Количество base asset должно соответствовать
	// ограничениям символа.
	if quantity.LessThan(constraints.MinQty) {
		return StepResult{}, fmt.Errorf(
			"quantity %s is below minimum quantity %s",
			quantity,
			constraints.MinQty,
		)
	}

	if constraints.MaxQty.GreaterThan(decimal.Zero) &&
		quantity.GreaterThan(constraints.MaxQty) {
		return StepResult{}, fmt.Errorf(
			"quantity %s exceeds maximum quantity %s",
			quantity,
			constraints.MaxQty,
		)
	}

	// Стоимость сделки.
	quoteQty := quantity.Mul(avgPrice)

	if constraints.MinNotional.GreaterThan(decimal.Zero) &&
		quoteQty.LessThan(constraints.MinNotional) {
		return StepResult{}, fmt.Errorf(
			"notional %s is below minimum notional %s",
			quoteQty,
			constraints.MinNotional,
		)
	}

	fillRatio := filledInput.Div(amountIn)
	if fillRatio.GreaterThan(decimal.NewFromInt(1)) {
		fillRatio = decimal.NewFromInt(1)
	}

	if fillRatio.LessThan(minFillRatio) {
		return StepResult{}, fmt.Errorf(
			"insufficient liquidity: fill ratio %s is below minimum %s",
			fillRatio,
			minFillRatio,
		)
	}

	fee, netAmountOut, err := calculateFee(
		edge,
		amountOut,
		feeRate,
		feeAsset,
	)

	if err != nil {
		return StepResult{}, err
	}

	return StepResult{
		Edge:      edge,
		AvgPrice:  avgPrice,
		Quantity:  quantity,
		QuoteQty:  quoteQty,
		AmountIn:  filledInput,
		AmountOut: netAmountOut,
		FillRatio: fillRatio,
		Fee:       fee,
		FeeAsset:  feeAsset,
	}, nil
}

func calculateFee(
	edge marketDomain.GraphEdge,
	amountOut decimal.Decimal,
	feeRate decimal.Decimal,
	feeAsset string,
) (
	fee decimal.Decimal,
	netAmountOut decimal.Decimal,
	err error,
) {
	if amountOut.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero,
			decimal.Zero,
			fmt.Errorf("amount out must be greater than zero")
	}

	if feeRate.IsZero() {
		return decimal.Zero, amountOut, nil
	}

	// На текущем этапе поддерживаем комиссию
	// только в валюте результата сделки.
	expectedFeeAsset := edge.To

	if feeAsset != expectedFeeAsset {
		return decimal.Zero,
			decimal.Zero,
			fmt.Errorf(
				"unsupported fee asset %s for step %s -> %s: expected %s",
				feeAsset,
				edge.From,
				edge.To,
				expectedFeeAsset,
			)
	}

	fee = amountOut.Mul(feeRate)
	netAmountOut = amountOut.Sub(fee)

	if netAmountOut.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero,
			decimal.Zero,
			fmt.Errorf("fee consumes entire amount out")
	}

	return fee, netAmountOut, nil
}

func walkBook(
	edge marketDomain.GraphEdge,
	levels []marketDomain.PriceLevel,
	amountIn decimal.Decimal,
	constraints marketDomain.SymbolConstraints,
) (
	avgPrice decimal.Decimal,
	amountOut decimal.Decimal,
	quantity decimal.Decimal,
	filledInput decimal.Decimal,
	err error,
) {
	remaining := amountIn

	var totalQuote decimal.Decimal
	var totalBase decimal.Decimal

	for _, level := range levels {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		if level.Price.LessThanOrEqual(decimal.Zero) ||
			level.Qty.LessThanOrEqual(decimal.Zero) {
			continue
		}

		if edge.Direction == marketDomain.Buy {
			// BUY:
			//
			// amountIn  = quote
			// amountOut = base

			maxQuote := level.Qty.Mul(level.Price)

			usedQuote := decimal.Min(
				remaining,
				maxQuote,
			)

			gotBase := usedQuote.DivRound(level.Price, 16)

			// StepSize применяется только если он задан.
			if constraints.StepSize.GreaterThan(decimal.Zero) {
				gotBase = floorToStep(
					gotBase,
					constraints.StepSize,
				)

				if gotBase.LessThanOrEqual(decimal.Zero) {
					continue
				}

				// После округления количества пересчитываем
				// фактически потраченный quote.
				usedQuote = gotBase.Mul(level.Price)
			}

			// Защита от переполнения из-за округления.
			if usedQuote.GreaterThan(remaining) {
				usedQuote = remaining
			}

			if usedQuote.LessThanOrEqual(decimal.Zero) {
				continue
			}

			totalQuote = totalQuote.Add(usedQuote)
			totalBase = totalBase.Add(gotBase)

			remaining = remaining.Sub(usedQuote)

			continue
		}

		// SELL:
		//
		// amountIn  = base
		// amountOut = quote

		usedBase := decimal.Min(
			remaining,
			level.Qty,
		)

		// StepSize применяется к продаваемому
		// количеству base asset.
		usedBase = floorToStep(
			usedBase,
			constraints.StepSize,
		)

		if usedBase.LessThanOrEqual(decimal.Zero) {
			continue
		}

		gotQuote := usedBase.Mul(level.Price)

		totalBase = totalBase.Add(usedBase)
		totalQuote = totalQuote.Add(gotQuote)

		remaining = remaining.Sub(usedBase)
	}

	if totalBase.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero,
			decimal.Zero,
			decimal.Zero,
			decimal.Zero,
			fmt.Errorf("order book has insufficient liquidity")
	}

	avgPrice = totalQuote.DivRound(totalBase, 16)
	quantity = totalBase

	if edge.Direction == marketDomain.Buy {
		amountOut = totalBase
		filledInput = totalQuote
	} else {
		amountOut = totalQuote
		filledInput = totalBase
	}

	// Из-за округления decimal значение фактически
	// исполненного объёма не должно превышать запрошенный.
	if filledInput.GreaterThan(amountIn) {
		filledInput = amountIn
	}

	return avgPrice,
		amountOut,
		quantity,
		filledInput,
		nil
}
func floorToStep(
	value decimal.Decimal,
	step decimal.Decimal,
) decimal.Decimal {
	if step.LessThanOrEqual(decimal.Zero) {
		return value
	}

	return value.Div(step).Floor().Mul(step)
}
