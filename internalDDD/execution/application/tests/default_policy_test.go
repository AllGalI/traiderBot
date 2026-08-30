package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
)

func TestDefaultExecutionPolicyContinuesWithoutCurrentOrder(t *testing.T) {
	policy := executionApplication.NewDefaultExecutionPolicy(
		decimal.NewFromFloat(0.90),
	)

	execution := &executionDomain.Execution{}

	decision := policy.Decide(execution)

	if decision.Type != executionDomain.DecisionContinue {
		t.Fatalf(
			"expected continue, got %v",
			decision.Type,
		)
	}
}

func TestDefaultExecutionPolicyContinuesWhenReceivedAmountIsAtLeastThreshold(t *testing.T) {
	policy := executionApplication.NewDefaultExecutionPolicy(
		decimal.NewFromFloat(0.90),
	)

	execution := &executionDomain.Execution{
		Orders: []executionDomain.Order{
			{
				ExpectedAmountOut: decimal.NewFromInt(100),
				ReceivedAmount:    decimal.NewFromInt(90),
			},
		},
	}

	decision := policy.Decide(execution)

	if decision.Type != executionDomain.DecisionContinue {
		t.Fatalf(
			"expected continue, got %v",
			decision.Type,
		)
	}
}

func TestDefaultExecutionPolicyAbortsWhenReceivedAmountIsBelowThreshold(t *testing.T) {
	policy := executionApplication.NewDefaultExecutionPolicy(
		decimal.NewFromFloat(0.90),
	)

	execution := &executionDomain.Execution{
		Orders: []executionDomain.Order{
			{
				ExpectedAmountOut: decimal.NewFromInt(100),
				ReceivedAmount:    decimal.NewFromInt(89),
			},
		},
	}

	decision := policy.Decide(execution)

	if decision.Type != executionDomain.DecisionAbort {
		t.Fatalf(
			"expected abort, got %v",
			decision.Type,
		)
	}

	if decision.Reason == "" {
		t.Fatal("expected abort reason, got empty reason")
	}
}

func TestDefaultExecutionPolicyAbortsWhenExecutionIsNil(t *testing.T) {
	policy := executionApplication.NewDefaultExecutionPolicy(
		decimal.NewFromFloat(0.90),
	)

	decision := policy.Decide(nil)

	if decision.Type != executionDomain.DecisionAbort {
		t.Fatalf(
			"expected abort, got %v",
			decision.Type,
		)
	}

	if decision.Reason == "" {
		t.Fatal("expected abort reason, got empty reason")
	}
}
