package domain

import (
	"time"

	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

type BalanceSnapshot struct {
	Asset    string
	Quantity decimal.Decimal
}

type ExecutionStepSnapshot struct {
	StepID          int
	PlannedInput    AssetAmount
	RequestedInput  AssetAmount
	ActualInput     AssetAmount
	GrossOutput     AssetAmount
	Fee             AssetAmount
	NetOutput       AssetAmount
	FillRatio       decimal.Decimal
	Result          OrderStatus
	Error           string
	StartedAt       time.Time
	CompletedAt     time.Time
}

type ExecutionSnapshot struct {
	ID            ExecutionID
	OpportunityID simulationDomain.OpportunityID
	RecoveryID    string
	State         ExecutionState

	StartedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time

	CurrentStep int
	TotalSteps  int
	ActualSteps int

	InitialState []BalanceSnapshot
	ActualState  []BalanceSnapshot

	Steps []ExecutionStepSnapshot

	Error string
}