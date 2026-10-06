package domain

import (
	"time"

	"github.com/shopspring/decimal"

	simulationApplication "traiderBot/internalDDD/simulation/application"
)

type OpportunityProvider struct {
	store simulationApplication.OpportunityStore
}

func NewOpportunityProvider(
	store simulationApplication.OpportunityStore,
) *OpportunityProvider {
	return &OpportunityProvider{
		store: store,
	}
}

func (p *OpportunityProvider) Name() string {
	return "OpportunityProvider"
}

func (p *OpportunityProvider) Snapshot() (any, error) {
	opportunity, ok := p.store.Latest()

	if !ok {
		return map[string]any{
			"status": "empty",
		}, nil
	}

	return map[string]any{
		"id":                   opportunity.ID,
		"route":                opportunity.Route.Symbols(),
		"startAsset":           opportunity.StartAsset,
		"requestedStartAmount": opportunity.RequestedStartAmount,
		"executedStartAmount":  opportunity.ExecutedStartAmount,
		"endAmount":            opportunity.EndAmount,
		"profitAbs":            opportunity.ProfitAbs,
		"profitPct":            opportunity.ProfitPct.Mul(decimal.NewFromInt(100)),
		"fillRatio":            opportunity.FillRatio.Mul(decimal.NewFromInt(100)),
		"timestamp":             opportunity.Timestamp.Format(time.RFC3339Nano),
	}, nil
}

func (p *OpportunityProvider) Metrics() (any, error) {
	opportunities := p.store.List(1000)

	response := map[string]any{
		"stored": len(opportunities),
	}

	if len(opportunities) == 0 {
		return response, nil
	}

	var total decimal.Decimal
	var best decimal.Decimal
	var worst decimal.Decimal

	var positive int
	var negative int

	for index, opportunity := range opportunities {
		profit := opportunity.ProfitPct

		total = total.Add(profit)

		if index == 0 {
			best = profit
			worst = profit
		}

		if profit.GreaterThan(best) {
			best = profit
		}

		if profit.LessThan(worst) {
			worst = profit
		}

		if profit.GreaterThanOrEqual(decimal.Zero) {
			positive++
		} else {
			negative++
		}
	}

	count := decimal.NewFromInt(int64(len(opportunities)))

	response["averageProfitPct"] = total.Div(count)
	response["bestProfitPct"] = best
	response["worstProfitPct"] = worst
	response["positive"] = positive
	response["negative"] = negative

	latest, ok := p.store.Latest()

	if ok {
		response["latestID"] = latest.ID
		response["latestProfitPct"] = latest.ProfitPct
	}

	return response, nil
}
