package arbitrage

import (
	"math"
	"time"
	"traiderBot/internal/dataloader"
	"traiderBot/internal/graph"
	"traiderBot/internal/orderbook"
	"traiderBot/internal/route"

	"go.uber.org/zap"
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
	Edge      graph.Edge
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
	logger *zap.Logger, // передаём логгер снаружи
) *Opportunity {

	logger.Debug("SIM start",
		zap.Any("route", route),
		zap.Float64("startAmount", startAmount),
	)

	if len(route) != 3 {
		logger.Warn("SIM invalid route length",
			zap.Int("len", len(route)),
		)
		return nil
	}

	currentAmount := startAmount
	startAsset := route[0].From
	steps := make([]StepResult, 0, 3)

	for i, edge := range route {
		sideStr := "SELL"
		if edge.IsBuy {
			sideStr = "BUY"
		}

		logger.Debug("SIM step",
			zap.Int("step", i+1),
			zap.String("symbol", edge.Symbol),
			zap.String("side", sideStr),
			zap.Float64("amountIn", currentAmount),
		)

		ob, ok := market.GetOrderBook(edge.Symbol)
		if !ok || ob == nil {
			logger.Debug("SIM orderbook not found",
				zap.String("symbol", edge.Symbol),
			)
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

		logger.Debug("SIM book levels",
			zap.Int("levels", len(levelsCopy)),
		)

		feeRate, ok := fees[edge.Symbol]
		if !ok {
			logger.Warn("SIM fee not found",
				zap.String("symbol", edge.Symbol),
			)
			return nil
		}

		filter, ok := filters[edge.Symbol]
		if !ok {
			logger.Warn("SIM filters not found",
				zap.String("symbol", edge.Symbol),
			)
			return nil
		}

		avgPrice, amountOut, baseQty, ok := walkBook(levelsCopy, currentAmount, edge.IsBuy)
		if !ok {
			logger.Debug("SIM not enough liquidity",
				zap.String("symbol", edge.Symbol),
				zap.Float64("amountIn", currentAmount),
			)
			return nil
		}

		logger.Debug("SIM walkBook",
			zap.Float64("avgPrice", avgPrice),
			zap.Float64("baseQty", baseQty),
			zap.Float64("amountOut", amountOut),
		)

		fee := amountOut * feeRate.Taker
		amountOutAfterFee := amountOut - fee

		logger.Debug("SIM fee",
			zap.Float64("feeRate", feeRate.Taker),
			zap.Float64("fee", fee),
			zap.Float64("amountAfterFee", amountOutAfterFee),
		)

		qtyRounded := roundStep(baseQty, filter.StepSize)

		if qtyRounded < filter.MinQty {
			logger.Debug("SIM LOT_SIZE failed",
				zap.String("symbol", edge.Symbol),
				zap.Float64("qty", baseQty),
				zap.Float64("rounded", qtyRounded),
				zap.Float64("minQty", filter.MinQty),
			)
			return nil
		}

		notional := qtyRounded * avgPrice

		if notional < filter.MinNotional {
			logger.Debug("SIM MIN_NOTIONAL failed",
				zap.String("symbol", edge.Symbol),
				zap.Float64("notional", notional),
				zap.Float64("minNotional", filter.MinNotional),
			)
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

		logger.Debug("SIM step completed",
			zap.Int("step", i+1),
			zap.Float64("amountOut", amountOutAfterFee),
		)

		currentAmount = amountOutAfterFee
	}

	if route[2].To != startAsset {
		logger.Warn("SIM route doesn't return to start asset",
			zap.String("start", startAsset),
			zap.String("end", route[2].To),
		)
		return nil
	}

	profitAbs := currentAmount - startAmount
	profitPct := profitAbs / startAmount

	logger.Debug("SIM result",
		zap.Float64("endAmount", currentAmount),
		zap.Float64("profitAbs", profitAbs),
		zap.Float64("profitPct", profitPct*100),
	)

	if profitPct < minProfitPct {
		logger.Debug("SIM profit too low",
			zap.Float64("profitPct", profitPct*100),
			zap.Float64("minProfitPct", minProfitPct*100),
		)
		return nil
	}

	logger.Info("SIM opportunity found",
		zap.Float64("profitPct", profitPct*100),
	)

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
