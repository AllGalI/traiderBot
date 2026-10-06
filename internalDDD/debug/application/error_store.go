package application

import (
	"fmt"
	"sort"

	debugDomain "traiderBot/internalDDD/debug/domain"
)

type DebugService struct {
	providers  map[string]debugDomain.Provider
	errorStore *ErrorStore
}

func NewDebugService() *DebugService {
	return &DebugService{
		providers:  make(map[string]debugDomain.Provider),
		errorStore: NewErrorStore(500),
	}
}

func (s *DebugService) Register(provider debugDomain.Provider) {
	s.providers[provider.Name()] = provider
}

func (s *DebugService) Providers() []string {
	names := make([]string, 0, len(s.providers))

	for name := range s.providers {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (s *DebugService) Provider(
	name string,
) (debugDomain.Provider, error) {
	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf(
			"provider %q not found",
			name,
		)
	}

	return provider, nil
}

func (s *DebugService) Errors() *ErrorStore {
	return s.errorStore
}

func (s *DebugService) Error(
	component string,
	message string,
	details map[string]string,
) {
	s.errorStore.Error(component, message, details)
}

func (s *DebugService) Warn(
	component string,
	message string,
	details map[string]string,
) {
	s.errorStore.Warn(component, message, details)
}