package domain

import (
	"fmt"
	"sync/atomic"
)

var opportunityIDCounter uint64

func NewOpportunityID() OpportunityID {
	id := atomic.AddUint64(&opportunityIDCounter, 1)

	return OpportunityID(
		fmt.Sprintf("opportunity-%d", id),
	)
}
