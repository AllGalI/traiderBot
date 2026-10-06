package domain

import (
	"time"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
)

type ExecutionProvider struct {
	store executionApplication.ExecutionStore
}

func NewExecutionProvider(
	store executionApplication.ExecutionStore,
) *ExecutionProvider {
	return &ExecutionProvider{
		store: store,
	}
}

func (p *ExecutionProvider) Name() string {
	return "ExecutionProvider"
}

func (p *ExecutionProvider) Snapshot() (any, error) {
	latest, ok := p.store.Latest()

	if !ok {
		return map[string]any{
			"status": "empty",
		}, nil
	}

	return buildExecutionSnapshot(*latest), nil
}

func (p *ExecutionProvider) Metrics() (any, error) {
	active := p.store.Active()
	history := p.store.List(1000)

	response := map[string]any{
		"active": len(active),
		"stored": len(history),
	}

	if len(history) == 0 {
		return response, nil
	}

	states := make(map[string]int)

	var completed int
	var failed int

	var totalDuration time.Duration
	var durationCount int

	var totalSteps int
	var completedSteps int

	for _, execution := range history {
		states[string(execution.State)]++

		if execution.Error != "" {
			failed++
		}

		if execution.CompletedAt != nil {
			completed++

			if !execution.StartedAt.IsZero() {
				duration := execution.CompletedAt.Sub(
					execution.StartedAt,
				)

				if duration >= 0 {
					totalDuration += duration
					durationCount++
				}
			}
		}

		totalSteps += execution.TotalSteps
		completedSteps += execution.ActualSteps
	}

	response["states"] = states
	response["completed"] = completed
	response["failed"] = failed

	if durationCount > 0 {
		response["averageDuration"] =
			totalDuration / time.Duration(durationCount)

		response["totalDuration"] = totalDuration
	}

	if totalSteps > 0 {
		response["totalSteps"] = totalSteps
		response["actualSteps"] = completedSteps
	}

	latest, ok := p.store.Latest()

	if ok {
		response["latestID"] = latest.ID
		response["latestState"] = latest.State

		if latest.CompletedAt != nil &&
			!latest.StartedAt.IsZero() {

			duration := latest.CompletedAt.Sub(
				latest.StartedAt,
			)

			if duration >= 0 {
				response["latestDuration"] = duration
			}
		}
	}

	return response, nil
}

func buildExecutionSnapshot(
	execution executionDomain.ExecutionSnapshot,
) map[string]any {
	result := map[string]any{
		"id":            execution.ID,
		"opportunityID": execution.OpportunityID,
		"state":         execution.State,
		"currentStep":   execution.CurrentStep,
		"totalSteps":    execution.TotalSteps,
		"actualSteps":   execution.ActualSteps,
		"startedAt":     formatTime(execution.StartedAt),
		"updatedAt":     formatTime(execution.UpdatedAt),
	}

	if execution.RecoveryID != "" {
		result["recoveryID"] = execution.RecoveryID
	}

	if execution.CompletedAt != nil {
		result["completedAt"] = formatTime(
			*execution.CompletedAt,
		)

		if !execution.StartedAt.IsZero() {
			duration := execution.CompletedAt.Sub(
				execution.StartedAt,
			)

			if duration >= 0 {
				result["duration"] = duration
			}
		}
	}

	if execution.Error != "" {
		result["error"] = execution.Error
	}

	result["initialState"] = buildBalanceState(
		execution.InitialState,
	)

	result["actualState"] = buildBalanceState(
		execution.ActualState,
	)

	result["steps"] = buildStepSnapshots(
		execution.Steps,
	)

	return result
}

func buildBalanceState(
	balances []executionDomain.BalanceSnapshot,
) []any {
	result := make([]any, 0, len(balances))

	for _, balance := range balances {
		result = append(
			result,
			map[string]any{
				"asset":    balance.Asset,
				"quantity": balance.Quantity,
			},
		)
	}

	return result
}

func buildStepSnapshots(
	steps []executionDomain.ExecutionStepSnapshot,
) []any {
	result := make([]any, 0, len(steps))

	for _, step := range steps {
		item := map[string]any{
			"stepID":         step.StepID,
			"plannedInput":   step.PlannedInput,
			"requestedInput": step.RequestedInput,
			"actualInput":    step.ActualInput,
			"grossOutput":    step.GrossOutput,
			"fee":            step.Fee,
			"netOutput":      step.NetOutput,
			"fillRatio":      step.FillRatio,
			"result":         step.Result,
			"startedAt":      formatTime(step.StartedAt),
			"completedAt":    formatTime(step.CompletedAt),
		}

		if !step.StartedAt.IsZero() &&
			!step.CompletedAt.IsZero() {

			duration := step.CompletedAt.Sub(
				step.StartedAt,
			)

			if duration >= 0 {
				item["duration"] = duration
			}
		}

		if step.Error != "" {
			item["error"] = step.Error
		}

		result = append(result, item)
	}

	return result
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(time.RFC3339Nano)
}
