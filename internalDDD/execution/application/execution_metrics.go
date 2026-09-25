package application

import (
	"sync"
	"time"
)


type ExecutionMetrics struct {

	mu sync.RWMutex


	started uint64

	completed uint64

	failed uint64

	recovered uint64


	totalDuration time.Duration
}


type ExecutionMetricsSnapshot struct {

	Started uint64

	Completed uint64

	Failed uint64

	Recovered uint64

	TotalDuration time.Duration
}