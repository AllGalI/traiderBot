package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	debugApplication "traiderBot/internalDDD/debug/application"
	debugDomain "traiderBot/internalDDD/debug/domain"
	"traiderBot/internalDDD/debug/infrastructure/console"
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

	marketStructureStore := infrastructure.NewInMemoryMarketStructureStore()

	marketService := application.NewMarketService(
		marketCache,
		marketStructureService,
		marketDataService,
		marketStructureStore,
	)

	debugService := debugApplication.NewDebugService()

	graphProvider := debugDomain.NewGraphProvider(
		marketStructureStore,
	)

	debugService.Register(graphProvider)

	consoleHandler := console.NewHandler(
		debugService,
		os.Stdout,
	)

	commandHandler := console.NewCommandHandler(
		consoleHandler,
		os.Stdin,
		os.Stdout,
	)

	startAssets := map[string]struct{}{
		"USDT": {},
	}

	const depth = 10
	const symbolsLimit = 10

	log.Println("starting market service...")

	go func() {
		err := marketService.Start(
			ctx,
			startAssets,
			depth,
			symbolsLimit,
		)

		if err != nil {
			if ctx.Err() != nil {
				log.Println("market service stopped")
				return
			}

			log.Fatalf("market service failed: %v", err)
		}
	}()

	select {
	case <-marketService.Ready():
		log.Println("market structure is ready")

	case <-ctx.Done():
		return
	}

	go func() {
		if err := commandHandler.Run(ctx); err != nil {
			log.Printf("console stopped: %v", err)
			stop()
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("market service stopped")

	}

}
