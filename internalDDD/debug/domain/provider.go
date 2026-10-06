package domain

import (
	"context"
	"io"
)

type Provider interface {
	Name() string
	Snapshot() (any, error)
	Metrics() (any, error)
}

type BrowserProvider interface {
	Provider

	Browse(
		ctx context.Context,
		reader io.Reader,
		writer io.Writer,
	) error
}
