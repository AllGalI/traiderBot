package application

import (
	"fmt"

	executionDomain "traiderBot/internalDDD/execution/domain"

	"github.com/shopspring/decimal"
)

type DefaultExecutionPolicy struct {
	minReceivedRatio decimal.Decimal
}

func NewDefaultExecutionPolicy(
	minReceivedRatio decimal.Decimal,
) *DefaultExecutionPolicy {
	return &DefaultExecutionPolicy{
		minReceivedRatio: minReceivedRatio,
	}
}

func (p *DefaultExecutionPolicy) Decide(
	execution *executionDomain.Execution,
) executionDomain.ExecutionDecision {
	if execution == nil {
		return executionDomain.ExecutionDecision{
			Type:   executionDomain.DecisionAbort,
			Reason: "execution is nil",
		}
	}

	order := execution.CurrentOrder()

	if order == nil {
		return executionDomain.ExecutionDecision{
			Type: executionDomain.DecisionContinue,
		}
	}

	if order.ExpectedAmountOut.IsZero() {
		return executionDomain.ExecutionDecision{
			Type:   executionDomain.DecisionAbort,
			Reason: "expected amount out is zero",
		}
	}

	ratio := order.ReceivedAmount.Div(order.ExpectedAmountOut)

	if ratio.LessThan(p.minReceivedRatio) {
		return executionDomain.ExecutionDecision{
			Type: executionDomain.DecisionAbort,
			Reason: fmt.Sprintf(
				"received amount ratio %s is below minimum %s",
				ratio,
				p.minReceivedRatio,
			),
		}
	}

	return executionDomain.ExecutionDecision{
		Type: executionDomain.DecisionContinue,
	}
}
