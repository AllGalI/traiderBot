package em

import (
	"fmt"
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
)

func (m *ExecutionManager) createExecution(
	execution *executionDomain.Execution,
	currentStep int,
	totalSteps int,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	err error,
) {
	if m.executionStore == nil {
		return
	}

	snapshot := m.snapshot(
		execution,
		currentStep,
		totalSteps,
		initialState,
		actualState,
		err,
	)

	snapshot.StartedAt = time.Now()

	_ = m.executionStore.Create(&snapshot)
}

func (m *ExecutionManager) persistExecution(
	execution *executionDomain.Execution,
	currentStep int,
	totalSteps int,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	err error,
) {
	if m.executionStore == nil {
		return
	}

	snapshot := m.snapshot(
		execution,
		currentStep,
		totalSteps,
		initialState,
		actualState,
		err,
	)

	_ = m.executionStore.Update(&snapshot)
}

func (m *ExecutionManager) completeExecution(
	execution *executionDomain.Execution,
	totalSteps int,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	err error,
) error {
	if err != nil {
		return m.failExecution(
			execution,
			totalSteps,
			initialState,
			actualState,
			err,
		)
	}

	if m.executionStore == nil {
		return nil
	}

	snapshot := m.snapshot(
		execution,
		totalSteps,
		totalSteps,
		initialState,
		actualState,
		nil,
	)

	now := time.Now()
	snapshot.CompletedAt = &now

	_ = m.executionStore.Complete(&snapshot)

	return nil
}

func (m *ExecutionManager) failExecution(
	execution *executionDomain.Execution,
	currentStep int,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	err error,
) error {
	if execution == nil {
		return nil
	}

	if execution.IsFinished() {
		return nil
	}

	if failErr := execution.Fail(); failErr != nil {
		return failErr
	}

	fmt.Printf(
		"[EXECUTION-FAILED] execution=%s opportunity=%s step=%d error=%v\n",
		execution.ID(),
		execution.Opportunity().ID,
		currentStep,
		err,
	)

	if m.executionStore == nil {
		return nil
	}

	snapshot := m.snapshot(
		execution,
		currentStep,
		len(execution.Opportunity().Route.Edges),
		initialState,
		actualState,
		err,
	)

	now := time.Now()
	snapshot.CompletedAt = &now

	_ = m.executionStore.Complete(&snapshot)

	return nil
}

func (m *ExecutionManager) snapshot(
	execution *executionDomain.Execution,
	currentStep int,
	totalSteps int,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	err error,
) executionDomain.ExecutionSnapshot {
	now := time.Now()

	result := executionDomain.ExecutionSnapshot{
		ID: execution.ID(),

		RecoveryID: execution.RecoveryID(),

		OpportunityID: execution.Opportunity().ID,

		State: execution.State(),

		UpdatedAt: now,

		CurrentStep: currentStep,

		TotalSteps: totalSteps,

		ActualSteps: len(
			execution.ActualStepResults(),
		),

		InitialState: m.buildBalanceSnapshots(
			initialState,
		),

		ActualState: m.buildBalanceSnapshots(
			actualState,
		),

		Steps: m.buildStepSnapshots(
			execution.ActualStepResults(),
		),
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result
}

func (m *ExecutionManager) buildStepSnapshots(
	results []executionDomain.ActualStepResult,
) []executionDomain.ExecutionStepSnapshot {
	snapshots := make(
		[]executionDomain.ExecutionStepSnapshot,
		0,
		len(results),
	)

	for _, result := range results {
		snapshots = append(
			snapshots,
			executionDomain.ExecutionStepSnapshot{
				StepID: result.StepID,

				PlannedInput:   result.PlannedInput,
				RequestedInput: result.RequestedInput,
				ActualInput:    result.ActualInput,

				GrossOutput: result.GrossOutput,
				Fee:         result.Fee,
				NetOutput:   result.NetOutput,

				FillRatio: result.FillRatio,

				Result: result.Result,
				Error:  result.Error,

				StartedAt:   result.StartedAt,
				CompletedAt: result.CompletedAt,
			},
		)
	}

	return snapshots
}

func (m *ExecutionManager) buildBalanceSnapshots(
	state recoveryDomain.State,
) []executionDomain.BalanceSnapshot {
	snapshots := make(
		[]executionDomain.BalanceSnapshot,
		0,
		len(state),
	)

	for asset, quantity := range state {
		snapshots = append(
			snapshots,
			executionDomain.BalanceSnapshot{
				Asset:    asset,
				Quantity: quantity,
			},
		)
	}

	return snapshots
}