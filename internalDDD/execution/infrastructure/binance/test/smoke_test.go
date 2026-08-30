
package binance_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	binance "traiderBot/internalDDD/execution/infrastructure/binance"
	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
)

func TestBinanceTestnetMarketOrder(t *testing.T) {
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

	clientOrderID := fmt.Sprintf(
		"smoke-test-%d",
		time.Now().UnixNano(),
	)

	request := executionDomain.OrderRequest{
		StepIndex: 0,

		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},

		// Маленькая тестовая сумма.
		//
		// Если Binance Testnet потребует другой minimum notional,
		// увеличим это значение после первого запуска.
		Amount: decimal.NewFromInt(10),

		ClientOrderID: clientOrderID,
	}

	t.Logf(
		"placing Binance Testnet order: symbol=%s side=BUY amount=%s clientOrderID=%s",
		request.Edge.Symbol,
		request.Amount,
		clientOrderID,
	)

	result, err := client.Execute(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf(
			"execute Binance Testnet order: %v",
			err,
		)
	}

	t.Logf(
		"order executed: exchangeOrderID=%s clientOrderID=%s status=%s executed=%s received=%s",
		result.ExchangeOrderID,
		result.ClientOrderID,
		result.Status,
		result.ExecutedAmount,
		result.ReceivedAmount,
	)

	if result.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			result.Symbol,
		)
	}

	if result.Direction != marketDomain.Buy {
		t.Fatalf(
			"expected BUY direction, got %v",
			result.Direction,
		)
	}

	if result.ClientOrderID != clientOrderID {
		t.Fatalf(
			"expected client order ID %q, got %q",
			clientOrderID,
			result.ClientOrderID,
		)
	}

	if result.ExchangeOrderID == "" {
		t.Fatal("expected exchange order ID")
	}

	if result.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected order to be filled, got %s",
			result.Status,
		)
	}

	if result.ExecutedAmount.LessThanOrEqual(decimal.Zero) {
		t.Fatalf(
			"expected positive executed amount, got %s",
			result.ExecutedAmount,
		)
	}

	if result.ReceivedAmount.LessThanOrEqual(decimal.Zero) {
		t.Fatalf(
			"expected positive received amount, got %s",
			result.ReceivedAmount,
		)
	}

	// Проверяем reconciliation через GET /api/v3/order.
	reconciled, err := client.GetOrder(
		context.Background(),
		"BTCUSDT",
		clientOrderID,
	)
	if err != nil {
		t.Fatalf(
			"reconcile Binance Testnet order: %v",
			err,
		)
	}

	t.Logf(
		"order reconciled: exchangeOrderID=%s clientOrderID=%s status=%s executed=%s received=%s",
		reconciled.ExchangeOrderID,
		reconciled.ClientOrderID,
		reconciled.Status,
		reconciled.ExecutedAmount,
		reconciled.ReceivedAmount,
	)

	if reconciled.ExchangeOrderID != result.ExchangeOrderID {
		t.Fatalf(
			"exchange order ID changed: execute=%s reconcile=%s",
			result.ExchangeOrderID,
			reconciled.ExchangeOrderID,
		)
	}

	if reconciled.ClientOrderID != result.ClientOrderID {
		t.Fatalf(
			"client order ID changed: execute=%s reconcile=%s",
			result.ClientOrderID,
			reconciled.ClientOrderID,
		)
	}

	if reconciled.Status != result.Status {
		t.Fatalf(
			"order status changed: execute=%s reconcile=%s",
			result.Status,
			reconciled.Status,
		)
	}
}
