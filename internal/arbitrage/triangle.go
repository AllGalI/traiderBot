package arbitrage

import (
	"math"
	"time"
	"traiderBot/internal/dataloader"
	"traiderBot/internal/graphb"
	"traiderBot/internal/observe"
	"traiderBot/internal/orderbook"
	"traiderBot/internal/route"
)

// // FeeRate — комиссии по паре
// type FeeRate struct {
// 	Maker float64
// 	Taker float64
// }

// // SymbolFilters — нужные фильтры Binance
// type SymbolFilters struct {
// 	MinQty      float64
// 	MaxQty      float64
// 	StepSize    float64
// 	MinNotional float64
// 	TickSize    float64
// }

// StepResult — результат одного шага
type StepResult struct {
	Edge      graphb.Edge
	Side      string
	AvgPrice  float64
	Quantity  float64 // базовый актив
	QuoteQty  float64 // котировочный актив
	Fee       float64
	FeeAsset  string
	AmountIn  float64
	AmountOut float64
}

// Opportunity — готовая арбитражная возможность
type Opportunity struct {
	Route       route.Route
	Steps       []StepResult
	StartAsset  string
	StartAmount float64
	EndAmount   float64
	ProfitAbs   float64
	ProfitPct   float64
	Timestamp   time.Time
}

// roundStep округляет количество вниз до stepSize
func roundStep(qty, step float64) float64 {
	if step <= 0 {
		return qty
	}
	return math.Floor(qty/step) * step
}

// walkBook покупает/продаёт, «съедая» стакан
// isBuy = true  → тратим quote, получаем base (идём по Asks)
// isBuy = false → тратим base, получаем quote (идём по Bids)
func walkBook(levels []orderbook.PriceLevel, amountIn float64, isBuy bool) (avgPrice, amountOut, filledQty float64, ok bool) {
	if len(levels) == 0 || amountIn <= 0 {
		return 0, 0, 0, false
	}

	var totalQuote, totalBase float64
	remaining := amountIn

	for _, level := range levels {
		if remaining <= 1e-12 {
			break
		}

		price := level.Price
		qty := level.Qty

		if isBuy {
			// Покупаем base за quote
			// Сколько quote можем потратить на этом уровне
			maxQuoteOnLevel := qty * price
			usedQuote := math.Min(remaining, maxQuoteOnLevel)
			gotBase := usedQuote / price

			totalQuote += usedQuote
			totalBase += gotBase
			remaining -= usedQuote
		} else {
			// Продаём base за quote
			usedBase := math.Min(remaining, qty)
			gotQuote := usedBase * price

			totalBase += usedBase
			totalQuote += gotQuote
			remaining -= usedBase
		}
	}

	if remaining > 1e-8 { // не хватило ликвидности
		return 0, 0, 0, false
	}

	if isBuy {
		avgPrice = totalQuote / totalBase
		return avgPrice, totalBase, totalBase, true
	}

	avgPrice = totalQuote / totalBase
	return avgPrice, totalQuote, totalBase, true
}

func SimulateRoute(
	route route.Route,
	startAmount float64,
	market *orderbook.Market,
	fees map[string]dataloader.FeeRate,
	filters map[string]dataloader.SymbolFilters,
	minProfitPct float64,
	topPercent *float64,
	totalPercent *float64,
	totalSimulatedCounter *int,
	simCounters *observe.SIMCounters,
) *Opportunity {

	if len(route.Edges) != 3 {
		simCounters.InvalidRouteLength.Add(1)
		return nil
	}

	currentAmount := startAmount
	startAsset := route.Edges[0].From
	steps := make([]StepResult, 0, 3)

	for _, edge := range route.Edges {
		ob, ok := market.GetOrderBook(edge.Symbol)
		if !ok || ob == nil {
			simCounters.OrderbookNotFound.Add(1)
			return nil
		}

		ob.Mu.RLock()
		var levels []orderbook.PriceLevel
		if edge.IsBuy {
			levels = ob.Asks
		} else {
			levels = ob.Bids
		}
		levelsCopy := make([]orderbook.PriceLevel, len(levels))
		copy(levelsCopy, levels)
		ob.Mu.RUnlock()

		feeRate, ok := fees[edge.Symbol]
		if !ok {
			simCounters.FeeNotFound.Add(1)
			return nil
		}

		filter, ok := filters[edge.Symbol]
		if !ok {
			simCounters.FiltersNotFound.Add(1)
			return nil
		}

		avgPrice, amountOut, baseQty, ok := walkBook(levelsCopy, currentAmount, edge.IsBuy)
		if !ok {
			simCounters.NotEnoughLiquidity.Add(1)
			return nil
		}

		fee := amountOut * feeRate.Taker
		amountOutAfterFee := amountOut - fee

		qtyRounded := roundStep(baseQty, filter.StepSize)

		if qtyRounded < filter.MinQty {
			simCounters.LotSize.Add(1)
			return nil
		}

		notional := qtyRounded * avgPrice

		if notional < filter.MinNotional {
			simCounters.MinNotional.Add(1)
			return nil
		}

		side := "SELL"
		if edge.IsBuy {
			side = "BUY"
		}

		step := StepResult{
			Edge:      edge,
			Side:      side,
			AvgPrice:  avgPrice,
			Quantity:  qtyRounded,
			QuoteQty:  notional,
			Fee:       fee,
			FeeAsset:  edge.To,
			AmountIn:  currentAmount,
			AmountOut: amountOutAfterFee,
		}

		steps = append(steps, step)

		currentAmount = amountOutAfterFee
	}

	if route.Edges[2].To != startAsset {
		simCounters.StartAsset.Add(1)
		return nil
	}

	profitAbs := currentAmount - startAmount
	profitPct := profitAbs / startAmount
	*topPercent = math.Max(*topPercent, profitPct)
	*totalPercent += profitPct
	*totalSimulatedCounter++

	if profitPct < minProfitPct {
		simCounters.ProfitTooLow.Add(1)
		return nil
	}

	simCounters.OpportunityFound.Add(1)

	return &Opportunity{
		Route:       route,
		Steps:       steps,
		StartAsset:  startAsset,
		StartAmount: startAmount,
		EndAmount:   currentAmount,
		ProfitAbs:   profitAbs,
		ProfitPct:   profitPct,
		Timestamp:   time.Now(),
	}
}
