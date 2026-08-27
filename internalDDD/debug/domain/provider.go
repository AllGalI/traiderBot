package domain

type DomainProvider interface {
	Name() string
}

type MetricsProvider interface {
	DomainProvider
	Metrics() any
}


type SnapshotProvider interface {
	DomainProvider
	Snapshot() string
}