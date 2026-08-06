package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"traiderBot/internal/arbitrage"
	"traiderBot/internal/arbitrage/deduplicate"
	"traiderBot/internal/binance"
	"traiderBot/internal/dataloader"
	"traiderBot/internal/graphb"
	"traiderBot/internal/myLogger"
	"traiderBot/internal/observe"
	"traiderBot/internal/orderbook"
	"traiderBot/internal/route"
)

var (
	startAmounts = map[string]float64{
		"USDT": 100.0, // 100 USDT
		"BTC":  0.002, // ≈ 100–130 USDT в зависимости от курса
		"ETH":  0.03,  // ≈ 100 USDT
		"BNB":  0.15,  // ≈ 100 USDT
	}
)

type ExchangeData struct {
	Pairs      []string                            `json:"pairs"`
	Filters    map[string]dataloader.SymbolFilters `json:"filters"`
	Fees       map[string]dataloader.FeeRate       `json:"fees"`
	Graph      graphb.Graph                        `json:"graph"`
	Routes     []route.Route                       `json:"routes"`
	RouteIndex map[string][]route.Route            `json:"routeIndex"`
}

// "exchange_data.json"
func GetCache(filePath string) ExchangeData {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var data ExchangeData

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		log.Fatal(err)
	}

	return data

}

func SaveCache(filePath string, data ExchangeData) {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // красивое форматирование (необязательно)

	if err := encoder.Encode(data); err != nil {
		log.Fatal(err)
	}

}

func LoadExchangeData(ctx context.Context, symbolsCount int) ExchangeData {
	const cacheFile = "exchange_data.json"
	if symbolsCount == 0 {
		symbolsCount = 50
	}

	var data ExchangeData

	client := dataloader.NewClient()

	if stat, err := os.Stat(cacheFile); err == nil && stat.Size() > 0 {
		// Кэш существует и не пустой
		data = GetCache(cacheFile)

		fmt.Println("Exchange data loaded from cache")
		fmt.Println(time.Now())
	} else {
		// Кэша нет или файл пустой — загружаем с Binance
		pairs, filters, err := dataloader.LoadExchangeInfo(ctx, symbolsCount)
		if err != nil {
			log.Fatal(err)
		}

		fees, err := client.LoadFees(ctx, pairs)
		if err != nil {
			log.Fatal(err)
		}
		data = ExchangeData{
			Pairs:   pairs,
			Filters: filters,
			Fees:    fees,
		}
		data.Graph, _ = graphb.NewGraph(data.Pairs)
		data.Routes = route.FindTriangularRoutes(data.Graph, startAmounts)
		data.RouteIndex = route.BuildRouteIndex(data.Routes)

		SaveCache(cacheFile, data)

		fmt.Println("Exchange data saved to cache")
	}

	return data
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	logger := myLogger.NewLogger()
	loggerF, err := myLogger.NewFileLogger("app.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()
	defer loggerF.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT)

	defer stop()

	echData := LoadExchangeData(ctx, 50)
	simCounters := observe.NewSIMCounters()

	m := orderbook.NewMarket(10)
	m.Start(echData.Pairs)

	start := time.Now()
	loggerF.Info(fmt.Sprintf("Session started at %v", start))

	// При старте
	updateCh := make(chan string, 256) // буфер, чтобы WS не блокировался

	defer close(updateCh)

	m.SetOnUpdate(func(symbol string) {
		// Очень быстро, без блокировок
		select {
		case updateCh <- symbol:
		default:
			// если канал переполнен — можно пропустить или залогировать
		}
	})

	const (
		maxBookAge    = 301 * time.Millisecond
		dedupWindow   = 300 * time.Millisecond
		cleanupMaxAge = dedupWindow * 5 // 1.5 секунды
	)

	dedup := deduplicate.NewDedupStore(dedupWindow)
	symbolChecksCounter := 0
	oldSymbols := 0
	unfreshRoutesCounter := 0
	freshRoutesCounter := 0
	okStartAmountCounter := 0
	totalSimulationsStarted := 0
	dubsCheckCounter := 0
	wrongStartAmount := 0
	var topPercent float64 = -1
	var totalPercent float64 = 0
	var totalSimulations int = 0

	go func() {

		// ... внутри воркера
		for symbol := range updateCh {
			symbolChecksCounter++
			routes := echData.RouteIndex[symbol]

			for _, r := range routes {
				if len(r.OldSymbols) != 0 {
					if _, ok := r.OldSymbols[symbol]; !ok {
						oldSymbols++
						continue
					}

					delete(r.OldSymbols, symbol)

					if len(r.OldSymbols) != 0 {
						oldSymbols++
						continue
					}
				}

				if !r.UpdateFreshness(m, maxBookAge) {
					unfreshRoutesCounter++
					continue
				}
				freshRoutesCounter++
				startAsset := r.Edges[0].From
				startAmount, ok := startAmounts[startAsset]
				if !ok {
					wrongStartAmount++
					continue
				}
				okStartAmountCounter++

				opp := arbitrage.SimulateRoute(
					r,
					startAmount,
					m,
					echData.Fees,
					echData.Filters,
					0.0005,
					&topPercent,
					&totalPercent,
					&totalSimulations,
					simCounters,
				)
				totalSimulationsStarted++
				if opp == nil {
					continue
				}
				routeKey := route.RouteKey(opp.Route)
				now := time.Now()

				isDup := dedup.IsDuplicate(routeKey, now)
				if isDup {
					dubsCheckCounter++
					// можно логировать с флагом duplicate=true
					continue
				}
				// новая возможность
				dedup.MarkSeen(routeKey, now)
				cB := binance.NewClient()

				for idx, step := range opp.Steps {
					st := time.Now()
					response, err := cB.PlaceLimitIOC(ctx, step)
					if err != nil {
						fmt.Println(err)
					}
					fmt.Printf(`
					idx %d -- time sinse start %v
					%#v
					`, idx, time.Since(st), response)
				}

				// здесь логируем возможность + ages из freshness.Ages
				loggerF.Info(fmt.Sprintf("Найдена возможность! Прибыль: %.4f%% (%.6f %s)\n",
					opp.ProfitPct*100, opp.ProfitAbs, opp.StartAsset))

			}
		}
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Second) // как часто чистим
		logTicker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				dedup.Cleanup(cleanupMaxAge)
			case <-logTicker.C:

				loggerF.StatsLog("stats.log",
					fmt.Sprintf(`
starts from %v

symbolChecksCounter %d

oldSymbolsCounet %d

totalSimulationsStarted %d

dubsCheckCounter %d

unfreshRoutesCounter %d

freshRoutesCounter %d

wrongStartAmount %d

okStartAmountCounter %d

totalSimulations %d

top percent %f

average percent = %f

working time -- %v

%s
`,

						start,
						symbolChecksCounter,
						oldSymbols,
						totalSimulationsStarted,
						dubsCheckCounter,
						unfreshRoutesCounter,
						freshRoutesCounter,
						wrongStartAmount,
						okStartAmountCounter,
						totalSimulations,
						topPercent,
						totalPercent/float64(totalSimulations),
						time.Since(start),
						fmt.Sprintf(`
------simFailuresCounters------

invalid-route-length %d

orderbook-not-found %d

fee-not-found %d

filters-not-found %d

not-enough-liquidity %d

LOT_SIZE-failed %d

MIN_NOTIONAL-failed %d

route doesn't return to start asset %d

profit too low %d

opportunity found %d
						`,
							simCounters.InvalidRouteLength.Load(),
							simCounters.OrderbookNotFound.Load(),
							simCounters.FeeNotFound.Load(),
							simCounters.FiltersNotFound.Load(),
							simCounters.NotEnoughLiquidity.Load(),
							simCounters.LotSize.Load(),
							simCounters.MinNotional.Load(),
							simCounters.StartAsset.Load(),
							simCounters.ProfitTooLow.Load(),
							simCounters.OpportunityFound.Load(),
						),
					),
				)

			case <-ctx.Done(): // если есть контекст для остановки
				return
			}
		}
	}()

	select {

	case <-ctx.Done():
		fmt.Println("Получен сигнал отмены Ctrl + c")

		fmt.Println("Канал закрыт")
		loggerF.Info(fmt.Sprintf("symbolChecksCounter %d\n", symbolChecksCounter))
		loggerF.Info(fmt.Sprintf("totalSimulationsStarted %d\n", totalSimulationsStarted))
		loggerF.Info(fmt.Sprintf("dubsCheckCounter %d\n", dubsCheckCounter))
		loggerF.Info(fmt.Sprintf("unfreshRoutesCounter %d\n", unfreshRoutesCounter))
		loggerF.Info(fmt.Sprintf("Top percent %f\n", topPercent))
		loggerF.Info(fmt.Sprintf("average percent = %f", totalPercent/float64(totalSimulations)))
		loggerF.Info(fmt.Sprintf("Working time -- %v\n", time.Since(start)))
	}
}

// package main

// import (
// 	"fmt"
// 	"time"
// 	"traiderBot/internal/myLogger"
// 	// "traiderBot/internal/myLogger/logReader"
// )

// func main() {
// 	loggerF, _ := myLogger.NewFileLogger("app.log")

// 	// logReader.CheckCummulativeProfit("app.log")
// 	loggerF.Info(fmt.Sprintf("Session started at %v", time.Now()))

// }
