package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/infrastructure"
	"traiderBot/internalDDD/market/infrastructure/binance"
	"traiderBot/internalDDD/market/infrastructure/cache"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Market structure.
	symbolProvider := binance.NewSymbolProvider()

	marketStructureService := application.NewMarketStructureService(
		symbolProvider,
	)

	marketCache := cache.NewMarketCache("market_cache.json")

	// Market data.
	orderBookProvider := binance.NewOrderBookProvider()
	orderBookStream := binance.NewOrderBookStream()
	orderBookStore := infrastructure.NewOrderBookStore()

	marketDataService := application.NewMarketDataService(
		orderBookProvider,
		orderBookStream,
		orderBookStore,
	)

	// Main market orchestration service.
	marketService := application.NewMarketService(
		marketCache,
		marketStructureService,
		marketDataService,
	)

	marketDataService.ShowStore()

	startAssets := map[string]struct{}{
		"USDT": {},
	}

	const depth = 20

	log.Println("starting market service...")

	go func() {
		if err := marketService.Start(
			ctx,
			startAssets,
			depth,
		); err != nil {
			if ctx.Err() != nil {
				log.Println("market service stopped")
				return
			}

			log.Fatalf("market service failed: %v", err)
		}
	}()

	

	select {
	case <-ctx.Done():
		log.Println("market service stopped")
		
	}

}
