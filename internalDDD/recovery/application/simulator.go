package application

import (
	"github.com/shopspring/decimal"

	recoveryDomain "traiderBot/internalDDD/recovery/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type Simulator struct {
	minFillRatio decimal.Decimal
}

func NewSimulator(
	minFillRatio decimal.Decimal,
) Simulator {
	return Simulator{
		minFillRatio: minFillRatio,
	}
}

func (s Simulator) Simulate(
	route recoveryDomain.RecoveryRoute,
	amount decimal.Decimal,
	ctx MarketContext,
) recoveryDomain.SimulationResult {

	currentAmount := amount

	steps := make(
		[]recoveryDomain.RecoveryStep,
		0,
		len(route.Edges),
	)

	for _, edge := range route.Edges {
		symbol, ok := ctx.Symbols[edge.Symbol]
		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		book, ok := ctx.OrderBooks[edge.Symbol]
		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		fee, ok := ctx.Fees[edge.Symbol]
		if !ok {
			return recoveryDomain.SimulationResult{}
		}

		step, err := simulationDomain.SimulateStep(
			edge,
			book,
			symbol.Constraints,
			currentAmount,
			s.minFillRatio,
			fee.Taker,
			edge.To,
		)
		if err != nil {
			return recoveryDomain.SimulationResult{}
		}

		recoveryStep := recoveryDomain.RecoveryStep{
			FromAsset:      edge.From,
			ToAsset:        edge.To,
			Symbol:         edge.Symbol,
			Direction:      edge.Direction,
			RequestedInput: step.RequestedInput,
			ExecutedInput:  step.ExecutedInput,
			GrossOutput:    step.GrossOutput,
			NetOutput:      step.NetOutput,
			SimulatedPrice: step.AvgPrice,
			SimulatedFee:   step.Fee,
			FillRatio:      step.FillRatio,
		}

		steps = append(
			steps,
			recoveryStep,
		)

		currentAmount = step.NetOutput
	}

	if len(steps) == 0 {
		return recoveryDomain.SimulationResult{}
	}

	return recoveryDomain.SimulationResult{
		Route:        route,
		InputAmount:  amount,
		OutputAmount: currentAmount,
		Steps:        steps,
	}
}
