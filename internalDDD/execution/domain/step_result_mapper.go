package domain

import (
	"fmt"
	"time"
)

type ActualStepResultMapper struct{}

func NewActualStepResultMapper() ActualStepResultMapper {
	return ActualStepResultMapper{}
}

func (m ActualStepResultMapper) Map(
	step ExecutionStep,
	result OrderResult,
	completedAt time.Time,
) (ActualStepResult, error) {
	if step.Edge.From == "" {
		return ActualStepResult{}, fmt.Errorf("step source asset is required")
	}

	if step.Edge.To == "" {
		return ActualStepResult{}, fmt.Errorf("step destination asset is required")
	}

	if result.ExecutedAmount.IsNegative() || result.ReceivedAmount.IsNegative() {
		return ActualStepResult{}, fmt.Errorf(
			"order amounts must not be negative",
		)
	}

	return ActualStepResult{
		StepID: step.ID,
		Input: AssetAmount{
			Asset:    step.Edge.From,
			Quantity: result.ExecutedAmount,
		},
		Output: AssetAmount{
			Asset:    step.Edge.To,
			Quantity: result.ReceivedAmount,
		},
		CompletedAt: completedAt,
	}, nil
}