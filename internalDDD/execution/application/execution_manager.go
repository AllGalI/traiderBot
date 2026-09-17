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
	exchangeAdapter  executionDomain.ExchangeAdapter
	resultMapper     executionDomain.ActualStepResultMapper
	evaluator        Evaluator
	orderIDGenerator *exchangeDomain.OrderIDGenerator
	recoveryService  recoveryApplication.RecoveryService
}

func NewExecutionManager(
	exchangeAdapter executionDomain.ExchangeAdapter,
	resultMapper executionDomain.ActualStepResultMapper,
	evaluator Evaluator,
	orderIDGenerator *exchangeDomain.OrderIDGenerator,
	recoveryService recoveryApplication.RecoveryService,
) *ExecutionManager {
	return &ExecutionManager{
		exchangeAdapter:  exchangeAdapter,
		resultMapper:     resultMapper,
		evaluator:        evaluator,
		orderIDGenerator: orderIDGenerator,
		recoveryService:  recoveryService,
	}
}

func (m *ExecutionManager) Execute(
	ctx context.Context,
	opportunity simulationDomain.Opportunity,
	marketContext recoveryApplication.MarketContext,
) (*executionDomain.Execution, recoveryDomain.State, error) {
	if m.exchangeAdapter == nil {
		return nil, nil, fmt.Errorf("exchange adapter is not configured")
	}

	if m.orderIDGenerator == nil {
		return nil, nil, fmt.Errorf("order ID generator is not configured")
	}

	if len(opportunity.Route.Edges) == 0 {
		return nil, nil, fmt.Errorf("opportunity route has no edges")
	}

	execution := executionDomain.NewExecution(opportunity)

	if err := execution.StartPreparing(); err != nil {
		return execution, nil, err
	}

	steps := m.buildExecutionSteps(opportunity)

	initialState := m.buildInitialState(opportunity)
	actualState := recoveryApplication.CloneState(initialState)

	currentAmount := opportunity.RequestedStartAmount

	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return execution, actualState, err
		}

		if err := execution.StartExecutingStep(); err != nil {
			return execution, actualState, err
		}

		request, err := m.buildOrderRequest(
			step,
			currentAmount,
		)
		if err != nil {
			return execution, actualState, err
		}

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

		if err := m.validateOrderResult(result); err != nil {
			return execution, actualState, fmt.Errorf(
				"step %d returned invalid order result: %w",
				step.ID,
				err,
			)
		}

		switch result.Status {
		case executionDomain.OrderStatusFilled:
			// Continue below and process the actual result.

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

			switch reconciledResult.Status {
			case executionDomain.OrderStatusFilled:
				result = reconciledResult

			case executionDomain.OrderStatusPartiallyFilled:
				actualState = m.applyOrderResult(
					actualState,
					step,
					reconciledResult,
				)

				return m.startRecovery(
					ctx,
					execution,
					initialState,
					actualState,
					marketContext,
					fmt.Errorf(
						"step %d requires recovery after reconciliation: status %s",
						step.ID,
						reconciledResult.Status,
					),
				)

			case executionDomain.OrderStatusRejected,
				executionDomain.OrderStatusCancelled:

				return m.startRecovery(
					ctx,
					execution,
					initialState,
					actualState,
					marketContext,
					fmt.Errorf(
						"step %d requires recovery after reconciliation: status %s",
						step.ID,
						reconciledResult.Status,
					),
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
			actualState = m.applyOrderResult(
				actualState,
				step,
				result,
			)

			return m.startRecovery(
				ctx,
				execution,
				initialState,
				actualState,
				marketContext,
				fmt.Errorf(
					"step %d requires recovery: order status %s",
					step.ID,
					result.Status,
				),
			)

		case executionDomain.OrderStatusRejected,
			executionDomain.OrderStatusCancelled:

			return m.startRecovery(
				ctx,
				execution,
				initialState,
				actualState,
				marketContext,
				fmt.Errorf(
					"step %d requires recovery: order status %s",
					step.ID,
					result.Status,
				),
			)

		default:
			return execution, actualState, fmt.Errorf(
				"step %d returned unsupported order status: %s",
				step.ID,
				result.Status,
			)
		}

		completedAt := time.Now()

		actualResult, err := m.resultMapper.Map(
			step,
			result,
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

		currentAmount = actualResult.Output.Quantity

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
			return m.runRecovery(
				ctx,
				execution,
				initialState,
				actualState,
				marketContext,
				fmt.Errorf(
					"step %d requires recovery after evaluation",
					step.ID,
				),
			)
		}
	}

	return execution, actualState, nil
}

func (m *ExecutionManager) startRecovery(
	ctx context.Context,
	execution *executionDomain.Execution,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	marketContext recoveryApplication.MarketContext,
	cause error,
) (*executionDomain.Execution, recoveryDomain.State, error) {
	if err := execution.StartEvaluating(); err != nil {
		return execution, actualState, fmt.Errorf(
			"start recovery evaluation: %w",
			err,
		)
	}

	if err := execution.ApplyDecision(
		executionDomain.DecisionRecoveryRequired,
	); err != nil {
		return execution, actualState, fmt.Errorf(
			"apply recovery decision: %w",
			err,
		)
	}

	return m.runRecovery(
		ctx,
		execution,
		initialState,
		actualState,
		marketContext,
		cause,
	)
}

func (m *ExecutionManager) runRecovery(
	ctx context.Context,
	execution *executionDomain.Execution,
	initialState recoveryDomain.State,
	actualState recoveryDomain.State,
	marketContext recoveryApplication.MarketContext,
	cause error,
) (*executionDomain.Execution, recoveryDomain.State, error) {
	recoveredState, err := m.recoveryService.Recover(
		ctx,
		recoveryDomain.RecoveryInput{
			InitialState: initialState,
			ActualState:  actualState,
		},
		marketContext,
	)
	if err != nil {
		return execution, recoveredState, fmt.Errorf(
			"%w: recovery failed: %w",
			cause,
			err,
		)
	}

	return execution, recoveredState, nil
}

func (m *ExecutionManager) buildInitialState(
	opportunity simulationDomain.Opportunity,
) recoveryDomain.State {
	state := make(recoveryDomain.State)

	startAsset := opportunity.Route.Edges[0].From

	state[startAsset] = opportunity.RequestedStartAmount

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

	return nextState
}

func (m *ExecutionManager) buildExecutionSteps(
	opportunity simulationDomain.Opportunity,
) []executionDomain.ExecutionStep {
	steps := make(
		[]executionDomain.ExecutionStep,
		0,
		len(opportunity.Route.Edges),
	)

	for index, edge := range opportunity.Route.Edges {
		steps = append(
			steps,
			executionDomain.ExecutionStep{
				ID:   index,
				Edge: edge,
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