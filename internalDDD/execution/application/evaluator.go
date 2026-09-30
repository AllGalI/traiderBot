package application

import (
	"github.com/shopspring/decimal"

	executionDomain "traiderBot/internalDDD/execution/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type Evaluator struct {
	maxDeviation decimal.Decimal
}

func NewEvaluator(
	maxDeviation decimal.Decimal,
) Evaluator {
	return Evaluator{
		maxDeviation: maxDeviation,
	}
}

func (e Evaluator) Evaluate(
	expected simulationDomain.StepResult,
	actual executionDomain.ActualStepResult,
) executionDomain.ExecutionDecision {

	// expectedAmount := expected.GrossOutput
	// actualAmount := actual.GrossOutput.Quantity

	// if expectedAmount.LessThanOrEqual(decimal.Zero) {
	// 	return executionDomain.DecisionRecoveryRequired
	// }

	// deviation := expectedAmount.
	// 	Sub(actualAmount).
	// 	Abs().
	// 	Div(expectedAmount)

	// if deviation.GreaterThan(e.maxDeviation) {
	// 	return executionDomain.DecisionRecoveryRequired
	// }

	return executionDomain.DecisionContinue
}