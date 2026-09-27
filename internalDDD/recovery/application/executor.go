package application

import (
	"context"
	"fmt"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"

	"github.com/shopspring/decimal"
)

type RecoveryExecutor interface {
	Execute(
		ctx context.Context,
		plan recoveryDomain.RecoveryPlan,
		state recoveryDomain.State,
	) (recoveryDomain.State, error)
}

type Executor struct {
	exchangeAdapter  exchangeDomain.Adapter
	orderIDGenerator *exchangeDomain.OrderIDGenerator
}

func NewExecutor(
	exchangeAdapter exchangeDomain.Adapter,
	orderIDGenerator *exchangeDomain.OrderIDGenerator,
) Executor {
	return Executor{
		exchangeAdapter:  exchangeAdapter,
		orderIDGenerator: orderIDGenerator,
	}
}

func (e Executor) Execute(
	ctx context.Context,
	plan recoveryDomain.RecoveryPlan,
	state recoveryDomain.State,
) (recoveryDomain.State, error) {
	if e.exchangeAdapter == nil {
		return nil, fmt.Errorf("exchange adapter is required")
	}

	if e.orderIDGenerator == nil {
		return nil, fmt.Errorf("order ID generator is required")
	}

	if len(plan.Route.Edges) == 0 {
		return nil, fmt.Errorf("recovery route has no edges")
	}

	actualState := CloneState(state)

	for stepID, edge := range plan.Route.Edges {
		if err := ctx.Err(); err != nil {
			return actualState, err
		}

		amount := actualState[edge.From]

		if amount.LessThanOrEqual(decimal.Zero) {
			return actualState, fmt.Errorf(
				"recovery asset %s has no amount to execute",
				edge.From,
			)
		}

		clientOrderID := e.orderIDGenerator.Next(stepID)

		request, err := exchangeDomain.NewOrderRequest(
			edge,
			amount,
			clientOrderID,
		)
		if err != nil {
			return actualState, fmt.Errorf(
				"create recovery order for %s -> %s: %w",
				edge.From,
				edge.To,
				err,
			)
		}

		result, err := e.exchangeAdapter.Execute(ctx, request)
		if err != nil {
			return actualState, fmt.Errorf(
				"execute recovery step %d: %w",
				stepID,
				err,
			)
		}

		if err := validateOrderResult(result); err != nil {
			return actualState, fmt.Errorf(
				"recovery step %d returned invalid order result: %w",
				stepID,
				err,
			)
		}

		if result.Status == exchangeDomain.OrderStatusPending ||
			result.Status == exchangeDomain.OrderStatusUnknown {

			result, err = e.reconcile(ctx, request)
			if err != nil {
				return actualState, fmt.Errorf(
					"reconcile recovery step %d: %w",
					stepID,
					err,
				)
			}

			if err := validateOrderResult(result); err != nil {
				return actualState, fmt.Errorf(
					"reconciled recovery step %d returned invalid order result: %w",
					stepID,
					err,
				)
			}
		}

		switch result.Status {
		case exchangeDomain.OrderStatusFilled:
			applyOrderResult(actualState, edge, result)

		case exchangeDomain.OrderStatusPartiallyFilled:
			applyOrderResult(actualState, edge, result)

			return actualState, fmt.Errorf(
				"recovery step %d partially filled",
				stepID,
			)

		case exchangeDomain.OrderStatusRejected:
			return actualState, fmt.Errorf(
				"recovery step %d was rejected",
				stepID,
			)

		case exchangeDomain.OrderStatusCancelled:
			return actualState, fmt.Errorf(
				"recovery step %d was cancelled",
				stepID,
			)

		case exchangeDomain.OrderStatusPending,
			exchangeDomain.OrderStatusUnknown:
			return actualState, fmt.Errorf(
				"recovery step %d remains unresolved after reconciliation: status %s",
				stepID,
				result.Status,
			)

		default:
			return actualState, fmt.Errorf(
				"recovery step %d returned unsupported order status: %s",
				stepID,
				result.Status,
			)
		}
	}

	return actualState, nil
}

func (e Executor) reconcile(
	ctx context.Context,
	request exchangeDomain.OrderRequest,
) (exchangeDomain.OrderResult, error) {
	return e.exchangeAdapter.GetOrderByClientID(
		ctx,
		request,
	)
}

func applyOrderResult(
	state recoveryDomain.State,
	edge marketDomain.GraphEdge,
	result exchangeDomain.OrderResult,
) {
	state[edge.From] = state[edge.From].Sub(
		result.ExecutedAmount,
	)

	state[edge.To] = state[edge.To].Add(
		result.ReceivedAmount,
	)
}

func validateOrderResult(
	result exchangeDomain.OrderResult,
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
