package application

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type OrderBookFreshness interface {
	IsFresh(symbol string, updatedAt time.Time, now time.Time) bool
}

type OrderBookReader interface {
	Get(symbol string) (marketDomain.OrderBook, bool)
}

type SymbolProvider interface {
	Get(symbol string) (marketDomain.Symbol, bool)
}

type SimulationService struct {
	orderBooks            OrderBookReader
	feeProvider           FeeProvider
	symbols               SymbolProvider
	freshness             OrderBookFreshness
	minLiquidityFillRatio decimal.Decimal
	minProfitPct          decimal.Decimal
}

func NewSimulationService(
	orderBooks OrderBookReader,
	feeProvider FeeProvider,
	symbols SymbolProvider,
	freshness OrderBookFreshness,
	minLiquidityFillRatio decimal.Decimal,
	minProfitPct decimal.Decimal,
) *SimulationService {
	return &SimulationService{
		orderBooks:            orderBooks,
		feeProvider:           feeProvider,
		symbols:               symbols,
		freshness:             freshness,
		minLiquidityFillRatio: minLiquidityFillRatio,
		minProfitPct:          minProfitPct,
	}
}

func (s *SimulationService) Simulate(
	ctx context.Context,
	route marketDomain.Route,
	startAmount decimal.Decimal,
) simulationDomain.SimulationResult {
	if !route.IsTriangle() {
		return simulationDomain.SimulationResult{
			Reason: simulationDomain.SimulationInvalidRoute,
		}
	}

	if startAmount.LessThanOrEqual(decimal.Zero) {
		return simulationDomain.SimulationResult{
			Reason: simulationDomain.SimulationInvalidStartAmount,
		}
	}

	currentAmount := startAmount
	routeFillRatio := decimal.NewFromInt(1)

	steps := make(
		[]simulationDomain.StepResult,
		0,
		len(route.Edges),
	)

	if err := ctx.Err(); err != nil {
		return simulationDomain.SimulationResult{
			Reason: simulationDomain.SimulationInvalidStartAmount,
		}
	}

	for _, edge := range route.Edges {
		orderBook, ok := s.orderBooks.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationOrderBookNotFound,
			}
		}

		// if s.freshness != nil &&
		// 	!s.freshness.IsFresh(
		// 		edge.Symbol,
		// 		orderBook.LastUpdateTime,
		// 		time.Now(),
		// 	) {
		// 	return simulationDomain.SimulationResult{
		// 		Reason: simulationDomain.SimulationOrderBookStale,
		// 	}
		// }

		symbol, ok := s.symbols.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationFiltersNotFound,
			}
		}

		fee, ok := s.feeProvider.Get(ctx, edge.Symbol)
		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationFeeNotFound,
			}
		}

		// if symbol.Name == "BTCUSDT" {
		// 	fmt.Printf(
		// 		"[SIM] %s updated=%s ask0=%s@%s bid0=%s@%s amount=%s\n",
		// 		edge.Symbol,
		// 		orderBook.LastUpdateTime,
		// 		orderBook.Asks[0].Qty,
		// 		orderBook.Asks[0].Price,
		// 		orderBook.Bids[0].Qty,
		// 		orderBook.Bids[0].Price,
		// 		currentAmount,
		// 	)

		// }

		step, err := simulationDomain.SimulateStep(
			edge,
			orderBook,
			symbol.Constraints,
			currentAmount,
			s.minLiquidityFillRatio,
			fee.Rate,
			edge.To,
		)

		if err != nil {
			fmt.Printf(
				"DEBUG SimulateStep: symbol=%s from=%s to=%s direction=%d amountIn=%s error=%v\n",
				edge.Symbol,
				edge.From,
				edge.To,
				edge.Direction,
				currentAmount,
				err,
			)

			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationInsufficientLiquidity,
			}
		}

		steps = append(steps, step)

		routeFillRatio = routeFillRatio.Mul(step.FillRatio)

		currentAmount = step.NetOutput
	}

	executedStartAmount := startAmount.Mul(routeFillRatio)

	profitAbs := currentAmount.Sub(executedStartAmount)

	profitPct := decimal.Zero

	if !executedStartAmount.IsZero() {
		profitPct = profitAbs.Div(executedStartAmount)
	}

	if profitPct.LessThan(s.minProfitPct) {
		return simulationDomain.SimulationResult{
			Reason:         simulationDomain.SimulationProfitTooLow,
			RouteFillRatio: routeFillRatio,
		}
	}

	opportunity := &simulationDomain.Opportunity{
		ID: simulationDomain.NewOpportunityID(),

		Route:      route,
		Steps:      steps,
		StartAsset: startAsset(route),

		RequestedStartAmount: startAmount,
		ExecutedStartAmount:  executedStartAmount,
		EndAmount:            currentAmount,

		FillRatio: routeFillRatio,

		ProfitAbs: profitAbs,
		ProfitPct: profitPct,

		Timestamp: time.Now(),
	}

	// fmt.Printf(
	// 	"INFO opportunity_created opportunity_id=%s start_asset=%s requested_amount=%s executed_amount=%s end_amount=%s profit_abs=%s profit_pct=%s\n",
	// 	opportunity.ID,
	// 	opportunity.StartAsset,
	// 	opportunity.RequestedStartAmount,
	// 	opportunity.ExecutedStartAmount,
	// 	opportunity.EndAmount,
	// 	opportunity.ProfitAbs,
	// 	opportunity.ProfitPct,
	// )

	return simulationDomain.SimulationResult{
		Reason:         simulationDomain.SimulationOK,
		RouteFillRatio: routeFillRatio,
		Opportunity:    opportunity,
	}
}

func startAsset(route marketDomain.Route) string {
	if len(route.Edges) == 0 {
		return ""
	}

	return route.Edges[0].From
}
