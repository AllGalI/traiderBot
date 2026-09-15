package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	"traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	
)

func TestOrderRequest(t *testing.T) {
    edge := marketDomain.GraphEdge{
        Symbol: "BTC/USDT",
        // Direction: marketDomain.Buy, // если это поле есть в GraphEdge
    }

    request, err := domain.NewOrderRequest(
        edge,
        decimal.RequireFromString("100"),
        "client-123",
    )
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if request.Edge.Symbol != "BTC/USDT" {
        t.Errorf("unexpected symbol: %s", request.Edge.Symbol)
    }

    if !request.Amount.Equal(decimal.RequireFromString("100")) {
        t.Errorf("unexpected amount: %s", request.Amount)
    }

    if request.ClientOrderID != "client-123" {
        t.Errorf(
            "unexpected client order ID: %s",
            request.ClientOrderID,
        )
    }
}
func TestOrderResult(t *testing.T) {
	result := domain.OrderResult{
		Status:          domain.OrderStatusFilled,
		ExchangeOrderID: "exchange-456",
		ClientOrderID:   "client-123",
		ExecutedAmount:  decimal.RequireFromString("100"),
		ReceivedAmount:  decimal.RequireFromString("0.001"),
		Fee:             decimal.RequireFromString("0.1"),
		FeeAsset:        "USDT",
	}

	if result.Status != domain.OrderStatusFilled {
		t.Errorf("unexpected status: %s", result.Status)
	}

	if result.ExchangeOrderID != "exchange-456" {
		t.Errorf("unexpected exchange order ID: %s", result.ExchangeOrderID)
	}

	if !result.ExecutedAmount.Equal(decimal.RequireFromString("100")) {
		t.Errorf("unexpected executed amount: %s", result.ExecutedAmount)
	}

	if !result.ReceivedAmount.Equal(decimal.RequireFromString("0.001")) {
		t.Errorf("unexpected received amount: %s", result.ReceivedAmount)
	}

	if !result.Fee.Equal(decimal.RequireFromString("0.1")) {
		t.Errorf("unexpected fee: %s", result.Fee)
	}
}
