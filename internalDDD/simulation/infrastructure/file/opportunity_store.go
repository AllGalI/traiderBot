package file

import (
	"fmt"
	"path/filepath"

	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
	"traiderBot/internalDDD/infrastructure/batchjson"
	memoryStore "traiderBot/internalDDD/simulation/infrastructure/memory"
)

type OpportunityStore struct {
	memory *memoryStore.OpportunityStore
	file   *batchjson.Writer
}

func NewOpportunityStore(
	dataDir string,
	limit int,
) (*OpportunityStore, error) {

	file, err := batchjson.NewWriter(
		filepath.Join(
			dataDir,
			"opportunities",
		),
		1000,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create opportunity file store: %w",
			err,
		)
	}

	return &OpportunityStore{
		memory: memoryStore.NewOpportunityStore(limit),
		file:   file,
	}, nil
}

func (s *OpportunityStore) Save(
	opportunity *simulationDomain.Opportunity,
) error {

	if opportunity == nil {
		return nil
	}

	if err := s.memory.Save(opportunity); err != nil {
		return err
	}

	return s.file.Append(opportunity)
}

func (s *OpportunityStore) Latest() (
	*simulationDomain.Opportunity,
	bool,
) {
	return s.memory.Latest()
}

func (s *OpportunityStore) List(
	limit int,
) []simulationDomain.Opportunity {
	return s.memory.List(limit)
}

func (s *OpportunityStore) Count() int {
	return s.memory.Count()
}

var _ simulationApplication.OpportunityStore =
	(*OpportunityStore)(nil)