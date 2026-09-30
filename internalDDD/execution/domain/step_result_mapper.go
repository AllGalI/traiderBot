package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type ActualStepResultMapper struct{}

func NewActualStepResultMapper() ActualStepResultMapper {
	return ActualStepResultMapper{}
}

func (m ActualStepResultMapper) Map(
	step ExecutionStep,
	request OrderRequest,
	result OrderResult,
	startedAt time.Time,
	completedAt time.Time,
) (ActualStepResult, error) {
	if step.Edge.From == "" {
		return ActualStepResult{}, fmt.Errorf("step source asset is required")
	}

	if step.Edge.To == "" {
		return ActualStepResult{}, fmt.Errorf("step destination asset is required")
	}

	if request.Amount.IsNegative() ||
		result.ExecutedAmount.IsNegative() ||
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

	if result.FeeAsset == "" {
		return ActualStepResult{}, fmt.Errorf(
			"fee asset is required",
		)
	}

	netOutput := result.ReceivedAmount

	if result.FeeAsset == step.Edge.To {
		netOutput = result.ReceivedAmount.Sub(result.Fee)
	}

	fillRatio := decimal.Zero

	if request.Amount.GreaterThan(decimal.Zero) {
		fillRatio = result.ExecutedAmount.Div(request.Amount)
	}

	if fillRatio.GreaterThan(decimal.NewFromInt(1)) {
		fillRatio = decimal.NewFromInt(1)
	}

	return ActualStepResult{
		StepID: step.ID,

		PlannedInput: step.PlannedInput,

		RequestedInput: AssetAmount{
			Asset:    step.Edge.From,
			Quantity: request.Amount,
		},

		ActualInput: AssetAmount{
			Asset:    step.Edge.From,
			Quantity: result.ExecutedAmount,
		},

		GrossOutput: AssetAmount{
			Asset:    step.Edge.To,
			Quantity: result.ReceivedAmount,
		},

		Fee: AssetAmount{
			Asset:    result.FeeAsset,
			Quantity: result.Fee,
		},

		NetOutput: AssetAmount{
			Asset:    step.Edge.To,
			Quantity: netOutput,
		},

		FillRatio: fillRatio,

		Result: result.Status,

		StartedAt:   startedAt,
		CompletedAt: completedAt,
	}, nil
}