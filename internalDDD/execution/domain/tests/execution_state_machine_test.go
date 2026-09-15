package tests

import (
	"testing"
	"traiderBot/internalDDD/execution/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestNewExecutionStartsNotStarted(t *testing.T) {
	execution := domain.NewExecution(simulationDomain.Opportunity{})
	if execution.State() != domain.StateNotStarted {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateNotStarted,
			execution.State(),
		)
	}
}

func TestNotStartedToPreparing(t *testing.T) {
	execution := domain.NewExecution(simulationDomain.Opportunity{})
	if err := execution.StartPreparing(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StatePreparing {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StatePreparing,
			execution.State(),
		)
	}
}

func TestPreparingToExecutingStep(t *testing.T) {
	execution := domain.NewExecution(simulationDomain.Opportunity{})
	_ = execution.StartPreparing()

	if err := execution.StartExecutingStep(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateExecutingStep {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateExecutingStep,
			execution.State(),
		)
	}
}

func TestExecutingStepToEvaluating(t *testing.T) {
	execution := domain.NewExecution(simulationDomain.Opportunity{})
	_ = execution.StartPreparing()
	_ = execution.StartExecutingStep()

	if err := execution.StartEvaluating(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateEvaluating {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateEvaluating,
			execution.State(),
		)
	}
}

func TestEvaluatingContinueStartsNextStep(t *testing.T) {
	execution := executionInEvaluating(t)

	if err := execution.ApplyDecision(domain.DecisionContinue); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateExecutingStep {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateExecutingStep,
			execution.State(),
		)
	}
}

func TestEvaluatingCompleteStartsFinalizing(t *testing.T) {
	execution := executionInEvaluating(t)

	if err := execution.ApplyDecision(domain.DecisionComplete); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateFinalizing {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateFinalizing,
			execution.State(),
		)
	}
}

func TestEvaluatingStopStartsFinalizing(t *testing.T) {
	execution := executionInEvaluating(t)

	if err := execution.ApplyDecision(domain.DecisionStop); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateFinalizing {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateFinalizing,
			execution.State(),
		)
	}
}

func TestEvaluatingRecoveryStartsRecovering(t *testing.T) {
	execution := executionInEvaluating(t)

	if err := execution.ApplyDecision(domain.DecisionRecoveryRequired); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateRecovering {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateRecovering,
			execution.State(),
		)
	}
}

func TestRecoveringToEvaluating(t *testing.T) {
	execution := executionInEvaluating(t)

	_ = execution.ApplyDecision(domain.DecisionRecoveryRequired)

	if err := execution.StartEvaluating(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateEvaluating {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateEvaluating,
			execution.State(),
		)
	}
}

func TestUnknownFactsRequireReconciliation(t *testing.T) {
	execution := executionInExecutingStep(t)

	if err := execution.RequireReconciliation(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.State() != domain.StateReconciliationRequired {
		t.Fatalf(
			"expected state %s, got %s",
			domain.StateReconciliationRequired,
			execution.State(),
		)
	}
}

func TestInvalidTransitionIsRejected(t *testing.T) {
	execution := domain.NewExecution(simulationDomain.Opportunity{})
	err := execution.StartEvaluating()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if execution.State() != domain.StateNotStarted {
		t.Fatalf(
			"expected state to remain %s, got %s",
			domain.StateNotStarted,
			execution.State(),
		)
	}
}

func executionInExecutingStep(t *testing.T) *domain.Execution {
	t.Helper()

	execution := domain.NewExecution(simulationDomain.Opportunity{})
	if err := execution.StartPreparing(); err != nil {
		t.Fatal(err)
	}

	if err := execution.StartExecutingStep(); err != nil {
		t.Fatal(err)
	}

	return execution
}

func executionInEvaluating(t *testing.T) *domain.Execution {
	t.Helper()

	execution := executionInExecutingStep(t)

	if err := execution.StartEvaluating(); err != nil {
		t.Fatal(err)
	}

	return execution
}
