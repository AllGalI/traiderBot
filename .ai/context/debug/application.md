# debug — application

Source: `internalDDD/debug/application/`

Generated: 2026-08-31 13:54:48

This file is generated automatically from the Go source code.
Do not edit manually.

## internalDDD/debug/application/debug_service.go

Package: `application`

### Interfaces

```go
Provider interface {
	Name() string
	Snapshot() (string, error)
	Metrics() (any, error)
}
```

### Structs

```go
DebugService struct {
	providers map[string]Provider
}
```

### Constructors

```go
func NewDebugService() *DebugService {
	return &DebugService{
		providers: make(map[string]Provider),
	}
}
```

### Methods

```go
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
```

```go
func (s *DebugService) Providers() []string {
	names := make([]string, 0, len(s.providers))

	for name := range s.providers {
		names = append(names, name)
	}

	return names
}
```

```go
func (s *DebugService) Register(provider Provider) {
	s.providers[provider.Name()] = provider
}
```

