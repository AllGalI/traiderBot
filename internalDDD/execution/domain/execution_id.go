package domain

import (
	"fmt"
	"sync/atomic"
)

var executionIDCounter uint64

func NewExecutionID() ExecutionID {
	id := atomic.AddUint64(&executionIDCounter, 1)

	return ExecutionID(
		fmt.Sprintf("execution-%d", id),
	)
}
