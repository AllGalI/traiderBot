package file

import (
	"fmt"
	"path/filepath"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	"traiderBot/internalDDD/infrastructure/batchjson"
	memoryStore "traiderBot/internalDDD/execution/infrastructure/memory"
)

type ExecutionStore struct {
	memory *memoryStore.ExecutionStore
	file   *batchjson.Writer
}

func NewExecutionStore(
	dataDir string,
	limit int,
) (*ExecutionStore, error) {

	file, err := batchjson.NewWriter(
		filepath.Join(
			dataDir,
			"executions",
		),
		1000,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create execution file store: %w",
			err,
		)
	}

	return &ExecutionStore{
		memory: memoryStore.NewExecutionStore(limit),
		file:   file,
	}, nil
}

func (s *ExecutionStore) Create(
	execution *executionDomain.ExecutionSnapshot,
) error {
	return s.memory.Create(execution)
}

func (s *ExecutionStore) Update(
	execution *executionDomain.ExecutionSnapshot,
) error {
	return s.memory.Update(execution)
}

func (s *ExecutionStore) Complete(
	execution *executionDomain.ExecutionSnapshot,
) error {

	if execution == nil {
		return nil
	}

	if err := s.memory.Complete(execution); err != nil {
		return err
	}

	return s.file.Append(execution)
}

func (s *ExecutionStore) Active() []executionDomain.ExecutionSnapshot {
	return s.memory.Active()
}

func (s *ExecutionStore) Latest() (
	*executionDomain.ExecutionSnapshot,
	bool,
) {
	return s.memory.Latest()
}

func (s *ExecutionStore) List(
	limit int,
) []executionDomain.ExecutionSnapshot {
	return s.memory.List(limit)
}

var _ executionApplication.ExecutionStore =
	(*ExecutionStore)(nil)