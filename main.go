package main

import (
	"context"
	"fmt"
	"log"
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
	defer logger.Sync()

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
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()

	// 1. Создаём рынок с глубиной 10
	m, err := orderbook.LoadMarket(pairs, 10)
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

	startAmounts := map[string]float64{
		"USDT": 100.0, // 100 USDT
		"BTC":  0.002, // ≈ 100–130 USDT в зависимости от курса
		"ETH":  0.03,  // ≈ 100 USDT
		"BNB":  0.15,  // ≈ 100 USDT
	}



	for i := range 1000000 {
		if i%10000 == 0 {
			fmt.Printf("Cicle %d\n", i%1000000)
		}
		for _, tR := range tradeRoutes {
			// route := ... // твой найденный Route

			startAsset := tR[0].From
			startAmount, ok := startAmounts[startAsset]
			if !ok {
				// можно поставить дефолт или пропустить
				startAmount = 50.0
			}

			opp := arbitrage.SimulateRoute(
				tR,
				startAmount,
				m, // ты создаёшь сам
				fees,
				filters,
				0.0005, // минимум 0.05% прибыли
				logger,
			)

			if opp != nil {
				fmt.Printf("Найдена возможность! Прибыль: %.4f%% (%.6f %s)\n",
					opp.ProfitPct*100, opp.ProfitAbs, opp.StartAsset)
			}
		}
	}

	fmt.Println("Routes was handled by 100 times")
	fmt.Printf("%v\n", time.Since(start))
	start = time.Now()
}

// func main() {
// 	// 1. Создаём рынок с глубиной 10
// 	m, err := orderbook.LoadMarket([]string{"BTCUSDT", "ETHUSDT", "ETHBTC"}, 10)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// 2. Запускаем real-time обновления через Partial Book Depth
// 	m.UpdateOrderBook("BTCUSDT")
// 	m.UpdateOrderBook("ETHUSDT")
// 	m.UpdateOrderBook("ETHBTC")

// 	// 3. Читаем данные
// 	for {
// 		time.Sleep(2 * time.Second)

// 		if ob, ok := m.GetOrderBook("BTCUSDT"); ok {
// 			bid, bidQty, _ := ob.GetBestBid()
// 			ask, askQty, _ := ob.GetBestAsk()
// 			fmt.Printf("BTCUSDT  bid=%s (%s)  ask=%s (%s)\n", bid, bidQty, ask, askQty)
// 		}
// 		if ob, ok := m.GetOrderBook("ETHUSDT"); ok {
// 			bid, bidQty, _ := ob.GetBestBid()
// 			ask, askQty, _ := ob.GetBestAsk()
// 			fmt.Printf("ETHUSDT  bid=%s (%s)  ask=%s (%s)\n", bid, bidQty, ask, askQty)
// 		}
// 		if ob, ok := m.GetOrderBook("ETHBTC"); ok {
// 			bid, bidQty, _ := ob.GetBestBid()
// 			ask, askQty, _ := ob.GetBestAsk()
// 			fmt.Printf("ETHBTC  bid=%s (%s)  ask=%s (%s)\n", bid, bidQty, ask, askQty)
// 		}
// 	}
// }
