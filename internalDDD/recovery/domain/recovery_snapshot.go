package domain

import (
	"time"

	executionDomain "traiderBot/internalDDD/execution/domain"
)

type RecoverySnapshot struct {

	ID RecoveryID

	ExecutionID executionDomain.ExecutionID


	ProcessState RecoveryProcessState


	InitialState State
	ActualState State


	StartedAt time.Time
	UpdatedAt time.Time
	CompletedAt *time.Time


	Attempts int
	PlansCreated int
	PlansExecuted int


	Error string
}