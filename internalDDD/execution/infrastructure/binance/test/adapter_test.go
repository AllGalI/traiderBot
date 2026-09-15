package binance_test

// import (
// 	"context"
// 	"testing"

// 	"github.com/shopspring/decimal"

// 	executionDomain "traiderBot/internalDDD/execution/domain"
// 	marketDomain "traiderBot/internalDDD/market/domain"

// 	binance "traiderBot/internalDDD/execution/infrastructure/binance"
// )

// type fakeClient struct {
// 	executeResult executionDomain.OrderExecutionResult
// 	executeErr    error

// 	getOrderResult executionDomain.OrderExecutionResult
// 	getOrderErr    error

// 	lastRequest executionDomain.OrderRequest

// 	lastSymbol  string
// 	lastOrderID string
// }

// func (f *fakeClient) Execute(
// 	ctx context.Context,
// 	request executionDomain.OrderRequest,
// ) (executionDomain.OrderExecutionResult, error) {
// 	f.lastRequest = request

// 	return f.executeResult, f.executeErr
// }

// func (f *fakeClient) GetOrder(
// 	ctx context.Context,
// 	symbol string,
// 	orderID string,
// ) (executionDomain.OrderExecutionResult, error) {
// 	f.lastSymbol = symbol
// 	f.lastOrderID = orderID

// 	return f.getOrderResult, f.getOrderErr
// }

// func TestNewAdapterRejectsNilClient(t *testing.T) {
// 	_, err := binance.NewAdapter(nil)

// 	if err == nil {
// 		t.Fatal("expected error for nil client")
// 	}
// }

// func TestAdapterExecuteDelegatesToClient(t *testing.T) {
// 	expected := executionDomain.OrderExecutionResult{
// 		StepIndex: 0,
// 		Symbol:    "BTCUSDT",
// 		Direction: marketDomain.Buy,

// 		ClientOrderID:   "execution-1-order-0-2",
// 		ExchangeOrderID: "12345",

// 		RequestedAmount:   decimal.NewFromInt(100),
// 		ExpectedAmountOut: decimal.NewFromFloat(0.0018),
// 		ExecutedAmount:    decimal.NewFromInt(100),
// 		ReceivedAmount:    decimal.NewFromFloat(0.0018),

// 		Status: executionDomain.OrderFilled,
// 	}

// 	client := &fakeClient{
// 		executeResult: expected,
// 	}

// 	adapter, err := binance.NewAdapter(client)
// 	if err != nil {
// 		t.Fatalf("unexpected adapter error: %v", err)
// 	}

// 	request := executionDomain.OrderRequest{
// 		StepIndex: 0,

// 		Edge: marketDomain.GraphEdge{
// 			From:      "USDT",
// 			To:        "BTC",
// 			Symbol:    "BTCUSDT",
// 			Direction: marketDomain.Buy,
// 		},

// 		Amount:            decimal.NewFromInt(100),
// 		ExpectedAmountOut: decimal.NewFromFloat(0.0018),
// 		ClientOrderID:     "execution-1-order-0-2",
// 	}

// 	result, err := adapter.Execute(
// 		context.Background(),
// 		request,
// 	)

// 	if err != nil {
// 		t.Fatalf("unexpected execute error: %v", err)
// 	}

// 	if result != expected {
// 		t.Fatalf(
// 			"expected result %+v, got %+v",
// 			expected,
// 			result,
// 		)
// 	}

// 	if client.lastRequest != request {
// 		t.Fatalf(
// 			"expected request %+v, got %+v",
// 			request,
// 			client.lastRequest,
// 		)
// 	}
// }

// func TestAdapterGetOrderDelegatesToClient(t *testing.T) {
// 	expected := executionDomain.OrderExecutionResult{
// 		StepIndex:       1,
// 		Symbol:          "BTCUSDT",
// 		ClientOrderID:   "execution-1-order-1-3",
// 		ExchangeOrderID: "98765",
// 		RequestedAmount: decimal.NewFromFloat(0.0018),
// 		ReceivedAmount:  decimal.NewFromInt(101),
// 		Status:          executionDomain.OrderFilled,
// 	}

// 	client := &fakeClient{
// 		getOrderResult: expected,
// 	}

// 	adapter, err := binance.NewAdapter(client)
// 	if err != nil {
// 		t.Fatalf("unexpected adapter error: %v", err)
// 	}

// 	result, err := adapter.GetOrder(
// 		context.Background(),
// 		"BTCUSDT",
// 		"execution-1-order-1-3",
// 	)

// 	if err != nil {
// 		t.Fatalf("unexpected get order error: %v", err)
// 	}

// 	if result != expected {
// 		t.Fatalf(
// 			"expected result %+v, got %+v",
// 			expected,
// 			result,
// 		)
// 	}

// 	if client.lastSymbol != "BTCUSDT" {
// 		t.Fatalf(
// 			"expected symbol BTCUSDT, got %s",
// 			client.lastSymbol,
// 		)
// 	}

// 	if client.lastOrderID != "execution-1-order-1-3" {
// 		t.Fatalf(
// 			"expected order ID execution-1-order-1-3, got %s",
// 			client.lastOrderID,
// 		)
// 	}
// }
