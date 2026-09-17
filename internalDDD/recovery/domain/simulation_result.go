package domain

import "github.com/shopspring/decimal"

type SimulationResult struct {
	Route        RecoveryRoute
	InputAmount  decimal.Decimal
	OutputAmount decimal.Decimal
	Steps        []RecoveryStep
}

func (s SimulationResult) IsBetterThan(
	other SimulationResult,
) bool {

	return s.OutputAmount.GreaterThan(
		other.OutputAmount,
	)
}
