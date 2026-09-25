package application

import (
	"context"
	"fmt"
	"time"

	recoveryDomain "traiderBot/internalDDD/recovery/domain"
)

const maxRecoveryAttempts = 3

type RecoveryService struct {
	planner  Planner
	executor RecoveryExecutor
	store    RecoveryStore
}

func NewRecoveryService(
	planner Planner,
	executor RecoveryExecutor,
	store RecoveryStore,
) RecoveryService {
	return RecoveryService{
		planner:  planner,
		executor: executor,
		store:    store,
	}
}

func (s RecoveryService) Recover(
	ctx context.Context,
	input recoveryDomain.RecoveryInput,
	marketContext MarketContext,
) (recoveryDomain.State, error) {

	actualState := CloneState(input.ActualState)

	snapshot := recoveryDomain.RecoverySnapshot{
		ID:          recoveryDomain.NewRecoveryID(),
		ExecutionID: input.ExecutionID,

		ProcessState: recoveryDomain.RecoveryStarted,

		InitialState: actualState,
		ActualState:  actualState,

		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if s.store != nil {
		_ = s.store.Create(&snapshot)
	}

	for attempt := 0; attempt < maxRecoveryAttempts; attempt++ {

		snapshot.ProcessState = recoveryDomain.RecoveryPlanning
		snapshot.Attempts = attempt + 1
		snapshot.UpdatedAt = time.Now()

		if s.store != nil {
			_ = s.store.Update(&snapshot)
		}

		if err := ctx.Err(); err != nil {
			return actualState, err
		}

		currentInput := recoveryDomain.RecoveryInput{
			ExecutionID: input.ExecutionID,
			InitialState: input.InitialState,
			ActualState:  actualState,
		}

		plans := s.planner.CreatePlan(
			currentInput,
			marketContext,
		)

		snapshot.PlansCreated += len(plans)

		if s.store != nil {
			_ = s.store.Update(&snapshot)
		}

		if len(plans) == 0 {
			now := time.Now()

			snapshot.ProcessState = recoveryDomain.RecoveryCompleted
			snapshot.CompletedAt = &now
			snapshot.UpdatedAt = now
			snapshot.ActualState = CloneState(actualState)

			if s.store != nil {
				_ = s.store.Complete(&snapshot)
			}

			return actualState, nil
		}

		changed := false

		for _, plan := range plans {
			previousState := CloneState(actualState)

			snapshot.ProcessState = recoveryDomain.RecoveryExecuting
			snapshot.PlansExecuted++

			snapshot.UpdatedAt = time.Now()

			if s.store != nil {
				_ = s.store.Update(&snapshot)
			}

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

			err := fmt.Errorf(
				"recovery made no progress on attempt %d",
				attempt+1,
			)

			now := time.Now()

			snapshot.ProcessState =
				recoveryDomain.RecoveryFailed

			snapshot.CompletedAt = &now
			snapshot.Error = err.Error()
			snapshot.UpdatedAt = now

			if s.store != nil {
				_ = s.store.Complete(&snapshot)
			}
			return actualState, err
		}
	}

	err := fmt.Errorf(
		"recovery failed after %d attempts",
		maxRecoveryAttempts,
	)

	now := time.Now()

	snapshot.ProcessState =
		recoveryDomain.RecoveryFailed

	snapshot.CompletedAt = &now
	snapshot.Error = err.Error()
	snapshot.UpdatedAt = now

	if s.store != nil {
		_ = s.store.Complete(&snapshot)
	}
	return actualState, err
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
