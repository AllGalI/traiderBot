package application

import (
	"context"
	"fmt"
	"sync/atomic"

	executionDomain "traiderBot/internalDDD/execution/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type ExecutionManager struct {
	executor   ExchangeExecutor
	orderQuery ExchangeOrderQuery
	policy     executionDomain.ExecutionPolicy

	counter uint64
}

func NewExecutionManager(
	executor ExchangeExecutor,
	orderQuery ExchangeOrderQuery,
	policy executionDomain.ExecutionPolicy,
) *ExecutionManager {
	return &ExecutionManager{
		executor:   executor,
		orderQuery: orderQuery,
		policy:     policy,
	}
}

func (m *ExecutionManager) Execute(
	ctx context.Context,
	opportunity *simulationDomain.Opportunity,
) (*executionDomain.Execution, error) {
	if opportunity == nil {
		return nil, fmt.Errorf("opportunity must not be nil")
	}

	if len(opportunity.Steps) == 0 {
		return nil, fmt.Errorf("opportunity has no steps")
	}

	if m.executor == nil {
		return nil, fmt.Errorf("executor is not configured")
	}

	if m.policy == nil {
		return nil, fmt.Errorf("execution policy is not configured")
	}

	execution := executionDomain.NewExecution(
		m.nextID(),
		*opportunity,
	)

	if err := execution.Start(); err != nil {
		return execution, err
	}

	for stepIndex, step := range opportunity.Steps {
		if err := ctx.Err(); err != nil {
			_ = execution.Abort()
			return execution, err
		}

		decision := m.policy.Decide(execution)

		if decision.Type == executionDomain.DecisionAbort {
			_ = execution.Abort()

			return execution, fmt.Errorf(
				"execution aborted: %s",
				decision.Reason,
			)
		}

		amountIn := execution.CurrentAmount

		if stepIndex == 0 {
			amountIn = step.AmountIn
		}

		clientOrderID := m.nextOrderID(
			execution.ID,
			stepIndex,
		)

		request := executionDomain.OrderRequest{
			StepIndex: stepIndex,

			Edge:              step.Edge,
			Amount:            amountIn,
			ExpectedAmountOut: step.AmountOut,

			ClientOrderID: clientOrderID,
		}

		result, err := m.executor.Execute(
			ctx,
			request,
		)

		if err != nil {
			executorErr := err

			reconciledResult, reconcileErr := m.reconcileOrder(
				ctx,
				request,
			)

			if reconcileErr != nil {
				_ = execution.Fail()

				return execution, fmt.Errorf(
					"execute step %d failed: %w; reconciliation failed: %v",
					stepIndex,
					executorErr,
					reconcileErr,
				)
			}

			result = reconciledResult
		}
		if err := m.handleOrderResult(
			execution,
			result,
		); err != nil {
			return execution, err
		}

		if execution.IsFinished() {
			return execution, nil
		}
	}

	if err := execution.Complete(
		execution.CurrentAmount,
	); err != nil {
		return execution, err
	}

	return execution, nil
}

func (m *ExecutionManager) handleOrderResult(
	execution *executionDomain.Execution,
	result executionDomain.OrderExecutionResult,
) error {
	order := executionDomain.NewOrder(
		result.StepIndex,
		result,
	)

	if err := execution.AddOrder(order); err != nil {
		_ = execution.Fail()
		return err
	}

	switch result.Status {
	case executionDomain.OrderFilled:
		if err := execution.AdvanceStep(
			result.ReceivedAmount,
		); err != nil {
			_ = execution.Fail()
			return err
		}

		return nil

	case executionDomain.OrderPartiallyFilled:
		_ = execution.Abort()

		return nil

	case executionDomain.OrderCancelled:
		_ = execution.Abort()

		return nil

	case executionDomain.OrderFailed:
		_ = execution.Abort()

		return nil

	default:
		_ = execution.Abort()

		return nil
	}
}

func (m *ExecutionManager) reconcileOrder(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if m.orderQuery == nil {
		return executionDomain.OrderExecutionResult{}, fmt.Errorf(
			"order query is not configured",
		)
	}

	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	return m.orderQuery.GetOrder(
		ctx,
		request.Edge.Symbol,
		request.ClientOrderID,
	)
}

func (m *ExecutionManager) nextID() string {
	id := atomic.AddUint64(&m.counter, 1)

	return fmt.Sprintf(
		"execution-%d",
		id,
	)
}

func (m *ExecutionManager) nextOrderID(
	executionID string,
	stepIndex int,
) string {
	id := atomic.AddUint64(&m.counter, 1)

	return fmt.Sprintf(
		"%s-order-%d-%d",
		executionID,
		stepIndex,
		id,
	)
}
