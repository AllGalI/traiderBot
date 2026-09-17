package application

import (
	"context"
	"fmt"

	recoveryDomain "traiderBot/internalDDD/recovery/domain"
)

const maxRecoveryAttempts = 3

type RecoveryService struct {
	planner  Planner
	executor RecoveryExecutor
}

func NewRecoveryService(
	planner Planner,
	executor RecoveryExecutor,
) RecoveryService {
	return RecoveryService{
		planner:  planner,
		executor: executor,
	}
}

func (s RecoveryService) Recover(
	ctx context.Context,
	input recoveryDomain.RecoveryInput,
	marketContext MarketContext,
) (recoveryDomain.State, error) {
	actualState := CloneState(input.ActualState)

	for attempt := 0; attempt < maxRecoveryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return actualState, err
		}

		currentInput := recoveryDomain.RecoveryInput{
			InitialState: input.InitialState,
			ActualState:  actualState,
		}

		plans := s.planner.CreatePlan(
			currentInput,
			marketContext,
		)

		if len(plans) == 0 {
			return actualState, nil
		}

		changed := false

		for _, plan := range plans {
			previousState := CloneState(actualState)

			nextState, err := s.executor.Execute(
				ctx,
				plan,
				actualState,
			)

			actualState = nextState

			if stateChanged(previousState, actualState) {
				changed = true
			}

			if err != nil {
				break
			}
		}

		if !changed {
			return actualState, fmt.Errorf(
				"recovery made no progress on attempt %d",
				attempt+1,
			)
		}
	}

	return actualState, fmt.Errorf(
		"recovery failed after %d attempts",
		maxRecoveryAttempts,
	)
}

func stateChanged(
	before recoveryDomain.State,
	after recoveryDomain.State,
) bool {
	if len(before) != len(after) {
		return true
	}

	for asset, beforeAmount := range before {
		afterAmount := after[asset]

		if !beforeAmount.Equal(afterAmount) {
			return true
		}
	}

	return false
}