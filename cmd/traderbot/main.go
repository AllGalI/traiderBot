package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	debugApplication "traiderBot/internalDDD/debug/application"
	debugDomain "traiderBot/internalDDD/debug/domain"
	"traiderBot/internalDDD/debug/infrastructure/console"
	marketApplication "traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/infrastructure"
	"traiderBot/internalDDD/market/infrastructure/binance"
	"traiderBot/internalDDD/market/infrastructure/cache"

	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
	simulationBinance "traiderBot/internalDDD/simulation/infrastructure/binance"

	"github.com/shopspring/decimal"
)

type logOpportunityHandler struct{}

func (h *logOpportunityHandler) Handle(
	opportunity *simulationDomain.Opportunity,
) {
	log.Printf(
		"OPPORTUNITY: route=%v profit=%s profitPct=%s fill=%s",
		opportunity.Route.Symbols(),
		opportunity.ProfitAbs,
		opportunity.ProfitPct,
		opportunity.FillRatio,
	)
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	const (
		depth        = 10
		symbolsLimit = 10
	)

	var (
		minLiquidityFillRatio = decimal.RequireFromString("0.90")
		minProfitPct          = decimal.RequireFromString("0.001")
		startAmount           = decimal.RequireFromString("100")
	)

	// -------------------------------------------------------------------------
	// Market structure.
	// -------------------------------------------------------------------------

	symbolProvider := binance.NewSymbolProvider()

	marketStructureService := marketApplication.NewMarketStructureService(
		symbolProvider,
	)

	marketCache := cache.NewMarketCache("market_cache.json")

	// -------------------------------------------------------------------------
	// Market data.
	// -------------------------------------------------------------------------

	orderBookProvider := binance.NewOrderBookProvider()
	orderBookStream := binance.NewOrderBookStream()
	orderBookStore := infrastructure.NewOrderBookStore()

	marketDataService := marketApplication.NewMarketDataService(
		orderBookProvider,
		orderBookStream,
		orderBookStore,
	)

	marketStructureStore := infrastructure.NewInMemoryMarketStructureStore()

	marketService := marketApplication.NewMarketService(
		marketCache,
		marketStructureService,
		marketDataService,
		marketStructureStore,
	)

	// -------------------------------------------------------------------------
	// Simulation.
	// -------------------------------------------------------------------------

	freshnessConfig, err := simulationApplication.LoadFreshnessConfig(
		"internalDDD/simulation/config/freshness.json",
	)
	if err != nil {
		log.Fatalf("load freshness config: %v", err)
	}

	freshnessPolicy := freshnessConfig.NewPolicy()

	feeLoader := simulationBinance.NewFeeProvider()

	feeCache := simulationApplication.NewFeeCache()

	feeRefreshWorker := simulationApplication.NewFeeRefreshWorker(
		feeLoader,
		feeCache,
		marketStructureStore,
		10*time.Minute,
	)

	symbolReader := simulationApplication.NewSymbolReader(
		marketStructureStore,
	)

	simulationService := simulationApplication.NewSimulationService(
		orderBookStore,
		feeCache,
		symbolReader,
		freshnessPolicy,
		minLiquidityFillRatio,
		minProfitPct,
	)

	// -------------------------------------------------------------------------
	// Opportunity handling.
	// -------------------------------------------------------------------------

	// Пока используем простой handler.
	//
	// Позже сюда можно поставить нормальный pipeline:
	// Opportunity -> validation -> execution / notification / storage.
	opportunityHandler := &logOpportunityHandler{}

	simulationWorker := simulationApplication.NewWorker(
		marketStructureStore,
		marketDataService.Updates(),
		simulationService,
		opportunityHandler,
		startAmount,
	)

	// -------------------------------------------------------------------------
	// Debug / console.
	// -------------------------------------------------------------------------

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

	// -------------------------------------------------------------------------
	// Start.
	// -------------------------------------------------------------------------

	startAssets := map[string]struct{}{
		"USDT": {},
	}

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

			log.Printf("market service failed: %v", err)
			stop()
		}
	}()

	select {
	case <-marketService.Ready():
		log.Println("market structure is ready")

	case <-ctx.Done():
		return
	}

	// -------------------------------------------------------------------------
	// Fee refresh worker.
	// -------------------------------------------------------------------------

	go func() {
		if err := feeRefreshWorker.Run(ctx); err != nil {
			if ctx.Err() != nil {
				log.Println("fee refresh worker stopped")
				return
			}

			log.Printf("fee refresh worker failed: %v", err)
			stop()
		}
	}()

	// -------------------------------------------------------------------------
	// Simulation worker.
	// -------------------------------------------------------------------------

	go func() {
		if err := simulationWorker.Run(ctx); err != nil {
			if ctx.Err() != nil {
				log.Println("simulation worker stopped")
				return
			}

			log.Printf("simulation worker failed: %v", err)
			stop()
		}
	}()

	// -------------------------------------------------------------------------
	// Console.
	// -------------------------------------------------------------------------

	go func() {
		if err := commandHandler.Run(ctx); err != nil {
			log.Printf("console stopped: %v", err)
			stop()
		}
	}()

	// -------------------------------------------------------------------------
	// Wait for shutdown.
	// -------------------------------------------------------------------------

	<-ctx.Done()

	log.Println("application stopped")
}
