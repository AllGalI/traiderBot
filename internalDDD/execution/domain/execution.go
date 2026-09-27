package domain

import (
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type ExecutionID string

type Execution struct {
	id                ExecutionID
	state             ExecutionState
	opportunity       simulationDomain.Opportunity
	actualStepResults []ActualStepResult

	recoveryID string
}

func NewExecution(
	opportunity simulationDomain.Opportunity,
) *Execution {
	return &Execution{
		id:                NewExecutionID(),
		state:             StateNotStarted,
		opportunity:       opportunity,
		actualStepResults: make([]ActualStepResult, 0),
	}
}

func (e *Execution) ID() ExecutionID {
	return e.id
}

func (e *Execution) State() ExecutionState {
	return e.state
}

func (e *Execution) Opportunity() simulationDomain.Opportunity {
	return e.opportunity
}

func (e *Execution) ActualStepResults() []ActualStepResult {
	return e.actualStepResults
}

func (e *Execution) AddActualStepResult(result ActualStepResult) {
	e.actualStepResults = append(e.actualStepResults, result)
}

func (e *Execution) RecoveryID() string {
	return e.recoveryID
}

func (e *Execution) AttachRecovery(
	id string,
) {
	e.recoveryID = id
}
