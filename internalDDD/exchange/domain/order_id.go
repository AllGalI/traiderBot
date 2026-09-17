package domain

import (
	"fmt"
	"sync/atomic"
)

type OrderIDGenerator struct {
	counter uint64
}

func NewOrderIDGenerator() *OrderIDGenerator {
	return &OrderIDGenerator{}
}

func (g *OrderIDGenerator) Next(stepID int) string {
	id := atomic.AddUint64(&g.counter, 1)

	return fmt.Sprintf(
		"step-%d-order-%d",
		stepID,
		id,
	)
}