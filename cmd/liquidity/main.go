package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"

	exchangeDomain "traiderBot/internalDDD/exchange/domain"
	exchangeBinance "traiderBot/internalDDD/exchange/infrastructure/binance"
	binanceRest "traiderBot/internalDDD/exchange/infrastructure/binance/rest"
	binanceWS "traiderBot/internalDDD/exchange/infrastructure/binance/ws"

	balanceApplication "traiderBot/internalDDD/balance/application"
	balanceBinance "traiderBot/internalDDD/balance/infrastructure"

	marketApplication "traiderBot/internalDDD/market/application"
	marketDomain "traiderBot/internalDDD/market/domain"
	marketInfrastructure "traiderBot/internalDDD/market/infrastructure"
	marketBinance "traiderBot/internalDDD/market/infrastructure/binance"
	"traiderBot/internalDDD/market/infrastructure/cache"

	recoveryApplication "traiderBot/internalDDD/recovery/application"

	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationBinance "traiderBot/internalDDD/simulation/infrastructure/binance"

	recoveryFile "traiderBot/internalDDD/recovery/infrastructure/file"
)

const (
	depth        = 10
	symbolsLimit = 30
	dataDir      = "data"

	targetAsset = "USDT"
)

var minLiquidityFillRatio = decimal.RequireFromString("0.60")

func main() {
	dryRun := flag.Bool(
		"dry-run",
		false,
		"build conversion plan without executing orders",
	)

	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded")
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("========================================")
	log.Println("Liquidity converter")
	log.Println("Target:", targetAsset)
	log.Println("Testnet: enabled")
	log.Printf("Dry run: %v", *dryRun)
	log.Println("========================================")

	// -------------------------------------------------------------------------
	// Binance.
	// -------------------------------------------------------------------------

	cfg, err := exchangeBinance.LoadConfig(true)
	if err != nil {
		log.Fatalf("load binance config: %v", err)
	}

	httpClient := exchangeBinance.NewHTTPClient()

	binanceRestClient, err := binanceRest.NewClient(
		cfg,
		httpClient,
	)
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

	if err := balanceService.Refresh(ctx); err != nil {
		log.Fatalf(
			"refresh account balances: %v",
			err,
		)
	}

	log.Println()
	log.Println("Current balances:")

	for _, balance := range balanceService.Snapshot() {
		if balance.Available.LessThanOrEqual(decimal.Zero) {
			continue
		}

		log.Printf(
			"  %-8s available=%s",
			balance.Asset,
			balance.Available,
		)
	}

	// -------------------------------------------------------------------------
	// Market structure.
	// -------------------------------------------------------------------------

	symbolProvider := marketBinance.NewSymbolProvider(
		binanceRestClient,
	)

	marketStructureService := marketApplication.NewMarketStructureService(
		symbolProvider,
		symbolsLimit,
	)

	marketCache := cache.NewMarketCache(
		"market_cache.json",
	)

	// -------------------------------------------------------------------------
	// Market data.
	// -------------------------------------------------------------------------

	orderBookProvider := marketBinance.NewOrderBookProvider()

	orderBookStream := marketBinance.NewOrderBookStream(
		wsClient,
	)

	orderBookStore := marketInfrastructure.NewOrderBookStore()

	marketDataService := marketApplication.NewMarketDataService(
		orderBookProvider,
		orderBookStream,
		orderBookStore,
	)

	marketStructureStore :=
		marketInfrastructure.NewInMemoryMarketStructureStore()

	marketService := marketApplication.NewMarketService(
		marketCache,
		marketStructureService,
		marketDataService,
		marketStructureStore,
	)

	// -------------------------------------------------------------------------
	// Fees.
	// -------------------------------------------------------------------------

	feeLoader := simulationBinance.NewFeeProvider(
		binanceRestClient,
	)

	feeCache := simulationApplication.NewFeeCache()

	feeRefreshWorker := simulationApplication.NewFeeRefreshWorker(
		feeLoader,
		feeCache,
		marketStructureStore,
		10*time.Minute,
	)

	// -------------------------------------------------------------------------
	// Start market service.
	// -------------------------------------------------------------------------

	startAssets := map[string]struct{}{
		targetAsset: {},
	}

	log.Println()
	log.Println("Starting market service...")

	go func() {
		if err := marketService.Start(
			ctx,
			startAssets,
			depth,
		); err != nil {
			if ctx.Err() != nil {
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
		log.Println("Market structure is ready")

	case <-ctx.Done():
		return
	}

	// -------------------------------------------------------------------------
	// Refresh fees once before building the conversion plan.
	// -------------------------------------------------------------------------

	log.Println("Loading trading fees...")

	if err := feeRefreshWorker.Refresh(ctx); err != nil {
		log.Fatalf(
			"refresh trading fees: %v",
			err,
		)
	}

	log.Println("Trading fees loaded")

	// -------------------------------------------------------------------------
	// Build Recovery.
	// -------------------------------------------------------------------------

	marketStructure, ok := marketStructureStore.Get()
	if !ok || marketStructure == nil {
		log.Fatal(
			"market structure is not available",
		)
	}

	recoveryRouteFinder := recoveryApplication.NewRouteFinder(
		marketStructure.Graph,
	)

	recoverySimulator := recoveryApplication.NewSimulator(
		minLiquidityFillRatio,
	)

	recoveryPlanner := recoveryApplication.NewPlanner(
		recoveryRouteFinder,
		recoverySimulator,
	)

	recoveryExecutor := recoveryApplication.NewExecutor(
		binanceRestClient,
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

	converter := recoveryApplication.NewConverter(
		recoveryPlanner,
		recoveryService,
	)

	// -------------------------------------------------------------------------
	// Build MarketContext.
	// -------------------------------------------------------------------------

	marketContext, err := buildMarketContext(
		marketStructure,
		orderBookStore,
		feeCache,
	)
	if err != nil {
		log.Fatalf(
			"build market context: %v",
			err,
		)
	}

	// -------------------------------------------------------------------------
	// Build conversion state from actual available balances.
	// -------------------------------------------------------------------------

	state := make(
		map[string]decimal.Decimal,
	)

	for _, balance := range balanceService.Snapshot() {
		if balance.Available.LessThanOrEqual(decimal.Zero) {
			continue
		}

		state[balance.Asset] = balance.Available
	}

	// -------------------------------------------------------------------------
	// Preview.
	// -------------------------------------------------------------------------

	preview, err := converter.Preview(
		ctx,
		state,
		targetAsset,
		marketContext,
	)
	if err != nil {
		log.Fatalf(
			"build conversion preview: %v",
			err,
		)
	}

	printPreview(preview)

	if *dryRun {
		log.Println()
		log.Println("DRY RUN: no orders were executed")
		return
	}

	if len(preview.Plans) == 0 {
		log.Println()
		log.Println("Nothing to convert.")
		return
	}

	// -------------------------------------------------------------------------
	// Execute.
	// -------------------------------------------------------------------------

	log.Println()
	log.Println("Starting conversion...")

	finalState, err := converter.Execute(
		ctx,
		state,
		targetAsset,
		marketContext,
	)
	if err != nil {
		log.Fatalf(
			"conversion failed: %v",
			err,
		)
	}

	log.Println()
	log.Println("Conversion finished.")

	printState(
		"Recovery final state:",
		finalState,
	)

	// -------------------------------------------------------------------------
	// Refresh actual exchange balances.
	// -------------------------------------------------------------------------

	log.Println()
	log.Println("Refreshing exchange balances...")

	if err := balanceService.Refresh(ctx); err != nil {
		log.Printf(
			"refresh final account balances: %v",
			err,
		)
		return
	}

	printBalances(balanceService)
}

func buildMarketContext(
	marketStructure *marketDomain.MarketStructure,
	orderBookStore *marketInfrastructure.OrderBookStore,
	feeCache *simulationApplication.FeeCache,
) (
	recoveryApplication.MarketContext,
	error,
) {
	if marketStructure == nil {
		return recoveryApplication.MarketContext{},
			fmt.Errorf("market structure is nil")
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

	for _, symbol := range orderBookStore.Symbols() {
		orderBook, ok := orderBookStore.Get(symbol)
		if !ok {
			continue
		}

		orderBooks[symbol] = orderBook
	}

	fees := make(
		map[string]marketDomain.FeeRate,
	)

	for symbol := range symbols {
		fee, ok := feeCache.Get(
			context.Background(),
			symbol,
		)
		if !ok {
			log.Printf(
				"warning: fee is not available for %s",
				symbol,
			)
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

func printPreview(
	preview recoveryApplication.ConversionPreview,
) {
	log.Println()
	log.Println("========================================")
	log.Println("Conversion preview")
	log.Println("========================================")
	log.Println()

	log.Printf(
		"Target asset: %s",
		preview.TargetAsset,
	)

	log.Println()
	log.Println("Plans:")

	if len(preview.Plans) == 0 {
		log.Println("  no executable conversion plans")
		return
	}

	for _, plan := range preview.Plans {
		log.Printf(
			"  %-8s amount=%s -> expected=%s route=%v",
			plan.Asset,
			plan.InitialAmount,
			plan.SimulatedAmount,
			plan.Route.Symbols(),
		)
	}

	log.Println()
}

func printState(
	title string,
	state map[string]decimal.Decimal,
) {
	log.Println(title)

	for asset, amount := range state {
		if amount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		log.Printf(
			"  %-8s %s",
			asset,
			amount,
		)
	}
}

func printBalances(
	balanceService *balanceApplication.BalanceService,
) {
	log.Println("Actual exchange balances:")

	for _, balance := range balanceService.Snapshot() {
		if balance.Total.IsZero() {
			continue
		}

		log.Printf(
			"  %-8s total=%s available=%s locked=%s reserved=%s",
			balance.Asset,
			balance.Total,
			balance.Available,
			balance.ExchangeLocked,
			balance.Reserved,
		)
	}
}
