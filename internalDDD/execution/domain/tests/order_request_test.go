package tests

import (
	"testing"

	"traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

func TestNewOrderRequest(t *testing.T) {
	edge := marketDomain.GraphEdge{
		Symbol: "BTCUSDT",
		// Direction: marketDomain.Buy, // если поле есть в текущем GraphEdge
	}

	amount := decimal.RequireFromString("100")
	clientOrderID := "client-123"

	request, err := domain.NewOrderRequest(
		edge,
		amount,
		clientOrderID,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if request.Edge.Symbol != "BTCUSDT" {
		t.Fatalf("unexpected symbol: %s", request.Edge.Symbol)
	}

	if !request.Amount.Equal(amount) {
		t.Fatalf(
			"unexpected amount: got %s, want %s",
			request.Amount,
			amount,
		)
	}

	if request.ClientOrderID != clientOrderID {
		t.Fatalf(
			"unexpected client order ID: got %s, want %s",
			request.ClientOrderID,
			clientOrderID,
		)
	}
}

func TestNewOrderRequestRejectsInvalidAmount(t *testing.T) {
	edge := marketDomain.GraphEdge{
		Symbol: "BTCUSDT",
	}

	_, err := domain.NewOrderRequest(
		edge,
		decimal.Zero,
		"client-123",
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewOrderRequestRejectsEmptyClientOrderID(t *testing.T) {
	edge := marketDomain.GraphEdge{
		Symbol: "BTCUSDT",
	}

	_, err := domain.NewOrderRequest(
		edge,
		decimal.RequireFromString("100"),
		"",
	)

	if err == nil {
		t.Fatal("expected error")
	}
}
