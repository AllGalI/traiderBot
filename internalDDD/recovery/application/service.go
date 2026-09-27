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

	now := time.Now()

	snapshot := recoveryDomain.RecoverySnapshot{
		ID:          recoveryDomain.NewRecoveryID(),
		ExecutionID: input.ExecutionID,
		Reason:      input.Reason,

		ProcessState: recoveryDomain.RecoveryStarted,

		InitialState: CloneState(actualState),
		ActualState:  CloneState(actualState),

		StartedAt: now,
		UpdatedAt: now,
	}

	if s.store != nil {
		_ = s.store.Create(&snapshot)
	}

	var lastErr error

	for attempt := 0; attempt < maxRecoveryAttempts; attempt++ {
		attemptSnapshot := recoveryDomain.RecoveryAttemptSnapshot{
			Index:       attempt + 1,
			StateBefore: CloneState(actualState),
			StartedAt:   time.Now(),
		}

		snapshot.ProcessState = recoveryDomain.RecoveryPlanning
		snapshot.Attempts = append(
			snapshot.Attempts,
			attemptSnapshot,
		)
		snapshot.UpdatedAt = time.Now()

		if s.store != nil {
			_ = s.store.Update(&snapshot)
		}

		if err := ctx.Err(); err != nil {
			lastErr = err

			attemptSnapshot.Error = err.Error()
			attemptSnapshot.StateAfter = CloneState(actualState)

			completedAt := time.Now()
			attemptSnapshot.CompletedAt = &completedAt

			snapshot.Attempts[len(snapshot.Attempts)-1] = attemptSnapshot
			snapshot.ActualState = CloneState(actualState)
			snapshot.ProcessState = recoveryDomain.RecoveryFailed
			snapshot.CompletedAt = &completedAt
			snapshot.Error = err.Error()
			snapshot.UpdatedAt = completedAt

			if s.store != nil {
				_ = s.store.Complete(&snapshot)
			}

			return actualState, err
		}

		currentInput := recoveryDomain.RecoveryInput{
			ExecutionID:  input.ExecutionID,
			InitialState: input.InitialState,
			Reason:       input.Reason,
			ActualState:  actualState,
		}

		plans := s.planner.CreatePlan(
			currentInput,
			marketContext,
		)

		attemptSnapshot.PlansCreated = len(plans)
		snapshot.PlansCreated += len(plans)

		snapshot.Attempts[len(snapshot.Attempts)-1] = attemptSnapshot
		snapshot.UpdatedAt = time.Now()

		if s.store != nil {
			_ = s.store.Update(&snapshot)
		}

		if len(plans) == 0 {
			completedAt := time.Now()

			attemptSnapshot.StateAfter = CloneState(actualState)
			attemptSnapshot.Changed = false
			attemptSnapshot.CompletedAt = &completedAt

			snapshot.Attempts[len(snapshot.Attempts)-1] = attemptSnapshot

			snapshot.ProcessState = recoveryDomain.RecoveryCompleted
			snapshot.CompletedAt = &completedAt
			snapshot.ActualState = CloneState(actualState)
			snapshot.UpdatedAt = completedAt

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

			attemptSnapshot.PlansAttempted++

			snapshot.Attempts[len(snapshot.Attempts)-1] = attemptSnapshot
			snapshot.UpdatedAt = time.Now()

			if s.store != nil {
				_ = s.store.Update(&snapshot)
			}

			nextState, err := s.executor.Execute(
				ctx,
				plan,
				actualState,
			)

			if nextState != nil {
				actualState = nextState
			}

			if stateChanged(previousState, actualState) {
				changed = true
			}

			if err != nil {
				attemptSnapshot.PlansFailed++
				attemptSnapshot.Error = err.Error()
				lastErr = err

				break
			}

			attemptSnapshot.PlansSucceeded++
		}

		attemptSnapshot.Changed = changed
		attemptSnapshot.StateAfter = CloneState(actualState)

		completedAt := time.Now()
		attemptSnapshot.CompletedAt = &completedAt

		snapshot.Attempts[len(snapshot.Attempts)-1] = attemptSnapshot

		snapshot.ActualState = CloneState(actualState)
		snapshot.UpdatedAt = completedAt

		if s.store != nil {
			_ = s.store.Update(&snapshot)
		}

		if !changed {
			var err error

			if lastErr != nil {
				err = fmt.Errorf(
					"recovery made no progress on attempt %d: %w",
					attempt+1,
					lastErr,
				)
			} else {
				err = fmt.Errorf(
					"recovery made no progress on attempt %d",
					attempt+1,
				)
			}

			snapshot.ProcessState = recoveryDomain.RecoveryFailed
			snapshot.CompletedAt = &completedAt
			snapshot.Error = err.Error()
			snapshot.UpdatedAt = completedAt
			snapshot.ActualState = CloneState(actualState)

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

	if lastErr != nil {
		err = fmt.Errorf(
			"%w: last error: %v",
			err,
			lastErr,
		)
	}

	now = time.Now()

	snapshot.ProcessState = recoveryDomain.RecoveryFailed
	snapshot.CompletedAt = &now
	snapshot.Error = err.Error()
	snapshot.ActualState = CloneState(actualState)
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