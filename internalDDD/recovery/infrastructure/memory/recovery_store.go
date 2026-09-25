package memory

import (
	"sync"

	recoveryDomain "traiderBot/internalDDD/recovery/domain"
)

type RecoveryStore struct {
	mu sync.RWMutex

	active map[string]recoveryDomain.RecoverySnapshot

	history []recoveryDomain.RecoverySnapshot

	limit int
}

func NewRecoveryStore(
	limit int,
) *RecoveryStore {

	if limit <= 0 {
		limit = 1000
	}

	return &RecoveryStore{
		active: make(
			map[string]recoveryDomain.RecoverySnapshot,
		),

		history: make(
			[]recoveryDomain.RecoverySnapshot,
			0,
			limit,
		),

		limit: limit,
	}
}


func (s *RecoveryStore) Create(
	recovery *recoveryDomain.RecoverySnapshot,
) error {

	if recovery == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.active[string(recovery.ID)] = *recovery

	return nil
}


func (s *RecoveryStore) Update(
	recovery *recoveryDomain.RecoverySnapshot,
) error {

	if recovery == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.active[string(recovery.ID)] = *recovery

	return nil
}


func (s *RecoveryStore) Complete(
	recovery *recoveryDomain.RecoverySnapshot,
) error {

	if recovery == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()


	delete(
		s.active,
		string(recovery.ID),
	)


	s.history = append(
		s.history,
		*recovery,
	)


	if len(s.history) > s.limit {

		remove := len(s.history) - s.limit

		s.history = s.history[remove:]
	}


	return nil
}


func (s *RecoveryStore) Active() []recoveryDomain.RecoverySnapshot {

	s.mu.RLock()
	defer s.mu.RUnlock()


	result := make(
		[]recoveryDomain.RecoverySnapshot,
		0,
		len(s.active),
	)


	for _, recovery := range s.active {
		result = append(
			result,
			recovery,
		)
	}


	return result
}


func (s *RecoveryStore) Latest() (
	*recoveryDomain.RecoverySnapshot,
	bool,
) {

	s.mu.RLock()
	defer s.mu.RUnlock()


	if len(s.history) == 0 {
		return nil, false
	}


	latest := s.history[len(s.history)-1]


	return &latest, true
}


func (s *RecoveryStore) List(
	limit int,
) []recoveryDomain.RecoverySnapshot {

	s.mu.RLock()
	defer s.mu.RUnlock()


	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}


	start := len(s.history) - limit


	result := make(
		[]recoveryDomain.RecoverySnapshot,
		limit,
	)


	copy(
		result,
		s.history[start:],
	)


	return result
}