package application

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"

	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type WorkerMetrics struct {
	mu sync.RWMutex

	updatesReceived uint64
	routesSimulated uint64
	opportunities   uint64

	lastUpdateAt     time.Time
	lastSimulationAt time.Time

	totalSimulationTime time.Duration

	minProfitPct decimal.Decimal
	maxProfitPct decimal.Decimal
	hasProfitRange bool
}

func NewWorkerMetrics() *WorkerMetrics {
	return &WorkerMetrics{}
}

func (m *WorkerMetrics) RecordUpdate() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updatesReceived++
	m.lastUpdateAt = time.Now()
}

func (m *WorkerMetrics) RecordSimulation(
	duration time.Duration,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.routesSimulated++
	m.lastSimulationAt = time.Now()
	m.totalSimulationTime += duration
}

func (m *WorkerMetrics) RecordOpportunity(
	opportunity *simulationDomain.Opportunity,
) {
	if opportunity == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.opportunities++

	profitPct := opportunity.ProfitPct

	if !m.hasProfitRange {
		m.minProfitPct = profitPct
		m.maxProfitPct = profitPct
		m.hasProfitRange = true

		return
	}

	if profitPct.LessThan(m.minProfitPct) {
		m.minProfitPct = profitPct
	}

	if profitPct.GreaterThan(m.maxProfitPct) {
		m.maxProfitPct = profitPct
	}
}

type WorkerMetricsSnapshot struct {
	UpdatesReceived     uint64
	RoutesSimulated     uint64
	Opportunities       uint64
	TotalSimulationTime time.Duration
	LastUpdateAt        time.Time
	LastSimulationAt    time.Time

	MinProfitPct decimal.Decimal
	MaxProfitPct decimal.Decimal
	HasProfitRange bool
}

func (m *WorkerMetrics) Snapshot() WorkerMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return WorkerMetricsSnapshot{
		UpdatesReceived:     m.updatesReceived,
		RoutesSimulated:     m.routesSimulated,
		Opportunities:       m.opportunities,
		TotalSimulationTime: m.totalSimulationTime,
		LastUpdateAt:        m.lastUpdateAt,
		LastSimulationAt:    m.lastSimulationAt,

		MinProfitPct:   m.minProfitPct,
		MaxProfitPct:   m.maxProfitPct,
		HasProfitRange: m.hasProfitRange,
	}
}