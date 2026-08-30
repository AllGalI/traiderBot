package domain

type ExecutionAction uint8

const (
	ActionContinue ExecutionAction = iota
	ActionAbort
	ActionCancel
)

func (a ExecutionAction) String() string {
	switch a {
	case ActionContinue:
		return "continue"

	case ActionAbort:
		return "abort"

	case ActionCancel:
		return "cancel"

	default:
		return "unknown"
	}
}