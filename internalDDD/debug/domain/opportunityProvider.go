package domain

import (
	"fmt"
	"strings"

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


func (p *OpportunityProvider) Snapshot() (string, error) {

	opportunity, ok := p.store.Latest()

	if !ok {
		return "No opportunities\n", nil
	}


	var builder strings.Builder


	fmt.Fprintln(
		&builder,
		"Latest Opportunity",
	)

	fmt.Fprintln(
		&builder,
		"------------------",
	)


	fmt.Fprintf(
		&builder,
		"ID: %s\n",
		opportunity.ID,
	)


	fmt.Fprintf(
		&builder,
		"Route: %v\n",
		opportunity.Route.Symbols(),
	)


	fmt.Fprintf(
		&builder,
		"Start asset: %s\n",
		opportunity.StartAsset,
	)


	fmt.Fprintf(
		&builder,
		"Requested: %s\n",
		opportunity.RequestedStartAmount,
	)


	fmt.Fprintf(
		&builder,
		"Executed: %s\n",
		opportunity.ExecutedStartAmount,
	)


	fmt.Fprintf(
		&builder,
		"End amount: %s\n",
		opportunity.EndAmount,
	)


	fmt.Fprintf(
		&builder,
		"Profit: %s (%s%%)\n",
		opportunity.ProfitAbs,
		opportunity.ProfitPct.Mul(decimal.NewFromInt(100)),
	)


	fmt.Fprintf(
		&builder,
		"Fill ratio: %s%%\n",
		opportunity.FillRatio.Mul(decimal.NewFromInt(100)),
	)


	fmt.Fprintf(
		&builder,
		"Created: %s\n",
		opportunity.Timestamp,
	)


	return builder.String(), nil
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


	response["averageProfitPct"] =
		total.Div(
			decimal.NewFromInt(
				int64(len(opportunities)),
			),
		)


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