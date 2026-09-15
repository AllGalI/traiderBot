package domain

import "github.com/shopspring/decimal"

type State map[string]decimal.Decimal

type RecoveryInput struct {
	InitialState State
	ActualState  State

}

func NewRecoveryInput(
	initial State,
	actual State,
) RecoveryInput {

	return RecoveryInput{
		InitialState: initial,
		ActualState:  actual,
	}
}