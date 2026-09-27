package domain

import (
	"fmt"
	"sync/atomic"
)


var recoveryIDCounter uint64

func NewRecoveryID() string {
	id := atomic.AddUint64(&recoveryIDCounter, 1)

	return fmt.Sprintf("recovery-%d", id)
	
}