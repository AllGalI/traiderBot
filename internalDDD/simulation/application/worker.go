package application

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type Simulator interface {
	Simulate(
		ctx context.Context,
		route marketDomain.Route,
		startAmount decimal.Decimal,
	) simulationDomain.SimulationResult
}

type OpportunityHandler interface {
	Handle(*simulationDomain.Opportunity)
}

type Worker struct {
	marketStructure  MarketStructureReader
	updates          <-chan string
	simulation       Simulator
	opportunities    OpportunityHandler
	opportunityStore OpportunityStore

	metrics *WorkerMetrics

	startAmount decimal.Decimal
}

func NewWorker(
	marketStructure MarketStructureReader,
	updates <-chan string,
	simulation Simulator,
	opportunities OpportunityHandler,
	opportunityStore OpportunityStore,
	startAmount decimal.Decimal,
) *Worker {
	return &Worker{
		marketStructure:  marketStructure,
		updates:          updates,
		simulation:       simulation,
		opportunities:    opportunities,
		opportunityStore: opportunityStore,
		metrics:          NewWorkerMetrics(),
		startAmount:      startAmount,
	}
}

func (w *Worker) Metrics() *WorkerMetrics {
	return w.metrics
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case symbol, ok := <-w.updates:
			if !ok {
				return nil
			}

			if err := w.runUpdate(ctx, symbol); err != nil {
				return err
			}
		}
	}
}

func (w *Worker) runUpdate(
	ctx context.Context,
	symbol string,
) error {
	w.metrics.RecordUpdate()

	structure, ok := w.marketStructure.Get()
	if !ok {
		return nil
	}

	routes := structure.RouteIndex.Get(symbol)

	for _, route := range routes {
		if err := ctx.Err(); err != nil {
			return err
		}

		started := time.Now()

		result := w.simulation.Simulate(
			ctx,
			route,
			w.startAmount,
		)

		w.metrics.RecordSimulation(
			time.Since(started),
		)

		w.metrics.RecordSimulationResult(
			result.Reason,
		)

		if result.Opportunity == nil {
			continue
		}

		if w.opportunityStore != nil {
			_ = w.opportunityStore.Save(
				result.Opportunity,
			)
		}

		w.metrics.RecordOpportunity(
			result.Opportunity,
		)

		w.opportunities.Handle(
			result.Opportunity,
		)
	}

	return nil
}
