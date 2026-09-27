package domain

import (
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"
)

type RecoveryAttemptSnapshot struct {
	Index int

	PlansCreated   int
	PlansAttempted int
	PlansSucceeded int
	PlansFailed    int

	StateBefore State
	StateAfter  State

	Changed bool

	StartedAt   time.Time
	CompletedAt *time.Time

	Error string
}

type RecoverySnapshot struct {
	ID string

	ExecutionID executionDomain.ExecutionID

	ProcessState RecoveryProcessState
	Reason       RecoveryReason

	InitialState State
	ActualState  State

	StartedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time

	Attempts      []RecoveryAttemptSnapshot
	PlansCreated  int
	PlansExecuted int

	Error string
}