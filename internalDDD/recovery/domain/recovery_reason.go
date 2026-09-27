package domain

type RecoveryReason string

const (
	RecoveryReasonUnknown RecoveryReason = "Unknown"

	RecoveryReasonPartialFill RecoveryReason = "PartialFill"

	RecoveryReasonRejected RecoveryReason = "Rejected"

	RecoveryReasonCancelled RecoveryReason = "Cancelled"

	RecoveryReasonDeviation RecoveryReason = "Deviation"
)