package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"

	// "time"
	// orderbook "traiderBot/internal/orderBook"

	"traiderBot/internal/arbitrage"
	"traiderBot/internal/dataloader"
	"traiderBot/internal/graph"
	"traiderBot/internal/myLogger"
	"traiderBot/internal/orderbook"
	"traiderBot/internal/route"
)

// var pairs = []string{"BTCUSDT",
// 	"ETHBTC",
// 	"ETHUSDT",
// 	"SOLBTC",
// 	"SOLUSDT",
// 	"SOLETH",
// 	"BNBBTC",
// 	"BNBUSDT"}

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

	ctx := context.Background()

	client := dataloader.NewClient()

	start := time.Now()
	pairs, filters, err := dataloader.LoadExchangeInfo(ctx, 50)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Pairs and filters loaded")
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()

	fees, err := client.LoadFees(ctx, pairs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Fees are loaded")
	fmt.Printf("%#v\n", fees)
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()
	if true {
		return
	}

	// 1. Создаём рынок с глубиной 10
	m := orderbook.NewMarket(10)
	m.Start(pairs)
	// m, err := orderbook.LoadMarket(pairs, 10)
	if err != nil {
		panic(err)
	}

	fmt.Println("Orderbook are loaded")
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()

	g, err := graph.NewGraph(pairs)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Grapth was builded loaded")
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()

	tradeRoutes := route.FindTriangularRoutes(g)

	fmt.Println("Routes are built")
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()

	routeIndex := route.BuildRouteIndex(tradeRoutes)

	startAmounts := map[string]float64{
		"USDT": 100.0, // 100 USDT
		"BTC":  0.002, // ≈ 100–130 USDT в зависимости от курса
		"ETH":  0.03,  // ≈ 100 USDT
		"BNB":  0.15,  // ≈ 100 USDT
	}

	// При старте
	updateCh := make(chan string, 256) // буфер, чтобы WS не блокировался

	m.SetOnUpdate(func(symbol string) {
		// Очень быстро, без блокировок
		select {
		case updateCh <- symbol:
		default:
			// если канал переполнен — можно пропустить или залогировать
		}
	})

	var cycles int64 = 10000000
	var counter atomic.Int64
	var percent int64 = 100000
	start = time.Now()

	// Воркер
	go func() {
		i := 0
		for symbol := range updateCh {
			currentCount := counter.Add(1)

			// Проверяем процент выполнения
			if currentCount%percent == 0 {
				fmt.Printf("workers are handled %d/%d of cycles --- %d%%\n", currentCount, cycles, (currentCount*100)/cycles)

				d := time.Since(start)
				h := int(d.Hours())
				m := int(d.Minutes()) % 60
				s := int(d.Seconds()) % 60
				fmt.Printf("Time passed -- %02d:%02d:%02d\n", h, m, s)
			}
			routes := routeIndex[symbol]
			for _, r := range routes {
				startAsset := r[0].From
				startAmount, ok := startAmounts[startAsset]
				if !ok {
					// можно поставить дефолт или пропустить
					// startAmount = 50.0
				}
				opp := arbitrage.SimulateRoute(r, startAmount, m, fees, filters, 0.0005, logger)
				if opp != nil {
					// нашли возможность — отправляем дальше
					// например в другой канал opportunityCh <- opp
					loggerF.Info(opp)

				}
				i++
			}
		}
	}()

	time.Sleep(2 * time.Hour)
	// for i := range 1000000 {
	// 	if i%10000 == 0 {
	// 		fmt.Printf("Cicle %d\n", i%1000000)
	// 	}
	// 	for _, tR := range tradeRoutes {
	// 		// route := ... // твой найденный Route

	// 		startAsset := tR[0].From
	// 		startAmount, ok := startAmounts[startAsset]
	// 		if !ok {
	// 			// можно поставить дефолт или пропустить
	// 			startAmount = 50.0
	// 		}

	// 		opp := arbitrage.SimulateRoute(
	// 			tR,
	// 			startAmount,
	// 			m, // ты создаёшь сам
	// 			fees,
	// 			filters,
	// 			0.0005, // минимум 0.05% прибыли
	// 			logger,
	// 		)

	// 		if opp != nil {
	// 			fmt.Printf("Найдена возможность! Прибыль: %.4f%% (%.6f %s)\n",
	// 				opp.ProfitPct*100, opp.ProfitAbs, opp.StartAsset)
	// 		}
	// 	}
	// }

	// fmt.Println("Routes was handled by 100 times")
	// fmt.Printf("%v\n", time.Since(start))
	// start = time.Now()
}



// package main

// import (
// 	"traiderBot/internal/myLogger/logReader"
// )

// func main() {
// 	logReader.CheckCummulativeProfit("app.log")
// }