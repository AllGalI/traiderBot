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

	if result.ExecutedAmount.IsNegative() ||
		result.ReceivedAmount.IsNegative() ||
		result.Fee.IsNegative() {
		return ActualStepResult{}, fmt.Errorf(
			"order amounts must not be negative",
		)
	}

	if result.Fee.GreaterThan(result.ReceivedAmount) {
		return ActualStepResult{}, fmt.Errorf(
			"fee must not be greater than received amount",
		)
	}

	feeAsset := result.FeeAsset
	if feeAsset == "" {
		return ActualStepResult{}, fmt.Errorf(
			"fee asset is required",
		)
	}

	netOutput := result.ReceivedAmount

	if feeAsset == step.Edge.To {
		netOutput = result.ReceivedAmount.Sub(result.Fee)
	}

	return ActualStepResult{
		StepID: step.ID,

		ActualInput: AssetAmount{
			Asset:    step.Edge.From,
			Quantity: result.ExecutedAmount,
		},

		GrossOutput: AssetAmount{
			Asset:    step.Edge.To,
			Quantity: result.ReceivedAmount,
		},

		Fee: AssetAmount{
			Asset:    feeAsset,
			Quantity: result.Fee,
		},

		NetOutput: AssetAmount{
			Asset:    step.Edge.To,
			Quantity: netOutput,
		},

		CompletedAt: completedAt,
	}, nil
}