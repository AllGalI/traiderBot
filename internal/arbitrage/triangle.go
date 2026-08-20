package arbitrage

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"
	"traiderBot/internal/dataloader"
	"traiderBot/internal/graphb"
	"traiderBot/internal/observe"
	"traiderBot/internal/orderbook"
	"traiderBot/internal/route"
	"traiderBot/stats"
)

// StepResult — результат одного шага
type StepResult struct {
	Symbol    string
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

func (op *Opportunity) SaveLog() error {
	err := stats.SaveLog("./stats/opportunities.json", map[string]any{
		"startAsset":  op.StartAsset,
		"startAmount": op.StartAmount,
		"endAmount":   op.EndAmount,
		"profitAbs":   op.ProfitAbs,
		"profitPct":   op.ProfitPct,
		"timestamp":   op.Timestamp,
	})
	return err
}

// roundStep округляет количество вниз до stepSize
func roundStep(qty, step float64) float64 {
	if step <= 0 {
		return qty
	}
	return math.Floor(qty/step) * step
}

type LiquidityFailReason uint8

const (
	ReasonNone LiquidityFailReason = iota
	ReasonInvalidInput
	ReasonInsufficientDepth
	ReasonBadLevelData
)

type LiquidityFailData struct {
	Symbol      string
	Time        time.Time
	IsBuy       bool
	AmountIn    float64
	Filled      float64 // сколько успели исполнить до отказа
	Remaining   float64 // сколько осталось неисполненным
	LevelsCount int
	BestPrice   float64 // цена лучшего уровня в стакане (для контекста)
	WorstPrice  float64 // цена худшего просмотренного уровня
	Reason      LiquidityFailReason
}

// LiquidityFails - потокобезопасный кольцевой буфер последних отказов,
// чтобы потом можно было посмотреть, что именно происходило со стаканом в момент отказа.
type LiquidityFails struct {
	mu      sync.Mutex
	items   []LiquidityFailData
	maxSize int
}

func NewLiquidityFails(maxSize int) *LiquidityFails {
	_ = os.Remove("./stats/notEnougthLiq.json")
	fmt.Printf("file removed %s\n", "./stats/notEnougthLiq.json")

	return &LiquidityFails{maxSize: maxSize}
}

func (l *LiquidityFails) Add(d LiquidityFailData) {
	var batch []LiquidityFailData

	l.mu.Lock()

	l.items = append(l.items, d)

	if len(l.items) == l.maxSize {
		batch = l.items
		l.items = make([]LiquidityFailData, 0, l.maxSize)
	}

	l.mu.Unlock()

	if batch != nil {
		if err := stats.SaveLog(
			"./stats/notEnougthLiq.json",
			batch,
		); err != nil {
			// обработать ошибку
		}
	}
}

func (l *LiquidityFails) Snapshot() []LiquidityFailData {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]LiquidityFailData, len(l.items))
	copy(out, l.items)
	return out
}

const liquidityEpsilon = 1e-12

// walkBook покупает/продаёт, «съедая» стакан.
//
// isBuy = true  → тратим quote, получаем base (идём по Asks)
// isBuy = false → тратим base, получаем quote (идём по Bids)
// walkBook покупает/продаёт, «съедая» стакан
//
// isBuy = true  → тратим quote, получаем base (идём по Asks)
// isBuy = false → тратим base, получаем quote (идём по Bids)
func walkBook(
	levels []orderbook.PriceLevel,
	amountIn float64,
	edge *graphb.Edge,
	liqFails *LiquidityFails,
) (
	avgPrice,
	amountOut,
	filledQty float64,
	reason LiquidityFailReason,
	ok bool,
) {
	fail := func(reason LiquidityFailReason, filled, remaining float64) {
		if liqFails == nil {
			return
		}

		d := LiquidityFailData{
			Symbol:      edge.Symbol,
			Time:        time.Now(),
			IsBuy:       edge.IsBuy,
			AmountIn:    amountIn,
			Filled:      filled,
			Remaining:   remaining,
			LevelsCount: len(levels),
			Reason:      reason,
		}

		if len(levels) > 0 {
			d.BestPrice = levels[0].Price
			d.WorstPrice = levels[len(levels)-1].Price
		}

		liqFails.Add(d)
	}

	if len(levels) == 0 || amountIn <= 0 {
		fail(ReasonInvalidInput, 0, amountIn)

		return 0, 0, 0, ReasonInvalidInput, false
	}

	var totalQuote, totalBase float64
	remaining := amountIn

	for _, level := range levels {
		if remaining <= 1e-12 {
			break
		}

		price := level.Price
		qty := level.Qty

		if price <= 0 || qty <= 0 {
			// Защита от кривых данных в стакане.
			continue
		}

		if edge.IsBuy {
			// BUY:
			// amountIn  = quote
			// amountOut = base

			maxQuoteOnLevel := qty * price
			usedQuote := math.Min(remaining, maxQuoteOnLevel)
			gotBase := usedQuote / price

			totalQuote += usedQuote
			totalBase += gotBase
			remaining -= usedQuote
		} else {
			// SELL:
			// amountIn  = base
			// amountOut = quote

			usedBase := math.Min(remaining, qty)
			gotQuote := usedBase * price

			totalBase += usedBase
			totalQuote += gotQuote
			remaining -= usedBase
		}
	}

	if remaining > 1e-8 {
		/*
			ВАЖНО:

			Раньше здесь мы возвращали false.

			Теперь возвращаем фактически исполненную часть.

			SimulateRoute увидит ReasonInsufficientDepth
			и посчитает коэффициент этой ноги.
		*/

		filledInput := amountIn - remaining

		fail(
			ReasonInsufficientDepth,
			filledInput,
			remaining,
		)

		if totalBase <= 0 {
			return 0, 0, 0, ReasonInsufficientDepth, false
		}

		avgPrice = totalQuote / totalBase

		if edge.IsBuy {
			return avgPrice, totalBase, totalBase, ReasonInsufficientDepth, true
		}

		return avgPrice, totalQuote, totalBase, ReasonInsufficientDepth, true
	}

	/*
		Ниже полностью сохраняем старую логику
		для 100% заполненного стакана.
	*/

	if totalBase <= 0 {
		fail(
			ReasonBadLevelData,
			amountIn-remaining,
			remaining,
		)

		return 0, 0, 0, ReasonBadLevelData, false
	}

	avgPrice = totalQuote / totalBase

	if edge.IsBuy {
		return avgPrice, totalBase, totalBase, ReasonNone, true
	}

	return avgPrice, totalQuote, totalBase, ReasonNone, true
}

func SimulateRoute(
	route route.Route,
	startAmount float64,
	market *orderbook.Market,
	fees map[string]dataloader.FeeRate,
	filters map[string]dataloader.SymbolFilters,
	minProfitPct float64,
	minLiquidityFillRatio float64,
	topPercent *float64,
	totalPercent *float64,
	totalSimulatedCounter *int,
	simCounters *observe.SIMCounters,
	liqFails *LiquidityFails,
) SimulationResult {
	if len(route.Edges) != 3 {
		simCounters.InvalidRouteLength.Add(1)

		return SimulationResult{
			Reason: SimulationInvalidRoute,
		}
	}

	currentAmount := startAmount
	startAsset := route.Edges[0].From

	steps := make([]StepResult, 0, 3)

	/*
		dealCoef = какая доля исходной сделки реально
		прошла через весь маршрут.

		Изначально считаем, что ликвидности хватает на 100%.
	*/
	dealCoef := 1.0

	/*
		Нужен флаг, потому что при полном заполнении
		мы вообще НЕ хотим менять старую математику.
	*/
	hasPartialFill := false

	for _, edge := range route.Edges {
		ob, ok := market.GetOrderBook(edge.Symbol)
		if !ok || ob == nil {
			simCounters.OrderbookNotFound.Add(1)

			return SimulationResult{
				Reason: SimulationOrderBookNotFound,
			}
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

			return SimulationResult{
				Reason: SimulationFeeNotFound,
			}
		}

		filter, ok := filters[edge.Symbol]
		if !ok {
			simCounters.FiltersNotFound.Add(1)

			return SimulationResult{
				Reason: SimulationFiltersNotFound,
			}
		}

		/*
			Запоминаем, сколько хотели провести
			через эту конкретную ногу.
		*/
		amountIn := currentAmount

		avgPrice, amountOut, baseQty, liquidityReason, ok := walkBook(
			levelsCopy,
			amountIn,
			&edge,
			liqFails,
		)

		if !ok {
			simCounters.NotEnoughLiquidity.Add(1)

			return SimulationResult{
				Reason:          SimulationInsufficientLiquidity,
				LiquidityReason: liquidityReason,
				LiquiditySymbol: edge.Symbol,
			}
		}

		/*
			======================================================
			PARTIAL FILL
			======================================================

			Если стакана хватило полностью:

				liquidityReason == ReasonNone

			Ничего здесь не делаем.

			Поэтому старый расчёт для полного маршрута
			остаётся полностью прежним.
		*/

		if liquidityReason == ReasonInsufficientDepth {
			hasPartialFill = true

			/*
				Определяем, сколько входной валюты реально
				было исполнено.

				BUY:

					amountIn = quote
					baseQty  = полученный base
					avgPrice = quote / base

					=> реально потраченный quote:

						baseQty * avgPrice

				SELL:

					amountIn = base
					baseQty  = реально проданный base

					=> filledInput = baseQty
			*/

			var filledInput float64

			if edge.IsBuy {
				filledInput = baseQty * avgPrice
			} else {
				filledInput = baseQty
			}

			if amountIn <= 1e-12 || filledInput <= 1e-12 {
				simCounters.NotEnoughLiquidity.Add(1)

				return SimulationResult{
					Reason:          SimulationInsufficientLiquidity,
					LiquidityReason: liquidityReason,
					LiquiditySymbol: edge.Symbol,
				}
			}

			/*
				Коэффициент заполнения этой конкретной ноги.

				Например:

					amountIn  = 100 USDT
					filled    = 75 USDT

					legCoef = 0.75
			*/

			legCoef := filledInput / amountIn

			/*
				Защита от кривых данных.
			*/
			if legCoef <= 0 {
				simCounters.NotEnoughLiquidity.Add(1)

				return SimulationResult{
					Reason:          SimulationInsufficientLiquidity,
					LiquidityReason: liquidityReason,
					LiquiditySymbol: edge.Symbol,
				}
			}

			if legCoef > 1 {
				legCoef = 1
			}

			/*
				Минимальный допустимый процент заполнения.

				Например:

					minLiquidityFillRatio = 0.50

					0.90 → принимаем
					0.75 → принимаем
					0.50 → принимаем
					0.49 → отклоняем
			*/
			if legCoef < minLiquidityFillRatio {
				simCounters.NotEnoughLiquidity.Add(1)

				return SimulationResult{
					Reason:          SimulationInsufficientLiquidity,
					LiquidityReason: liquidityReason,
					LiquiditySymbol: edge.Symbol,
				}
			}

			/*
				Накопительный коэффициент маршрута.

				Например:

					первая нога: 75%
					вторая:      80%
					третья:     100%

					dealCoef =
						1 × 0.75 × 0.80 × 1
						= 0.60
			*/
			dealCoef *= legCoef

			if dealCoef > 1 {
				dealCoef = 1
			}
		}

		/*
			======================================================
			Ниже оставляем ТВОЮ исходную математику без изменений.
			======================================================
		*/

		fee := amountOut * feeRate.Taker
		amountOutAfterFee := amountOut - fee

		qtyRounded := roundStep(baseQty, filter.StepSize)

		if qtyRounded < filter.MinQty {
			simCounters.LotSize.Add(1)

			return SimulationResult{
				Reason: SimulationLotSize,
			}
		}

		notional := qtyRounded * avgPrice

		if notional < filter.MinNotional {
			simCounters.MinNotional.Add(1)

			return SimulationResult{
				Reason: SimulationMinNotional,
			}
		}

		side := "SELL"

		if edge.IsBuy {
			side = "BUY"
		}

		step := StepResult{
			Symbol:    edge.Symbol,
			Side:      side,
			AvgPrice:  avgPrice,
			Quantity:  qtyRounded,
			QuoteQty:  notional,
			Fee:       fee,
			FeeAsset:  edge.To,
			AmountIn:  amountIn,
			AmountOut: amountOutAfterFee,
		}

		steps = append(steps, step)

		/*
			Ключевой момент:

			currentAmount всегда получает ФАКТИЧЕСКИЙ
			результат этой ноги.

			Если было partial fill — он уже уменьшен.
			Если было full fill — всё как раньше.
		*/
		currentAmount = amountOutAfterFee
	}

	if route.Edges[2].To != startAsset {
		simCounters.StartAsset.Add(1)

		return SimulationResult{
			Reason: SimulationStartAsset,
		}
	}

	/*
		==========================================================
		РАСЧЁТ ПРИБЫЛИ
		==========================================================

		Если все ноги заполнены полностью:

			hasPartialFill = false
			dealCoef       = 1

		Поэтому:

			actualStartAmount = startAmount

		и формула будет полностью идентична старой.
	*/

	actualStartAmount := startAmount

	if hasPartialFill {
		actualStartAmount = startAmount * dealCoef
	}

	if actualStartAmount <= 1e-12 {
		simCounters.NotEnoughLiquidity.Add(1)

		return SimulationResult{
			Reason: SimulationInsufficientLiquidity,
		}
	}

	profitAbs := currentAmount - actualStartAmount
	profitPct := profitAbs / actualStartAmount

	*topPercent = math.Max(*topPercent, profitPct)
	*totalPercent += profitPct
	*totalSimulatedCounter++

	if profitPct < minProfitPct {
		simCounters.ProfitTooLow.Add(1)

		return SimulationResult{
			Reason: SimulationProfitTooLow,
		}
	}

	simCounters.OpportunityFound.Add(1)

	return SimulationResult{
		Reason: SimulationOK,
		Opportunity: &Opportunity{
			Route:      route,
			Steps:      steps,
			StartAsset: startAsset,

			/*
				При full fill это ровно startAmount.

				При partial fill это реальный объём,
				который фактически прошёл через весь маршрут.
			*/
			StartAmount: actualStartAmount,

			EndAmount: currentAmount,
			ProfitAbs: profitAbs,
			ProfitPct: profitPct,
			Timestamp: time.Now(),
		},
	}
}

type SimulationReason uint8

const (
	SimulationOK SimulationReason = iota
	SimulationInvalidRoute
	SimulationOrderBookNotFound
	SimulationFeeNotFound
	SimulationFiltersNotFound
	SimulationInsufficientLiquidity
	SimulationLotSize
	SimulationMinNotional
	SimulationStartAsset
	SimulationProfitTooLow
)

type SimulationResult struct {
	Opportunity *Opportunity
	Reason      SimulationReason

	// Заполняется только для SimulationInsufficientLiquidity.
	LiquidityReason LiquidityFailReason
	LiquiditySymbol string
}
