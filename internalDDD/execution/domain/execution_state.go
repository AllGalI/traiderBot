package domain

type ExecutionStatus uint8

const (
	ExecutionCreated ExecutionStatus = iota
	ExecutionExecuting
	ExecutionCompleted
	ExecutionAborted
	ExecutionFailed
)

func (s ExecutionStatus) String() string {
	switch s {
	case ExecutionCreated:
		return "created"

	case ExecutionExecuting:
		return "executing"

	case ExecutionCompleted:
		return "completed"

	case ExecutionAborted:
		return "aborted"

	case ExecutionFailed:
		return "failed"

	default:
		return "unknown"
	}
}