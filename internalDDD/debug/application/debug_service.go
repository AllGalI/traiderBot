package application

import (
	"fmt"
)

type Provider interface {
	Name() string
	Snapshot() (string, error)
	Metrics() (any, error)
}

type DebugService struct {
	providers map[string]Provider
}

func NewDebugService() *DebugService {
	return &DebugService{
		providers: make(map[string]Provider),
	}
}

func (s *DebugService) Register(provider Provider) {
	s.providers[provider.Name()] = provider
}

func (s *DebugService) Providers() []string {
	names := make([]string, 0, len(s.providers))

	for name := range s.providers {
		names = append(names, name)
	}

	return names
}

func (s *DebugService) Provider(name string) (Provider, error) {
	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf(
			"provider %q not found",
			name,
		)
	}

	return provider, nil
}