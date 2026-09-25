package memory

import (
	"sync"

	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type OpportunityStore struct {
	mu sync.RWMutex

	limit int

	opportunities []simulationDomain.Opportunity
}

func NewOpportunityStore(
	limit int,
) *OpportunityStore {

	if limit <= 0 {
		limit = 1000
	}

	return &OpportunityStore{
		limit: limit,
		opportunities: make(
			[]simulationDomain.Opportunity,
			0,
			limit,
		),
	}
}

func (s *OpportunityStore) Save(
	opportunity *simulationDomain.Opportunity,
) error {

	if opportunity == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.opportunities = append(
		s.opportunities,
		*opportunity,
	)

	if len(s.opportunities) > s.limit {

		remove := len(s.opportunities) - s.limit

		s.opportunities = s.opportunities[remove:]
	}

	return nil
}

func (s *OpportunityStore) Latest() (
	*simulationDomain.Opportunity,
	bool,
) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.opportunities) == 0 {
		return nil, false
	}

	latest := s.opportunities[len(s.opportunities)-1]

	return &latest, true
}

func (s *OpportunityStore) List(
	limit int,
) []simulationDomain.Opportunity {

	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.opportunities) {
		limit = len(s.opportunities)
	}

	start := len(s.opportunities) - limit

	result := make(
		[]simulationDomain.Opportunity,
		limit,
	)

	copy(
		result,
		s.opportunities[start:],
	)

	return result
}

func (s *OpportunityStore) Count() int {

	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.opportunities)
}