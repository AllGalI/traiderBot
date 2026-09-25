package domain

import (
	"fmt"
	"sync/atomic"
)

type RecoveryID string

var recoveryIDCounter uint64

func NewRecoveryID() RecoveryID {
	id := atomic.AddUint64(&recoveryIDCounter, 1)

	return RecoveryID(
		fmt.Sprintf("recovery-%d", id),
	)
}