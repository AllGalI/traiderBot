package application

import (
	"context"
	"fmt"
	"time"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	executionDomain "traiderBot/internalDDD/execution/domain"
	recoveryApplication "traiderBot/internalDDD/recovery/application"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

type ExecutionManager struct {
	exchangeAdapter  exchangeDomain.Adapter
	resultMapper     executionDomain.ActualStepResultMapper
	evaluator        Evaluator
	orderIDGenerator *exchangeDomain.OrderIDGenerator
	recoveryService  recoveryApplication.RecoveryService

	executionStore ExecutionStore
}

func NewExecutionManager(
	exchangeAdapter exchangeDomain.Adapter,
	resultMapper executionDomain.ActualStepResultMapper,
	evaluator Evaluator,
	orderIDGenerator *exchangeDomain.OrderIDGenerator,
	executionStore ExecutionStore,
) *ExecutionManager {
	return &ExecutionManager{
		exchangeAdapter:  exchangeAdapter,
		resultMapper:     resultMapper,
		evaluator:        evaluator,
		orderIDGenerator: orderIDGenerator,
		executionStore:   executionStore,
	}
}

func (m *ExecutionManager) Execute(
	ctx context.Context,
	opportunity simulationDomain.Opportunity,
	marketContext recoveryApplication.MarketContext,
) (
	execution *executionDomain.Execution,
	actualState recoveryDomain.State,
	err error,
) {
	if m.exchangeAdapter == nil {
		return nil, nil, fmt.Errorf("exchange adapter is not configured")
	}

	if m.orderIDGenerator == nil {
		return nil, nil, fmt.Errorf("order ID generator is not configured")
	}

	if len(opportunity.Route.Edges) == 0 {
		return nil, nil, fmt.Errorf("opportunity route has no edges")
	}

	execution = executionDomain.NewExecution(opportunity)

	if err = execution.StartPreparing(); err != nil {
		return execution, nil, err
	}

	steps := m.buildExecutionSteps(opportunity)
	initialState := m.buildInitialState(opportunity)
	actualState = recoveryApplication.CloneState(initialState)

	defer func() {
		if execution == nil {
			return
		}

		if err == nil {
			return
		}

		if execution.IsFinished() {
			return
		}

		m.failExecution(
			execution,
			len(steps),
			initialState,
			actualState,
			err,
		)
	}()

	m.createExecution(
		execution,
		0,
		len(steps),
		initialState,
		actualState,
		nil,
	)

	currentAmount := opportunity.ExecutedStartAmount

	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return execution, actualState, err
		}

		if err := execution.StartExecutingStep(); err != nil {
			return execution, actualState, err
		}

		m.persistExecution(
			execution,
			step.ID,
			len(steps),
			initialState,
			actualState,
			nil,
		)

		requestAmount := currentAmount

		if step.ID == 0 {
			requestAmount = step.PlannedInput.Quantity
		}

		request, err := m.buildOrderRequest(
			step,
			requestAmount,
		)
		if err != nil {
			return execution, actualState, err
		}

		startedAt := time.Now()

		result, err := m.exchangeAdapter.Execute(
			ctx,
			request,
		)

		if err != nil {
			return execution, actualState, fmt.Errorf(
				"execute step %d: %w",
				step.ID,
				err,
			)
		}

		completedAt := time.Now()

		if err := m.validateOrderResult(result); err != nil {
			return execution, actualState, fmt.Errorf(
				"step %d returned invalid order result: %w",
				step.ID,
				err,
			)
		}

		switch result.Status {
		case executionDomain.OrderStatusFilled:
			// Final result is already available.
			// Continue below and persist the actual step result.

		case executionDomain.OrderStatusPending,
			executionDomain.OrderStatusUnknown:

			if err := execution.RequireReconciliation(); err != nil {
				return execution, actualState, fmt.Errorf(
					"require reconciliation for step %d: %w",
					step.ID,
					err,
				)
			}

			reconciledResult, err := m.exchangeAdapter.GetOrderByClientID(
				ctx,
				request,
			)
			if err != nil {
				return execution, actualState, fmt.Errorf(
					"reconcile step %d order %q: %w",
					step.ID,
					request.ClientOrderID,
					err,
				)
			}

			if err := m.validateOrderResult(reconciledResult); err != nil {
				return execution, actualState, fmt.Errorf(
					"reconciled step %d returned invalid order result: %w",
					step.ID,
					err,
				)
			}

			result = reconciledResult
			completedAt = time.Now()

			switch reconciledResult.Status {
			case executionDomain.OrderStatusFilled:
			case executionDomain.OrderStatusPartiallyFilled:

			case executionDomain.OrderStatusRejected:

				return execution, actualState, fmt.Errorf(
					"step %d requires recovery after reconciliation: status %s",
					step.ID,
					reconciledResult.Status,
				)

			case executionDomain.OrderStatusCancelled:
				return execution, actualState, fmt.Errorf(
					"step %d requires recovery after reconciliation: status %s",
					step.ID,
					reconciledResult.Status,
				)

			case executionDomain.OrderStatusPending,
				executionDomain.OrderStatusUnknown:

				return execution, actualState, fmt.Errorf(
					"step %d remains unresolved after reconciliation: status %s",
					step.ID,
					reconciledResult.Status,
				)

			default:
				return execution, actualState, fmt.Errorf(
					"step %d returned unsupported reconciled order status: %s",
					step.ID,
					reconciledResult.Status,
				)
			}

		case executionDomain.OrderStatusPartiallyFilled:

		case executionDomain.OrderStatusRejected:
			return execution, actualState, fmt.Errorf(
				"step %d requires recovery: order status %s",
				step.ID,
				result.Status,
			)

		case executionDomain.OrderStatusCancelled:
			return execution, actualState, fmt.Errorf(
				"step %d requires recovery: order status %s",
				step.ID,
				result.Status,
			)

		default:
			return execution, actualState, fmt.Errorf(
				"step %d returned unsupported order status: %s",
				step.ID,
				result.Status,
			)
		}

		actualResult, err := m.resultMapper.Map(
			step,
			request,
			result,
			startedAt,
			completedAt,
		)
		if err != nil {
			return execution, actualState, fmt.Errorf(
				"map actual result for step %d: %w",
				step.ID,
				err,
			)
		}

		execution.AddActualStepResult(actualResult)

		actualState = m.applyOrderResult(
			actualState,
			step,
			result,
		)

		currentAmount = actualResult.NetOutput.Quantity

		if step.ID >= len(steps)-1 {
			continue
		}

		if err := execution.StartEvaluating(); err != nil {
			return execution, actualState, err
		}

		decision := m.evaluator.Evaluate(
			opportunity.Steps[step.ID],
			actualResult,
		)

		if err := execution.ApplyDecision(decision); err != nil {
			return execution, actualState, err
		}

		if decision == executionDomain.DecisionRecoveryRequired {
			if err := execution.StartEvaluating(); err != nil {
				return execution, actualState, err
			}
			// 	return m.runRecovery(
			// 		ctx,
			// 		execution,
			// 		initialState,
			// 		actualState,
			// 		marketContext,
			// 		recoveryDomain.RecoveryReasonDeviation,
			// 		fmt.Errorf(
			// 			"step %d requires recovery after evaluation",
			// 			step.ID,
			// 		),
			// 	)
		}
	}

	// fmt.Printf(
	// 	"INFO execution_completed opportunity_id=%s execution_id=%s state=%s\n",
	// 	opportunity.ID,
	// 	execution.ID(),
	// 	execution.State(),
	// )

	if err := m.completeExecution(
		execution,
		len(steps),
		initialState,
		actualState,
		nil,
	); err != nil {
		return execution, actualState, err
	}

	return execution, actualState, nil
}

// func (m *ExecutionManager) startRecovery(
// 	ctx context.Context,
// 	execution *executionDomain.Execution,
// 	initialState recoveryDomain.State,
// 	actualState recoveryDomain.State,
// 	marketContext recoveryApplication.MarketContext,
// 	reason recoveryDomain.RecoveryReason,
// 	cause error,
// ) (*executionDomain.Execution, recoveryDomain.State, error) {
// 	if err := execution.StartEvaluating(); err != nil {
// 		return execution, actualState, fmt.Errorf(
// 			"start recovery evaluation: %w",
// 			err,
// 		)
// 	}

// 	if err := execution.ApplyDecision(
// 		executionDomain.DecisionRecoveryRequired,
// 	); err != nil {
// 		return execution, actualState, fmt.Errorf(
// 			"apply recovery decision: %w",
// 			err,
// 		)
// 	}

// 	return m.runRecovery(
// 		ctx,
// 		execution,
// 		initialState,
// 		actualState,
// 		marketContext,
// 		reason,
// 		cause,
// 	)
// }

// func (m *ExecutionManager) runRecovery(
// 	ctx context.Context,
// 	execution *executionDomain.Execution,
// 	initialState recoveryDomain.State,
// 	actualState recoveryDomain.State,
// 	marketContext recoveryApplication.MarketContext,
// 	reason recoveryDomain.RecoveryReason,
// 	cause error,
// ) (*executionDomain.Execution, recoveryDomain.State, error) {
// 	recoveryID := recoveryDomain.NewRecoveryID()

// 	execution.AttachRecovery(
// 		recoveryID,
// 	)

// 	recoveredState, err := m.recoveryService.Recover(
// 		ctx,
// 		recoveryDomain.RecoveryInput{
// 			ExecutionID: execution.ID(),
// 			Reason:      reason,

// 			InitialState: initialState,
// 			ActualState:  actualState,
// 		},
// 		marketContext,
// 	)

// 	if err != nil {
// 		return execution, recoveredState, fmt.Errorf(
// 			"%w: recovery %s failed: %w",
// 			cause,
// 			recoveryID,
// 			err,
// 		)
// 	}

// 	if err := m.completeExecution(
// 		execution,
// 		len(execution.Opportunity().Route.Edges),
// 		initialState,
// 		actualState,
// 		nil,
// 	); err != nil {
// 		return execution, recoveredState, err
// 	}

// 	return execution, recoveredState, nil
// }

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

func (m *ExecutionManager) buildOrderRequest(
	step executionDomain.ExecutionStep,
	amount decimal.Decimal,
) (executionDomain.OrderRequest, error) {
	clientOrderID := m.orderIDGenerator.Next(step.ID)

	return executionDomain.NewOrderRequest(
		step.Edge,
		amount,
		clientOrderID,
	)
}

func (m *ExecutionManager) validateOrderResult(
	result executionDomain.OrderResult,
) error {
	if result.ExecutedAmount.IsNegative() {
		return fmt.Errorf(
			"executed amount must not be negative",
		)
	}

	if result.ReceivedAmount.IsNegative() {
		return fmt.Errorf(
			"received amount must not be negative",
		)
	}

	if result.Fee.IsNegative() {
		return fmt.Errorf(
			"fee must not be negative",
		)
	}

	return nil
}

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

	_ = m.executionStore.Create(
		&snapshot,
	)
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

	_ = m.executionStore.Update(
		&snapshot,
	)
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

	// if err := execution.Complete(); err != nil {
	// 	return err
	// }

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

	_ = m.executionStore.Complete(
		&snapshot,
	)

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

	_ = m.executionStore.Complete(
		&snapshot,
	)

	return nil
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
