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
	exchangeBinance "traiderBot/internalDDD/exchange/infrastructure/binance"
	binanceRest "traiderBot/internalDDD/exchange/infrastructure/binance/rest"
	binanceWS "traiderBot/internalDDD/exchange/infrastructure/binance/ws"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionManager "traiderBot/internalDDD/execution/application/execution_manager"
	executionDomain "traiderBot/internalDDD/execution/domain"

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
	simulationFile "traiderBot/internalDDD/simulation/infrastructure/file"

	balanceApplication "traiderBot/internalDDD/balance/application"
	balanceBinance "traiderBot/internalDDD/balance/infrastructure"

	"github.com/shopspring/decimal"
)

type executionOpportunityHandler struct {
	manager              *executionManager.ExecutionManager
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
		depth = 10
		// symbolsLimit = 150
		symbolsLimit = 30
		dataDir      = "data"
	)

	var (
		// minLiquidityFillRatio = decimal.RequireFromString("0.60")
		minLiquidityFillRatio = decimal.RequireFromString("0.90")
		minProfitPct          = decimal.RequireFromString("0.0008")
		// minProfitPct = decimal.RequireFromString("-0.001")
		startAmount = decimal.RequireFromString("100")
		// startAmount  = decimal.RequireFromString("30")
		maxDeviation = decimal.RequireFromString("0.003")
	)

	// -------------------------------------------------------------------------
	// Binance Config and Clients
	// -------------------------------------------------------------------------
	cfg, err := exchangeBinance.LoadConfig(true)
	if err != nil {
		log.Fatalf("load binance config: %v", err)
	}

	httpClient := exchangeBinance.NewHTTPClient()

	binanceRestClient, err := binanceRest.NewClient(cfg, httpClient)
	if err != nil {
		log.Fatalf("create binance REST client: %v", err)
	}

	wsClient := binanceWS.NewClient(cfg)

	orderIDGenerator := exchangeDomain.NewOrderIDGenerator()

	// -------------------------------------------------------------------------
	// Balance.
	// -------------------------------------------------------------------------

	balanceProvider := balanceBinance.NewBinanceBalanceProvider(
		binanceRestClient,
	)

	balanceService := balanceApplication.NewBalanceService(
		balanceProvider,
	)

	balanceService.StartAutoRefresh(
		ctx,
		30*time.Second,
	)

	if err := balanceService.Refresh(ctx); err != nil {
		log.Fatalf(
			"refresh account balances: %v",
			err,
		)
	}

	log.Printf(
		"account balance loaded: USDT available=%s",
		balanceService.Available("USDT"),
	)

	// -------------------------------------------------------------------------
	// Market structure.
	// -------------------------------------------------------------------------

	symbolProvider := marketBinance.NewSymbolProvider(binanceRestClient)

	marketStructureService := marketApplication.NewMarketStructureService(
		symbolProvider,
		symbolsLimit,
	)

	marketCache := cache.NewMarketCache("market_cache.json")

	// -------------------------------------------------------------------------
	// Market data.
	// -------------------------------------------------------------------------

	orderBookProvider := marketBinance.NewOrderBookProvider()
	orderBookStream := marketBinance.NewOrderBookStream(wsClient)
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
		"freshness.json",
	)
	if err != nil {
		log.Fatalf(
			"load freshness config: %v",
			err,
		)
	}

	freshnessPolicy := freshnessConfig.NewPolicy()

	feeLoader := simulationBinance.NewFeeProvider(binanceRestClient)

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
	// Exchange
	// -------------------------------------------------------------------------

	// paperExecutor := executionPaper.NewExecutor(
	// 	orderBookStore,
	// 	symbolReader,
	// 	feeCache,
	// 	minLiquidityFillRatio,
	// )

	// if err != nil {
	// 	log.Fatal(err)
	// }

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

	executionManager := executionManager.NewExecutionManager(
		binanceRestClient,
		resultMapper,
		evaluator,
		orderIDGenerator,
		executionStore,
		balanceService,
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

	// // -------------------------------------------------------------------------
	// // Tg bot
	// // -------------------------------------------------------------------------
	// telegramBot, err := telegramInfrastructure.NewBot(
	// 	balanceService,
	// 	executionStore,
	// 	opportunityStore,
	// )
	// if err != nil {
	// 	log.Fatalf(
	// 		"create telegram bot: %v",
	// 		err,
	// 	)
	// }

	// -------------------------------------------------------------------------
	// Debug / console.
	// -------------------------------------------------------------------------

	debugService := debugApplication.NewDebugService()

	graphProvider := debugDomain.NewGraphProvider(
		marketStructureStore,
	)

	balanceDebugProvider := debugDomain.NewBalanceProvider(balanceService)
	debugService.Register(balanceDebugProvider)

	debugService.Register(graphProvider)
	debugService.Register(orderBookDebugProvider)
	debugService.Register(simulationDebugProvider)
	debugService.Register(opportunityDebugProvider)
	debugService.Register(executionDebugProvider)

	consoleHandler := console.NewHandler(
		debugService,
		os.Stdout,
	)

	commandHandler := console.NewCommandHandler(
		consoleHandler,
		os.Stdin,
		os.Stdout,
	)

	// debugMux := http.NewServeMux()

	// debugHandler := debugHTTP.NewHandler(debugService)
	// debugHandler.RegisterRoutes(debugMux)

	// debugServer := debugHTTP.NewServer(
	// 	"127.0.0.1:8080",
	// 	debugMux,
	// )

	// go func() {
	// 	if err := debugServer.Run(ctx); err != nil {
	// 		log.Printf(
	// 			"debug HTTP server stopped: %v",
	// 			err,
	// 		)
	// 	}
	// }()

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
	// tg
	// -------------------------------------------------------------------------

	// go func() {
	// 	if err := telegramBot.Run(ctx); err != nil {
	// 		if ctx.Err() != nil {
	// 			log.Println("telegram bot stopped")
	// 			return
	// 		}

	// 		log.Printf(
	// 			"telegram bot failed: %v",
	// 			err,
	// 		)

	// 		stop()
	// 	}
	// }()

	// -------------------------------------------------------------------------
	// Wait for shutdown.
	// -------------------------------------------------------------------------

	<-ctx.Done()

	log.Println("application stopped")
}
