package domain

import "fmt"

func (e *Execution) StartPreparing() error {
	if e.state != StateNotStarted {
		return fmt.Errorf(
			"cannot start preparing from state %s",
			e.state,
		)
	}

	e.state = StatePreparing
	return nil
}

func (e *Execution) StartExecutingStep() error {
	if e.state != StatePreparing &&
		e.state != StateEvaluating {
		return fmt.Errorf(
			"cannot start executing step from state %s",
			e.state,
		)
	}

	e.state = StateExecutingStep
	return nil
}

func (e *Execution) StartEvaluating() error {
	if e.state != StateExecutingStep &&
		e.state != StateRecovering {
		return fmt.Errorf(
			"cannot start evaluating from state %s",
			e.state,
		)
	}

	e.state = StateEvaluating
	return nil
}

func (e *Execution) ApplyDecision(
	decision ExecutionDecision,
) error {
	if e.state != StateEvaluating {
		return fmt.Errorf(
			"cannot apply decision from state %s",
			e.state,
		)
	}

	switch decision {
	case DecisionContinue:
		e.state = StateEvaluating

	case DecisionComplete:
		e.state = StateFinalizing

	case DecisionStop:
		e.state = StateFinalizing

	case DecisionRecoveryRequired:
		e.state = StateRecovering

	default:
		return fmt.Errorf(
			"unknown execution decision: %s",
			decision,
		)
	}

	return nil
}

func (e *Execution) RequireReconciliation() error {
	switch e.state {
	case StateExecutingStep,
		StateRecovering,
		StateEvaluating:

		e.state = StateReconciliationRequired
		return nil

	default:
		return fmt.Errorf(
			"cannot require reconciliation from state %s",
			e.state,
		)
	}
}

func (e *Execution) Complete() error {
	switch e.state {
	case StateFinalizing,
		StateRecovering,
		StateReconciliationRequired:

		e.state = StateCompleted
		return nil

	default:
		return fmt.Errorf(
			"cannot complete execution from state %s",
			e.state,
		)
	}
}

func (e *Execution) Fail() error {
	if e.state == StateCompleted {
		return fmt.Errorf(
			"cannot fail completed execution",
		)
	}

	if e.state == StateFailed {
		return nil
	}

	e.state = StateFailed
	return nil
}

func (e *Execution) IsFinished() bool {
	return e.state == StateCompleted ||
		e.state == StateFailed
}
