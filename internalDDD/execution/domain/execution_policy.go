package domain

type DecisionType uint8

const (
	DecisionContinue DecisionType = iota
	DecisionAbort
)

type ExecutionDecision struct {
	Type   DecisionType
	Reason string
}

type ExecutionPolicy interface {
	Decide(execution *Execution) ExecutionDecision
}