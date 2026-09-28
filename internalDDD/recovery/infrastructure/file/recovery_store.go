package file

import (
	"fmt"
	"path/filepath"

	recoveryApplication "traiderBot/internalDDD/recovery/application"
	recoveryDomain "traiderBot/internalDDD/recovery/domain"
	"traiderBot/internalDDD/infrastructure/batchjson"
	memoryStore "traiderBot/internalDDD/recovery/infrastructure/memory"
)

type RecoveryStore struct {
	memory *memoryStore.RecoveryStore
	file   *batchjson.Writer
}

func NewRecoveryStore(
	dataDir string,
	limit int,
) (*RecoveryStore, error) {

	file, err := batchjson.NewWriter(
		filepath.Join(
			dataDir,
			"recoveries",
		),
		1000,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create recovery file store: %w",
			err,
		)
	}

	return &RecoveryStore{
		memory: memoryStore.NewRecoveryStore(limit),
		file:   file,
	}, nil
}

func (s *RecoveryStore) Create(
	recovery *recoveryDomain.RecoverySnapshot,
) error {
	return s.memory.Create(recovery)
}

func (s *RecoveryStore) Update(
	recovery *recoveryDomain.RecoverySnapshot,
) error {
	return s.memory.Update(recovery)
}

func (s *RecoveryStore) Complete(
	recovery *recoveryDomain.RecoverySnapshot,
) error {

	if recovery == nil {
		return nil
	}

	if err := s.memory.Complete(recovery); err != nil {
		return err
	}

	return s.file.Append(recovery)
}

func (s *RecoveryStore) Active() []recoveryDomain.RecoverySnapshot {
	return s.memory.Active()
}

func (s *RecoveryStore) Latest() (
	*recoveryDomain.RecoverySnapshot,
	bool,
) {
	return s.memory.Latest()
}

func (s *RecoveryStore) List(
	limit int,
) []recoveryDomain.RecoverySnapshot {
	return s.memory.List(limit)
}

var _ recoveryApplication.RecoveryStore =
	(*RecoveryStore)(nil)