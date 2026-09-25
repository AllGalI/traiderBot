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

	simulationOK                    uint64
	simulationInvalidRoute          uint64
	simulationInvalidStartAmount    uint64
	simulationOrderBookNotFound     uint64
	simulationOrderBookStale        uint64
	simulationFiltersNotFound       uint64
	simulationFeeNotFound            uint64
	simulationInsufficientLiquidity uint64
	simulationProfitTooLow           uint64

	lastUpdateAt     time.Time
	lastSimulationAt time.Time

	totalSimulationTime time.Duration

	minProfitPct   decimal.Decimal
	maxProfitPct   decimal.Decimal
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

func (m *WorkerMetrics) RecordSimulationResult(
	reason simulationDomain.SimulationReason,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch reason {
	case simulationDomain.SimulationOK:
		m.simulationOK++

	case simulationDomain.SimulationInvalidRoute:
		m.simulationInvalidRoute++

	case simulationDomain.SimulationInvalidStartAmount:
		m.simulationInvalidStartAmount++

	case simulationDomain.SimulationOrderBookNotFound:
		m.simulationOrderBookNotFound++

	case simulationDomain.SimulationOrderBookStale:
		m.simulationOrderBookStale++

	case simulationDomain.SimulationFiltersNotFound:
		m.simulationFiltersNotFound++

	case simulationDomain.SimulationFeeNotFound:
		m.simulationFeeNotFound++

	case simulationDomain.SimulationInsufficientLiquidity:
		m.simulationInsufficientLiquidity++

	case simulationDomain.SimulationProfitTooLow:
		m.simulationProfitTooLow++
	}
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

	SimulationOK                    uint64
	SimulationInvalidRoute          uint64
	SimulationInvalidStartAmount    uint64
	SimulationOrderBookNotFound     uint64
	SimulationOrderBookStale        uint64
	SimulationFiltersNotFound       uint64
	SimulationFeeNotFound            uint64
	SimulationInsufficientLiquidity uint64
	SimulationProfitTooLow           uint64

	MinProfitPct   decimal.Decimal
	MaxProfitPct   decimal.Decimal
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

		SimulationOK:                    m.simulationOK,
		SimulationInvalidRoute:          m.simulationInvalidRoute,
		SimulationInvalidStartAmount:    m.simulationInvalidStartAmount,
		SimulationOrderBookNotFound:     m.simulationOrderBookNotFound,
		SimulationOrderBookStale:        m.simulationOrderBookStale,
		SimulationFiltersNotFound:       m.simulationFiltersNotFound,
		SimulationFeeNotFound:            m.simulationFeeNotFound,
		SimulationInsufficientLiquidity: m.simulationInsufficientLiquidity,
		SimulationProfitTooLow:           m.simulationProfitTooLow,

		MinProfitPct:   m.minProfitPct,
		MaxProfitPct:   m.maxProfitPct,
		HasProfitRange: m.hasProfitRange,
	}
}