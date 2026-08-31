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

	executionApplication "traiderBot/internalDDD/execution/application"
	executionPaper "traiderBot/internalDDD/execution/infrastructure/paper"

	marketApplication "traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/infrastructure"
	marketBinance "traiderBot/internalDDD/market/infrastructure/binance"
	"traiderBot/internalDDD/market/infrastructure/cache"

	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
	simulationBinance "traiderBot/internalDDD/simulation/infrastructure/binance"

	"github.com/shopspring/decimal"
)

type executionOpportunityHandler struct {
	manager *executionApplication.ExecutionManager
}

func (h *executionOpportunityHandler) Handle(
	opportunity *simulationDomain.Opportunity,
) {
	if opportunity == nil {
		log.Println("execution: received nil opportunity")
		return
	}

	log.Printf(
		"OPPORTUNITY: route=%v profit=%s profitPct=%s fill=%s",
		opportunity.Route.Symbols(),
		opportunity.ProfitAbs,
		opportunity.ProfitPct,
		opportunity.FillRatio,
	)

	execution, err := h.manager.Execute(
		context.Background(),
		opportunity,
	)

	if err != nil {
		log.Printf(
			"EXECUTION ERROR: %v",
			err,
		)

		if execution != nil {
			log.Printf(
				"EXECUTION: id=%s status=%s orders=%d currentAmount=%s",
				execution.ID,
				execution.Status,
				len(execution.Orders),
				execution.CurrentAmount,
			)
		}

		return
	}

	log.Printf(
		"EXECUTION: id=%s status=%s orders=%d finalAmount=%s",
		execution.ID,
		execution.Status,
		len(execution.Orders),
		execution.FinalAmount,
	)

	for _, order := range execution.Orders {
		log.Printf(
			"EXECUTION ORDER: execution=%s step=%d symbol=%s direction=%v status=%s requested=%s executed=%s received=%s clientOrderID=%s",
			execution.ID,
			order.StepIndex,
			order.Symbol,
			order.Direction,
			order.Status,
			order.RequestedAmount,
			order.ExecutedAmount,
			order.ReceivedAmount,
			order.ClientOrderID,
		)
	}
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

	symbolProvider := marketBinance.NewSymbolProvider()

	marketStructureService := marketApplication.NewMarketStructureService(
		symbolProvider,
	)

	marketCache := cache.NewMarketCache("market_cache.json")

	// -------------------------------------------------------------------------
	// Market data.
	// -------------------------------------------------------------------------

	orderBookProvider := marketBinance.NewOrderBookProvider()
	orderBookStream := marketBinance.NewOrderBookStream()
	orderBookStore := infrastructure.NewOrderBookStore()

	orderBookDebugProvider := debugDomain.NewOrderBookProvider(
		orderBookStore,
	)

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
		log.Fatalf(
			"load freshness config: %v",
			err,
		)
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
	// Execution.
	// -------------------------------------------------------------------------

	paperExecutor := executionPaper.NewExecutor()

	executionPolicy := executionApplication.NewDefaultExecutionPolicy(
		decimal.RequireFromString("0.90"),
	)

	executionManager := executionApplication.NewExecutionManager(
		paperExecutor,
		nil,
		executionPolicy,
	)

	opportunityHandler := &executionOpportunityHandler{
		manager: executionManager,
	}

	simulationWorker := simulationApplication.NewWorker(
		marketStructureStore,
		marketDataService.Updates(),
		simulationService,
		opportunityHandler,
		startAmount,
	)

	simulationDebugProvider := debugDomain.NewSimulationProvider(
		simulationWorker.Metrics(),
	)

	// -------------------------------------------------------------------------
	// Debug / console.
	// -------------------------------------------------------------------------

	debugService := debugApplication.NewDebugService()

	graphProvider := debugDomain.NewGraphProvider(
		marketStructureStore,
	)

	debugService.Register(graphProvider)
	debugService.Register(orderBookDebugProvider)
	debugService.Register(simulationDebugProvider)

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

			log.Printf(
				"market service failed: %v",
				err,
			)

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

			log.Printf(
				"fee refresh worker failed: %v",
				err,
			)

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

			log.Printf(
				"simulation worker failed: %v",
				err,
			)

			stop()
		}
	}()

	// -------------------------------------------------------------------------
	// Console.
	// -------------------------------------------------------------------------

	go func() {
		if err := commandHandler.Run(ctx); err != nil {
			log.Printf(
				"console stopped: %v",
				err,
			)

			stop()
		}
	}()

	// -------------------------------------------------------------------------
	// Wait for shutdown.
	// -------------------------------------------------------------------------

	<-ctx.Done()

	log.Println("application stopped")
}
