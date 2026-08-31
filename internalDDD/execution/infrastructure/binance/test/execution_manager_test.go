package binance_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	binance "traiderBot/internalDDD/execution/infrastructure/binance"
	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"

	"github.com/shopspring/decimal"
)

func TestExecutionManagerBinanceTestnet(t *testing.T) {
	binance.LoadConfig(true)

	if os.Getenv("RUN_BINANCE_TESTNET") != "1" {
		t.Skip("set RUN_BINANCE_TESTNET=1 to run Binance Testnet integration test")
	}

	cfg, err := binance.LoadConfig(true)
	if err != nil {
		t.Fatalf("load Binance Testnet config: %v", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	client, err := binance.NewClient(
		cfg,
		httpClient,
	)
	if err != nil {
		t.Fatalf("create Binance client: %v", err)
	}

	adapter, err := binance.NewAdapter(client)
	if err != nil {
		t.Fatalf("create Binance adapter: %v", err)
	}

	policy := executionApplication.NewDefaultExecutionPolicy(
		decimal.NewFromFloat(0.90),
	)

	manager := executionApplication.NewExecutionManager(
		adapter,
		adapter,
		policy,
	)

	opportunity := &simulationDomain.Opportunity{
		Route: marketDomain.Route{
			Edges: []marketDomain.GraphEdge{
				{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: marketDomain.Buy,
				},
			},
		},

		Steps: []simulationDomain.StepResult{
			{
				Edge: marketDomain.GraphEdge{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: marketDomain.Buy,
				},

				AmountIn:  decimal.NewFromInt(10),
				AmountOut: decimal.NewFromFloat(0.00019),
			},
		},

		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(10),
		ExecutedStartAmount:  decimal.NewFromInt(10),

		EndAmount: decimal.NewFromFloat(0.00019),

		FillRatio: decimal.NewFromInt(1),

		ProfitAbs: decimal.Zero,
		ProfitPct: decimal.Zero,

		Timestamp: time.Now(),
	}

	t.Logf(
		"starting ExecutionManager Testnet execution: amount=%s symbol=%s",
		opportunity.RequestedStartAmount,
		opportunity.Steps[0].Edge.Symbol,
	)

	execution, err := manager.Execute(
		context.Background(),
		opportunity,
	)
	if err != nil {
		t.Fatalf(
			"execute opportunity through ExecutionManager: %v",
			err,
		)
	}

	if execution == nil {
		t.Fatal("expected execution, got nil")
	}

	t.Logf(
		"execution finished: id=%s status=%s currentAmount=%s orders=%d",
		execution.ID,
		execution.Status,
		execution.CurrentAmount,
		len(execution.Orders),
	)

	if execution.Status != executionDomain.ExecutionCompleted {
		t.Fatalf(
			"expected execution status %s, got %s",
			executionDomain.ExecutionCompleted,
			execution.Status,
		)
	}

	if len(execution.Orders) != 1 {
		t.Fatalf(
			"expected 1 executed order, got %d",
			len(execution.Orders),
		)
	}

	order := execution.Orders[0]

	if order.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected BTCUSDT, got %s",
			order.Symbol,
		)
	}

	if order.Direction != marketDomain.Buy {
		t.Fatalf(
			"expected BUY, got %v",
			order.Direction,
		)
	}

	if order.ClientOrderID == "" {
		t.Fatal("expected client order ID")
	}

	if order.ExchangeOrderID == "" {
		t.Fatal("expected exchange order ID")
	}

	if order.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected order status %s, got %s",
			executionDomain.OrderFilled,
			order.Status,
		)
	}

	if order.ExecutedAmount.LessThanOrEqual(decimal.Zero) {
		t.Fatalf(
			"expected positive executed amount, got %s",
			order.ExecutedAmount,
		)
	}

	if order.ReceivedAmount.LessThanOrEqual(decimal.Zero) {
		t.Fatalf(
			"expected positive received amount, got %s",
			order.ReceivedAmount,
		)
	}

	t.Logf(
		"order: exchangeOrderID=%s clientOrderID=%s status=%s requested=%s executed=%s received=%s fee=%s feeAsset=%s",
		order.ExchangeOrderID,
		order.ClientOrderID,
		order.Status,
		order.RequestedAmount,
		order.ExecutedAmount,
		order.ReceivedAmount,
		order.Fee,
		order.FeeAsset,
	)
}