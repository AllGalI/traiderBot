package application

import (
	"context"

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
	marketStructure MarketStructureReader
	updates         <-chan string
	simulation      Simulator
	opportunities   OpportunityHandler

	startAmount decimal.Decimal
}

func NewWorker(
	marketStructure MarketStructureReader,
	updates <-chan string,
	simulation Simulator,
	opportunities OpportunityHandler,
	startAmount decimal.Decimal,
) *Worker {
	return &Worker{
		marketStructure: marketStructure,
		updates:         updates,
		simulation:      simulation,
		opportunities:   opportunities,
		startAmount:     startAmount,
	}
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

			if err := w.processUpdate(
				ctx,
				symbol,
			); err != nil {
				return err
			}
		}
	}
}

func (w *Worker) processUpdate(
	ctx context.Context,
	symbol string,
) error {
	if symbol == "" {
		return nil
	}

	structure, ok := w.marketStructure.Get()
	if !ok {
		return nil
	}

	routes := structure.RouteIndex.Get(symbol)

	for _, route := range routes {
		if err := ctx.Err(); err != nil {
			return err
		}

		result := w.simulation.Simulate(
			ctx,
			route,
			w.startAmount,
		)

		if result.Opportunity == nil {
			continue
		}

		w.opportunities.Handle(
			result.Opportunity,
		)
	}

	return nil
}