package domain

import (
	executionDomain "traiderBot/internalDDD/execution/domain"

	"github.com/shopspring/decimal"
)

type State map[string]decimal.Decimal

type RecoveryInput struct {
	ExecutionID executionDomain.ExecutionID
	Reason      RecoveryReason

	InitialState State
	ActualState  State
}

func NewRecoveryInput(
	executionID executionDomain.ExecutionID,
	initial State,
	actual State,
) RecoveryInput {

	return RecoveryInput{
		ExecutionID: executionID,

		InitialState: initial,
		ActualState:  actual,
	}
}
