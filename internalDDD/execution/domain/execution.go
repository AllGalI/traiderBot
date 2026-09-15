package domain

import (
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type Execution struct {
	state             ExecutionState
	opportunity       simulationDomain.Opportunity
	actualStepResults []ActualStepResult
}

func NewExecution(opportunity simulationDomain.Opportunity) *Execution {
	return &Execution{
		state:             StateNotStarted,
		opportunity:       opportunity,
		actualStepResults: make([]ActualStepResult, 0),
	}
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