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

// walkBook покупает/продаёт, «съедая» стакан
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
			// защита от кривых данных в стакане - такой уровень пропускаем
			continue
		}

		if edge.IsBuy {
			maxQuoteOnLevel := qty * price
			usedQuote := math.Min(remaining, maxQuoteOnLevel)
			gotBase := usedQuote / price

			totalQuote += usedQuote
			totalBase += gotBase
			remaining -= usedQuote
		} else {
			usedBase := math.Min(remaining, qty)
			gotQuote := usedBase * price

			totalBase += usedBase
			totalQuote += gotQuote
			remaining -= usedBase
		}
	}

	if remaining > 1e-8 {
		fail(
			ReasonInsufficientDepth,
			amountIn-remaining,
			remaining,
		)

		return 0, 0, 0, ReasonInsufficientDepth, false
	}

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

		avgPrice, amountOut, baseQty, liquidityReason, ok := walkBook(
			levelsCopy,
			currentAmount,
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

		return SimulationResult{
			Reason: SimulationStartAsset,
		}
	}

	profitAbs := currentAmount - startAmount
	profitPct := profitAbs / startAmount

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
			Route:       route,
			Steps:       steps,
			StartAsset:  startAsset,
			StartAmount: startAmount,
			EndAmount:   currentAmount,
			ProfitAbs:   profitAbs,
			ProfitPct:   profitPct,
			Timestamp:   time.Now(),
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
