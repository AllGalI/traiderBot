package application

import (
	"github.com/shopspring/decimal"

	recoveryDomain "traiderBot/internalDDD/recovery/domain"
)

type Planner struct {
	routeFinder RouteFinder
	simulator   Simulator
}

func NewPlanner(
	routeFinder RouteFinder,
	simulator Simulator,
) Planner {
	return Planner{
		routeFinder: routeFinder,
		simulator:   simulator,
	}
}

func (p Planner) CreatePlan(
	input recoveryDomain.RecoveryInput,
	targetAsset string,
	ctx MarketContext,
) []recoveryDomain.RecoveryPlan {
	if targetAsset == "" {
		return nil
	}

	plans := make(
		[]recoveryDomain.RecoveryPlan,
		0,
	)

	for asset, amount := range input.ActualState {
		if amount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// Актив уже является целевым.
		if asset == targetAsset {
			continue
		}

		routes := p.routeFinder.Find(
			asset,
			targetAsset,
			3,
		)

		best, ok := p.findBest(
			routes,
			amount,
			ctx,
		)

		if !ok {
			continue
		}

		plan := recoveryDomain.RecoveryPlan{
			Asset:           asset,
			Route:           best.Route,
			InitialAmount:   amount,
			SimulatedAmount: best.OutputAmount,
			Steps:           best.Steps,
		}

		plans = append(
			plans,
			plan,
		)
	}

	return plans
}

func (p Planner) findBest(
	routes []recoveryDomain.RecoveryRoute,
	amount decimal.Decimal,
	ctx MarketContext,
) (
	recoveryDomain.SimulationResult,
	bool,
) {
	var best recoveryDomain.SimulationResult

	found := false

	for _, route := range routes {
		result := p.simulator.Simulate(
			route,
			amount,
			ctx,
		)

		if result.OutputAmount.LessThanOrEqual(
			decimal.Zero,
		) {
			continue
		}

		if !found ||
			result.IsBetterThan(best) {

			best = result
			found = true
		}
	}

	return best, found
}