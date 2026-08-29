package application

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

type OrderBookReader interface {
	Get(symbol string) (marketDomain.OrderBook, bool)
}

type SymbolProvider interface {
	Get(symbol string) (marketDomain.Symbol, bool)
}

type SimulationService struct {
	orderBooks            OrderBookReader
	fees                  FeeProvider
	symbols               SymbolProvider
	minLiquidityFillRatio decimal.Decimal
	minProfitPct          decimal.Decimal
}

func NewSimulationService(
	orderBooks OrderBookReader,
	fees FeeProvider,
	symbols SymbolProvider,
	minLiquidityFillRatio decimal.Decimal,
	minProfitPct decimal.Decimal,
) *SimulationService {
	return &SimulationService{
		orderBooks:            orderBooks,
		fees:                  fees,
		symbols:               symbols,
		minLiquidityFillRatio: minLiquidityFillRatio,
		minProfitPct:          minProfitPct,
	}
}

func (s *SimulationService) Simulate(
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

	for _, edge := range route.Edges {
		orderBook, ok := s.orderBooks.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationOrderBookNotFound,
			}
		}

		symbol, ok := s.symbols.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationFiltersNotFound,
			}
		}

		fee, ok := s.fees.Get(edge.Symbol)

		if !ok {
			return simulationDomain.SimulationResult{
				Reason: simulationDomain.SimulationFeeNotFound,
			}
		}

		step, err := simulationDomain.SimulateStep(
			edge,
			orderBook,
			symbol.Constraints,
			currentAmount,
			s.minLiquidityFillRatio,
			fee.Rate,
			fee.Asset,
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

		// Коэффициент заполнения маршрута —
		// произведение коэффициентов заполнения
		// всех последовательных шагов.
		//
		// Например:
		//
		// 1.00 × 0.75 × 0.80 × 1.00 = 0.60
		routeFillRatio = routeFillRatio.Mul(step.FillRatio)

		// Следующий шаг работает с фактически
		// полученной валютой предыдущего шага.
		//
		// Здесь уже учитывается комиссия,
		// потому что AmountOut содержит
		// net amount после комиссии.
		currentAmount = step.AmountOut
	}

	// Реально исполненный стартовый объём.
	//
	// Например:
	//
	// requested = 100 USDT
	// routeFill = 0.60
	//
	// executed = 60 USDT
	executedStartAmount := startAmount.Mul(routeFillRatio)

	profitAbs := currentAmount.Sub(executedStartAmount)

	profitPct := decimal.Zero

	if !executedStartAmount.IsZero() {
		profitPct = profitAbs.Div(executedStartAmount)
	}

	// Opportunity создаём только для маршрута,
	// который прошёл минимальный порог прибыльности.
	if profitPct.LessThan(s.minProfitPct) {
		return simulationDomain.SimulationResult{
			Reason:         simulationDomain.SimulationProfitTooLow,
			RouteFillRatio: routeFillRatio,
		}
	}

	return simulationDomain.SimulationResult{
		Reason:         simulationDomain.SimulationOK,
		RouteFillRatio: routeFillRatio,

		Opportunity: &simulationDomain.Opportunity{
			Route: route,
			Steps: steps,

			StartAsset: startAsset(route),

			RequestedStartAmount: startAmount,
			ExecutedStartAmount:  executedStartAmount,
			EndAmount:            currentAmount,

			FillRatio: routeFillRatio,

			ProfitAbs: profitAbs,
			ProfitPct: profitPct,

			Timestamp: time.Now(),
		},
	}
}

func startAsset(route marketDomain.Route) string {
	if len(route.Edges) == 0 {
		return ""
	}

	return route.Edges[0].From
}
