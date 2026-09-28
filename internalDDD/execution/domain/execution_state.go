package domain

type ExecutionState string

const (
	StateNotStarted             ExecutionState = "NotStarted"
	StatePreparing              ExecutionState = "Preparing"
	StateExecutingStep          ExecutionState = "ExecutingStep"
	StateEvaluating             ExecutionState = "Evaluating"
	StateRecovering             ExecutionState = "Recovering"
	StateReconciliationRequired ExecutionState = "ReconciliationRequired"
	StateFinalizing             ExecutionState = "Finalizing"

	StateCompleted ExecutionState = "Completed"
	StateFailed    ExecutionState = "Failed"
)

type ExecutionDecision string

const (
	DecisionContinue         ExecutionDecision = "Continue"
	DecisionComplete         ExecutionDecision = "Complete"
	DecisionStop             ExecutionDecision = "Stop"
	DecisionRecoveryRequired ExecutionDecision = "RecoveryRequired"
)