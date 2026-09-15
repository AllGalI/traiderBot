package application

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

type ExecutionManager struct {
	exchangeAdapter executionDomain.ExchangeAdapter
	resultMapper    executionDomain.ActualStepResultMapper
	evaluator       Evaluator

	orderCounter uint64
}

func NewExecutionManager(
	exchangeAdapter executionDomain.ExchangeAdapter,
	resultMapper executionDomain.ActualStepResultMapper,
	evaluator Evaluator,
) *ExecutionManager {
	return &ExecutionManager{
		exchangeAdapter: exchangeAdapter,
		resultMapper:    resultMapper,
		evaluator:       evaluator,
	}
}

func (m *ExecutionManager) Execute(
	ctx context.Context,
	opportunity simulationDomain.Opportunity,
) (*executionDomain.Execution, error) {
	if m.exchangeAdapter == nil {
		return nil, fmt.Errorf("exchange adapter is not configured")
	}

	if len(opportunity.Route.Edges) == 0 {
		return nil, fmt.Errorf("opportunity route has no edges")
	}

	execution := executionDomain.NewExecution(opportunity)

	if err := execution.StartPreparing(); err != nil {
		return execution, err
	}

	steps := m.buildExecutionSteps(opportunity)

	currentAmount := opportunity.RequestedStartAmount

	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return execution, err
		}

		if err := execution.StartExecutingStep(); err != nil {
			return execution, err
		}

		request, err := m.buildOrderRequest(
			step,
			currentAmount,
		)
		if err != nil {
			return execution, err
		}

		result, err := m.exchangeAdapter.Execute(
			ctx,
			request,
		)
		if err != nil {
			return execution, fmt.Errorf(
				"execute step %d: %w",
				step.ID,
				err,
			)
		}

		if err := m.validateOrderResult(result); err != nil {
			return execution, fmt.Errorf(
				"step %d returned invalid order result: %w",
				step.ID,
				err,
			)
		}

		switch result.Status {
		case executionDomain.OrderStatusFilled:
			// Order is completely filled.
			// Continue with actual result evaluation.

		case executionDomain.OrderStatusPending,
			executionDomain.OrderStatusUnknown:

			if err := execution.RequireReconciliation(); err != nil {
				return execution, fmt.Errorf(
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
				return execution, fmt.Errorf(
					"reconcile step %d order %q: %w",
					step.ID,
					request.ClientOrderID,
					err,
				)
			}

			if err := m.validateOrderResult(reconciledResult); err != nil {
				return execution, fmt.Errorf(
					"reconciled step %d returned invalid order result: %w",
					step.ID,
					err,
				)
			}

			// Primary execution does not resume after reconciliation.
			// Recovery will work with the actual state.
			return execution, fmt.Errorf(
				"step %d requires recovery after reconciliation: status %s",
				step.ID,
				reconciledResult.Status,
			)

		case executionDomain.OrderStatusPartiallyFilled,
			executionDomain.OrderStatusRejected,
			executionDomain.OrderStatusCancelled:

			if err := execution.StartEvaluating(); err != nil {
				return execution, fmt.Errorf(
					"start evaluation for recovery on step %d: %w",
					step.ID,
					err,
				)
			}

			if err := execution.ApplyDecision(
				executionDomain.DecisionRecoveryRequired,
			); err != nil {
				return execution, fmt.Errorf(
					"apply recovery decision for step %d: %w",
					step.ID,
					err,
				)
			}

			// Recovery is not connected yet.
			return execution, fmt.Errorf(
				"step %d requires recovery: order status %s",
				step.ID,
				result.Status,
			)

		default:
			return execution, fmt.Errorf(
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
			return execution, fmt.Errorf(
				"map actual result for step %d: %w",
				step.ID,
				err,
			)
		}

		execution.AddActualStepResult(actualResult)

		currentAmount = actualResult.Output.Quantity

		if step.ID >= len(steps)-1 {
			continue
		}

		if err := execution.StartEvaluating(); err != nil {
			return execution, err
		}

		decision := m.evaluator.Evaluate(
			opportunity.Steps[step.ID],
			actualResult,
		)

		if err := execution.ApplyDecision(decision); err != nil {
			return execution, err
		}

		if decision == executionDomain.DecisionRecoveryRequired {
			// Recovery is not connected yet.
			return execution, fmt.Errorf(
				"step %d requires recovery after evaluation",
				step.ID,
			)
		}
	}

	return execution, nil
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
	clientOrderID := m.nextOrderID(step.ID)

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

func (m *ExecutionManager) nextOrderID(stepID int) string {
	id := atomic.AddUint64(
		&m.orderCounter,
		1,
	)

	return fmt.Sprintf(
		"step-%d-order-%d",
		stepID,
		id,
	)
}