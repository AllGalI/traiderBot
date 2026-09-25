package application

import (
	"fmt"
	"sync/atomic"
	"traiderBot/internalDDD/simulation/domain"
)

type OpportunityIDGenerator struct {
	counter uint64
}

func NewOpportunityIDGenerator() *OpportunityIDGenerator {
	return &OpportunityIDGenerator{}
}

func (g *OpportunityIDGenerator) Next()  domain.OpportunityID {
	id := atomic.AddUint64(&g.counter, 1)

	return domain.OpportunityID(fmt.Sprintf("opportunity-%d", id))
}