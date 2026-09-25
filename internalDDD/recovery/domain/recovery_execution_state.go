package domain

type RecoveryProcessState string

const (
	RecoveryStarted    RecoveryProcessState = "Started"
	RecoveryPlanning   RecoveryProcessState = "Planning"
	RecoveryExecuting  RecoveryProcessState = "Executing"
	RecoveryCompleted  RecoveryProcessState = "Completed"
	RecoveryFailed     RecoveryProcessState = "Failed"
)