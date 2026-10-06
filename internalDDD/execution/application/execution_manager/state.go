package em

import (
	executionDomain "traiderBot/internalDDD/execution/domain"
	recoveryApplication "traiderBot/internalDDD/recovery/application"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func (m *ExecutionManager) buildInitialState(
	opportunity simulationDomain.Opportunity,
) recoveryDomain.State {
	state := make(recoveryDomain.State)

	startAsset := opportunity.Route.Edges[0].From

	state[startAsset] = opportunity.ExecutedStartAmount

	return state
}

func (m *ExecutionManager) applyOrderResult(
	state recoveryDomain.State,
	step executionDomain.ExecutionStep,
	result executionDomain.OrderResult,
) recoveryDomain.State {
	nextState := recoveryApplication.CloneState(state)

	nextState[step.Edge.From] = nextState[step.Edge.From].Sub(
		result.ExecutedAmount,
	)

	nextState[step.Edge.To] = nextState[step.Edge.To].Add(
		result.ReceivedAmount,
	)

	if !result.Fee.IsZero() {
		nextState[result.FeeAsset] = nextState[result.FeeAsset].Sub(
			result.Fee,
		)
	}

	return nextState
}

func (m *ExecutionManager) buildExecutionSteps(
	opportunity simulationDomain.Opportunity,
) []executionDomain.ExecutionStep {
	steps := make(
		[]executionDomain.ExecutionStep,
		0,
		len(opportunity.Steps),
	)

	for index, step := range opportunity.Steps {
		steps = append(
			steps,
			executionDomain.ExecutionStep{
				ID:   index,
				Edge: step.Edge,
				PlannedInput: executionDomain.AssetAmount{
					Asset:    step.Edge.From,
					Quantity: step.ExecutedInput,
				},
			},
		)
	}

	return steps
}
