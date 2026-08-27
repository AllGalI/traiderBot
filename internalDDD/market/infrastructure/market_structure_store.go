package infrastructure

import (
	"sync"

	"traiderBot/internalDDD/market/domain"
)

type InMemoryMarketStructureStore struct {
	mu        sync.RWMutex
	structure domain.MarketStructure
	exists    bool
}

func NewInMemoryMarketStructureStore() *InMemoryMarketStructureStore {
	return &InMemoryMarketStructureStore{}
}

func (s *InMemoryMarketStructureStore) Get() (*domain.MarketStructure, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.exists {
		return &domain.MarketStructure{}, false
	}

	return &s.structure, true
}

func (s *InMemoryMarketStructureStore) Set(
	structure *domain.MarketStructure,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.structure = *structure
	s.exists = true
}
