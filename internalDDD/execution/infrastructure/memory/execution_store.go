package memory

import (
	"sync"

	"traiderBot/internalDDD/execution/domain"
)

type ExecutionStore struct {
	mu sync.RWMutex

	active map[string]domain.ExecutionSnapshot

	history []domain.ExecutionSnapshot

	limit int
}

func NewExecutionStore(
	limit int,
) *ExecutionStore {

	if limit <= 0 {
		limit = 1000
	}

	return &ExecutionStore{
		active: make(
			map[string]domain.ExecutionSnapshot,
		),

		history: make(
			[]domain.ExecutionSnapshot,
			0,
			limit,
		),

		limit: limit,
	}
}

func (s *ExecutionStore) Create(
	execution *domain.ExecutionSnapshot,
) error {

	if execution == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.active[string(execution.ID)] = *execution

	return nil
}

func (s *ExecutionStore) Update(
	execution *domain.ExecutionSnapshot,
) error {

	if execution == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := *execution

	if previous, exists := s.active[string(execution.ID)]; exists {
		if current.StartedAt.IsZero() {
			current.StartedAt = previous.StartedAt
		}
	}

	s.active[string(execution.ID)] = current

	if current.CompletedAt != nil {

		delete(
			s.active,
			string(execution.ID),
		)

		s.history = append(
			s.history,
			current,
		)

		if len(s.history) > s.limit {

			remove :=
				len(s.history) - s.limit

			s.history =
				s.history[remove:]
		}
	}

	return nil
}

func (s *ExecutionStore) Active() []domain.ExecutionSnapshot {

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(
		[]domain.ExecutionSnapshot,
		0,
		len(s.active),
	)

	for _, execution := range s.active {
		result = append(
			result,
			execution,
		)
	}

	return result
}

func (s *ExecutionStore) Latest() (
	*domain.ExecutionSnapshot,
	bool,
) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.history) == 0 {
		return nil, false
	}

	latest :=
		s.history[len(s.history)-1]

	return &latest, true
}

func (s *ExecutionStore) List(
	limit int,
) []domain.ExecutionSnapshot {

	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 ||
		limit > len(s.history) {
		limit = len(s.history)
	}

	start :=
		len(s.history) - limit

	result := make(
		[]domain.ExecutionSnapshot,
		limit,
	)

	copy(
		result,
		s.history[start:],
	)

	return result
}

func (s *ExecutionStore) Complete(
	execution *domain.ExecutionSnapshot,
) error {

	if execution == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(
		s.active,
		string(execution.ID),
	)

	s.history = append(
		s.history,
		*execution,
	)

	if len(s.history) > s.limit {

		remove := len(s.history) - s.limit

		s.history = s.history[remove:]
	}

	return nil
}
