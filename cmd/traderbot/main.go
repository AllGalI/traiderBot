package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	debugApplication "traiderBot/internalDDD/debug/application"
	debugDomain "traiderBot/internalDDD/debug/domain"
	"traiderBot/internalDDD/debug/infrastructure/console"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	executionPaper "traiderBot/internalDDD/execution/infrastructure/paper"

	marketApplication "traiderBot/internalDDD/market/application"
	marketDomain "traiderBot/internalDDD/market/domain"
	"traiderBot/internalDDD/market/infrastructure"
	marketBinance "traiderBot/internalDDD/market/infrastructure/binance"
	"traiderBot/internalDDD/market/infrastructure/cache"

	recoveryApplication "traiderBot/internalDDD/recovery/application"

	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
	simulationBinance "traiderBot/internalDDD/simulation/infrastructure/binance"

	executionFile "traiderBot/internalDDD/execution/infrastructure/file"
	recoveryFile "traiderBot/internalDDD/recovery/infrastructure/file"
	simulationFile "traiderBot/internalDDD/simulation/infrastructure/file"

	"github.com/shopspring/decimal"
)

type executionOpportunityHandler struct {
	manager              *executionApplication.ExecutionManager
	marketStructureStore marketApplication.MarketStructureStore
	orderBookStore       *infrastructure.OrderBookStore
	feeCache             *simulationApplication.FeeCache
}

func (h *executionOpportunityHandler) Handle(
	opportunity *simulationDomain.Opportunity,
) {
	if opportunity == nil {
		log.Println("execution: received nil opportunity")
		return
	}

	// log.Printf(
	// 	"OPPORTUNITY: route=%v profit=%s profitPct=%s fill=%s",
	// 	opportunity.Route.Symbols(),
	// 	opportunity.ProfitAbs,
	// 	opportunity.ProfitPct,
	// 	opportunity.FillRatio,
	// )

	marketContext, err := h.buildMarketContext()
	if err != nil {
		log.Printf(
			"EXECUTION ERROR: build market context: %v",
			err,
		)
		return
	}

	// execution, actualState, err := h.manager.Execute(
	execution, _, err := h.manager.Execute(
		context.Background(),
		*opportunity,
		marketContext,
	)

	if err != nil {
		// log.Printf(
		// 	"EXECUTION ERROR: %v",
		// 	err,
		// )

		if execution != nil {
			// log.Printf(
			// 	"EXECUTION: state=%s actualStepResults=%d",
			// 	execution.State(),
			// 	len(execution.ActualStepResults()),
			// )
		}

		// log.Printf(
		// 	"EXECUTION STATE: %+v",
		// 	actualState,
		// )

		return
	}

	// log.Printf(
	// 	"EXECUTION: state=%s actualStepResults=%d",
	// 	execution.State(),
	// 	len(execution.ActualStepResults()),
	// )

	// log.Printf(
	// 	"EXECUTION STATE: %+v",
	// 	actualState,
	// )
}
func (h *executionOpportunityHandler) buildMarketContext() (
	recoveryApplication.MarketContext,
	error,
) {
	marketStructure, ok := h.marketStructureStore.Get()
	if !ok || marketStructure == nil {
		return recoveryApplication.MarketContext{},
			fmt.Errorf("market structure is not available")
	}

	symbols := make(
		map[string]marketDomain.Symbol,
		len(marketStructure.Symbols),
	)

	for _, symbol := range marketStructure.Symbols {
		symbols[symbol.Name] = symbol
	}

	orderBooks := make(
		map[string]marketDomain.OrderBook,
	)

	for _, symbol := range h.orderBookStore.Symbols() {
		orderBook, ok := h.orderBookStore.Get(symbol)
		if !ok {
			continue
		}

		orderBooks[symbol] = orderBook
	}

	fees := make(
		map[string]marketDomain.FeeRate,
	)

	for symbol := range symbols {
		fee, ok := h.feeCache.Get(
			context.Background(),
			symbol,
		)
		if !ok {
			continue
		}

		fees[symbol] = marketDomain.FeeRate{
			Maker: fee.Rate,
			Taker: fee.Rate,
		}
	}

	return recoveryApplication.MarketContext{
		Symbols:    symbols,
		OrderBooks: orderBooks,
		Fees:       fees,
	}, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded")
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	const (
		depth        = 10
		symbolsLimit = 30
		dataDir      = "data"
	)

	var (
		minLiquidityFillRatio = decimal.RequireFromString("0.90")
		// minProfitPct          = decimal.RequireFromString("0.001")
		minProfitPct = decimal.RequireFromString("-0.003")
		startAmount  = decimal.RequireFromString("100")
		maxDeviation = decimal.RequireFromString("0.003")
	)

	// -------------------------------------------------------------------------
	// Market structure.
	// -------------------------------------------------------------------------

	symbolProvider := marketBinance.NewSymbolProvider()

	marketStructureService := marketApplication.NewMarketStructureService(
		symbolProvider,
		symbolsLimit,
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
	// Start market service.
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
	// Recovery.
	// -------------------------------------------------------------------------

	marketStructure, ok := marketStructureStore.Get()
	if !ok || marketStructure == nil {
		log.Fatal("market structure is not available after market service became ready")
	}

	orderIDGenerator := exchangeDomain.NewOrderIDGenerator()

	paperExecutor := executionPaper.NewExecutor(
		orderBookStore,
		symbolReader,
		feeCache,
		minLiquidityFillRatio,
	)

	recoveryRouteFinder := recoveryApplication.NewRouteFinder(
		marketStructure.Graph,
	)

	recoverySimulator := recoveryApplication.NewSimulator(minLiquidityFillRatio)

	recoveryPlanner := recoveryApplication.NewPlanner(
		recoveryRouteFinder,
		recoverySimulator,
	)

	recoveryExecutor := recoveryApplication.NewExecutor(
		paperExecutor,
		orderIDGenerator,
	)

	recoveryStore, err := recoveryFile.NewRecoveryStore(
		dataDir,
		1000,
	)

	if err != nil {
		log.Fatalf(
			"create recovery store: %v",
			err,
		)
	}

	recoveryService := recoveryApplication.NewRecoveryService(
		recoveryPlanner,
		recoveryExecutor,
		recoveryStore,
	)

	recoveryDebugProvider :=
		debugDomain.NewRecoveryProvider(
			recoveryStore,
		)
		// -------------------------------------------------------------------------
		// Execution.
		// -------------------------------------------------------------------------

	executionStore, err := executionFile.NewExecutionStore(
		dataDir,
		1000,
	)
	if err != nil {
		log.Fatalf(
			"create execution store: %v",
			err,
		)
	}
	executionDebugProvider :=
		debugDomain.NewExecutionProvider(
			executionStore,
		)

	evaluator := executionApplication.NewEvaluator(
		maxDeviation,
	)

	resultMapper := executionDomain.NewActualStepResultMapper()

	executionManager := executionApplication.NewExecutionManager(
		paperExecutor,
		resultMapper,
		evaluator,
		orderIDGenerator,
		recoveryService,
		executionStore,
	)

	opportunityHandler := &executionOpportunityHandler{
		manager:              executionManager,
		marketStructureStore: marketStructureStore,
		orderBookStore:       orderBookStore,
		feeCache:             feeCache,
	}

	// -------------------------------------------------------------------------
	// Simulation worker.
	// -------------------------------------------------------------------------

	opportunityStore, err := simulationFile.NewOpportunityStore(
		dataDir,
		1000,
	)
	if err != nil {
		log.Fatalf(
			"create opportunity store: %v",
			err,
		)
	}
	opportunityDebugProvider :=
		debugDomain.NewOpportunityProvider(
			opportunityStore,
		)

	simulationWorker := simulationApplication.NewWorker(
		marketStructureStore,
		marketDataService.Updates(),
		simulationService,
		opportunityHandler,
		opportunityStore,
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
	debugService.Register(opportunityDebugProvider)
	debugService.Register(executionDebugProvider)
	debugService.Register(recoveryDebugProvider)

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
