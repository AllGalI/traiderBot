package em

import (
	"context"
	"fmt"
	"time"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	executionDomain "traiderBot/internalDDD/execution/domain"
)

func (m *ExecutionManager) reconcileIfRequired(
	ctx context.Context,
	step executionDomain.ExecutionStep,
	request exchangeDomain.OrderRequest,
	result executionDomain.OrderResult,
	completedAt time.Time,
) (
	executionDomain.OrderResult,
	time.Time,
	error,
) {
	switch result.Status {
	case executionDomain.OrderStatusFilled:
		return result, completedAt, nil

	case executionDomain.OrderStatusPartiallyFilled:
		return result, completedAt, nil

	case executionDomain.OrderStatusRejected:
		return result, completedAt, fmt.Errorf(
			"step %d requires recovery: order status %s",
			step.ID,
			result.Status,
		)

	case executionDomain.OrderStatusCancelled:
		return result, completedAt, fmt.Errorf(
			"step %d requires recovery: order status %s",
			step.ID,
			result.Status,
		)

	case executionDomain.OrderStatusPending,
		executionDomain.OrderStatusUnknown:

		reconciledResult, err := m.exchangeAdapter.GetOrderByClientID(
			ctx,
			request,
		)
		if err != nil {
			return result, completedAt, fmt.Errorf(
				"reconcile step %d order %q: %w",
				step.ID,
				request.ClientOrderID,
				err,
			)
		}

		if err := m.validateOrderResult(reconciledResult); err != nil {
			return result, completedAt, fmt.Errorf(
				"reconciled step %d returned invalid order result: %w",
				step.ID,
				err,
			)
		}

		completedAt = time.Now()

		switch reconciledResult.Status {
		case executionDomain.OrderStatusFilled,
			executionDomain.OrderStatusPartiallyFilled:
			return reconciledResult, completedAt, nil

		case executionDomain.OrderStatusRejected,
			executionDomain.OrderStatusCancelled:
			return result, completedAt, fmt.Errorf(
				"step %d requires recovery after reconciliation: status %s",
				step.ID,
				reconciledResult.Status,
			)

		case executionDomain.OrderStatusPending,
			executionDomain.OrderStatusUnknown:
			return result, completedAt, fmt.Errorf(
				"step %d remains unresolved after reconciliation: status %s",
				step.ID,
				reconciledResult.Status,
			)

		default:
			return result, completedAt, fmt.Errorf(
				"step %d returned unsupported reconciled order status: %s",
				step.ID,
				reconciledResult.Status,
			)
		}

	default:
		return result, completedAt, fmt.Errorf(
			"step %d returned unsupported order status: %s",
			step.ID,
			result.Status,
		)
	}
}