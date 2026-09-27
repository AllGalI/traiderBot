package domain

import (
	"time"

	simulationDomain "traiderBot/internalDDD/simulation/domain"

)

type ExecutionSnapshot struct {
	ID ExecutionID

	OpportunityID simulationDomain.OpportunityID
	RecoveryID string

	State ExecutionState

	StartedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time

	CurrentStep int
	TotalSteps  int

	ActualSteps int

	Error string
}
