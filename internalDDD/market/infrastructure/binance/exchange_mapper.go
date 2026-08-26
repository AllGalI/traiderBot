package binance

import (
	"fmt"

	"github.com/shopspring/decimal"

	"traiderBot/internalDDD/market/domain"
)

func mapSymbol(dto symbolDTO) (domain.Symbol, error) {
	if dto.Symbol == "" {
		return domain.Symbol{}, fmt.Errorf("symbol is empty")
	}

	if dto.BaseAsset == "" {
		return domain.Symbol{}, fmt.Errorf(
			"symbol %q has empty base asset",
			dto.Symbol,
		)
	}

	if dto.QuoteAsset == "" {
		return domain.Symbol{}, fmt.Errorf(
			"symbol %q has empty quote asset",
			dto.Symbol,
		)
	}

	constraints, err := mapSymbolConstraints(dto.Filters)
	if err != nil {
		return domain.Symbol{}, fmt.Errorf(
			"map symbol %q constraints: %w",
			dto.Symbol,
			err,
		)
	}

	return domain.Symbol{
		Name:        dto.Symbol,
		BaseAsset:   dto.BaseAsset,
		QuoteAsset:  dto.QuoteAsset,
		Constraints: constraints,
	}, nil
}

func mapSymbolConstraints(
	filters []filterDTO,
) (domain.SymbolConstraints, error) {
	var constraints domain.SymbolConstraints

	for _, filter := range filters {
		switch filter.FilterType {
		case "LOT_SIZE":
			minQty, err := parseDecimal(filter.MinQty)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse minQty: %w",
					err,
				)
			}

			maxQty, err := parseDecimal(filter.MaxQty)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse maxQty: %w",
					err,
				)
			}

			stepSize, err := parseDecimal(filter.StepSize)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse stepSize: %w",
					err,
				)
			}

			constraints.MinQty = minQty
			constraints.MaxQty = maxQty
			constraints.StepSize = stepSize

		case "MIN_NOTIONAL":
			minNotional, err := parseDecimal(filter.MinNotional)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse minNotional: %w",
					err,
				)
			}

			constraints.MinNotional = minNotional

		case "PRICE_FILTER":
			tickSize, err := parseDecimal(filter.TickSize)
			if err != nil {
				return domain.SymbolConstraints{}, fmt.Errorf(
					"parse tickSize: %w",
					err,
				)
			}

			constraints.PriceStep = tickSize
		}
	}

	return constraints, nil
}

func parseDecimal(value string) (decimal.Decimal, error) {
	if value == "" {
		return decimal.Zero, nil
	}

	result, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf(
			"parse %q as decimal: %w",
			value,
			err,
		)
	}

	return result, nil
}