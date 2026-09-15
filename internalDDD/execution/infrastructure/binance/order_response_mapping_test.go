package binance

import (
	"strings"
	"testing"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

func TestToOrderResult_Buy(t *testing.T) {
	request := executionDomain.OrderRequest{
		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		Amount:        decimal.RequireFromString("1000"),
		ClientOrderID: "client-123",
	}

	response := orderResponse{
		Symbol:             "BTCUSDT",
		OrderID:            12345,
		ClientOrderID:      "client-123",
		Status:             "FILLED",
		ExecutedQty:        "0.02",
		CumulativeQuoteQty: "1000",
		Fills: []orderFill{
			{
				Commission:      "0.00002",
				CommissionAsset: "BTC",
			},
		},
	}

	result, err := toOrderResult(request, response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != executionDomain.OrderStatusFilled {
		t.Fatalf("expected status Filled, got %v", result.Status)
	}

	if result.ExchangeOrderID != "12345" {
		t.Fatalf(
			"expected exchange order ID %q, got %q",
			"12345",
			result.ExchangeOrderID,
		)
	}

	if result.ClientOrderID != "client-123" {
		t.Fatalf(
			"expected client order ID %q, got %q",
			"client-123",
			result.ClientOrderID,
		)
	}

	if !result.ExecutedAmount.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf(
			"expected executed amount %s, got %s",
			"1000",
			result.ExecutedAmount,
		)
	}

	if !result.ReceivedAmount.Equal(decimal.RequireFromString("0.02")) {
		t.Fatalf(
			"expected received amount %s, got %s",
			"0.02",
			result.ReceivedAmount,
		)
	}

	if !result.Fee.Equal(decimal.RequireFromString("0.00002")) {
		t.Fatalf(
			"expected fee %s, got %s",
			"0.00002",
			result.Fee,
		)
	}

	if result.FeeAsset != "BTC" {
		t.Fatalf(
			"expected fee asset %q, got %q",
			"BTC",
			result.FeeAsset,
		)
	}
}

func TestToOrderResult_Sell(t *testing.T) {
	request := executionDomain.OrderRequest{
		Edge: marketDomain.GraphEdge{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Sell,
		},
		Amount:        decimal.RequireFromString("0.02"),
		ClientOrderID: "client-456",
	}

	response := orderResponse{
		Symbol:             "BTCUSDT",
		OrderID:            67890,
		ClientOrderID:      "client-456",
		Status:             "FILLED",
		ExecutedQty:        "0.02",
		CumulativeQuoteQty: "1000",
		Fills: []orderFill{
			{
				Commission:      "1",
				CommissionAsset: "USDT",
			},
		},
	}

	result, err := toOrderResult(request, response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != executionDomain.OrderStatusFilled {
		t.Fatalf("expected status Filled, got %v", result.Status)
	}

	if !result.ExecutedAmount.Equal(decimal.RequireFromString("0.02")) {
		t.Fatalf(
			"expected executed amount %s, got %s",
			"0.02",
			result.ExecutedAmount,
		)
	}

	if !result.ReceivedAmount.Equal(decimal.RequireFromString("1000")) {
		t.Fatalf(
			"expected received amount %s, got %s",
			"1000",
			result.ReceivedAmount,
		)
	}

	if !result.Fee.Equal(decimal.RequireFromString("1")) {
		t.Fatalf(
			"expected fee %s, got %s",
			"1",
			result.Fee,
		)
	}

	if result.FeeAsset != "USDT" {
		t.Fatalf(
			"expected fee asset %q, got %q",
			"USDT",
			result.FeeAsset,
		)
	}
}

func TestToOrderResult_InvalidExecutedQty(t *testing.T) {
	request := executionDomain.OrderRequest{
		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		Amount:        decimal.RequireFromString("1000"),
		ClientOrderID: "client-123",
	}

	response := orderResponse{
		ExecutedQty:        "invalid",
		CumulativeQuoteQty: "1000",
		Status:             "FILLED",
	}

	_, err := toOrderResult(request, response)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "parse executed qty") {
		t.Fatalf(
			"expected executed qty error, got %v",
			err,
		)
	}
}

func TestToOrderResult_InvalidCumulativeQuoteQty(t *testing.T) {
	request := executionDomain.OrderRequest{
		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		Amount:        decimal.RequireFromString("1000"),
		ClientOrderID: "client-123",
	}

	response := orderResponse{
		ExecutedQty:        "0.02",
		CumulativeQuoteQty: "invalid",
		Status:             "FILLED",
	}

	_, err := toOrderResult(request, response)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "parse cumulative quote qty") {
		t.Fatalf(
			"expected cumulative quote qty error, got %v",
			err,
		)
	}
}

func TestToOrderResult_UnknownDirection(t *testing.T) {
	request := executionDomain.OrderRequest{
		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.TradeDirection(99),
		},
		Amount:        decimal.RequireFromString("1000"),
		ClientOrderID: "client-123",
	}

	response := orderResponse{
		ExecutedQty:        "0.02",
		CumulativeQuoteQty: "1000",
		Status:             "FILLED",
	}

	_, err := toOrderResult(request, response)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported trade direction") {
		t.Fatalf(
			"expected unsupported direction error, got %v",
			err,
		)
	}
}

func TestMapOrderStatus(t *testing.T) {
	tests := []struct {
		name           string
		binanceStatus  string
		expectedStatus executionDomain.OrderStatus
	}{
		{
			name:           "new",
			binanceStatus:  "NEW",
			expectedStatus: executionDomain.OrderStatusPending,
		},
		{
			name:           "partially filled",
			binanceStatus:  "PARTIALLY_FILLED",
			expectedStatus: executionDomain.OrderStatusPartiallyFilled,
		},
		{
			name:           "filled",
			binanceStatus:  "FILLED",
			expectedStatus: executionDomain.OrderStatusFilled,
		},
		{
			name:           "canceled",
			binanceStatus:  "CANCELED",
			expectedStatus: executionDomain.OrderStatusCancelled,
		},
		{
			name:           "expired",
			binanceStatus:  "EXPIRED",
			expectedStatus: executionDomain.OrderStatusCancelled,
		},
		{
			name:           "pending cancel",
			binanceStatus:  "PENDING_CANCEL",
			expectedStatus: executionDomain.OrderStatusCancelled,
		},
		{
			name:           "rejected",
			binanceStatus:  "REJECTED",
			expectedStatus: executionDomain.OrderStatusRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := mapOrderStatus(tt.binanceStatus)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if status != tt.expectedStatus {
				t.Fatalf(
					"expected status %v, got %v",
					tt.expectedStatus,
					status,
				)
			}
		})
	}
}

func TestMapOrderStatus_Unknown(t *testing.T) {
	status, err := mapOrderStatus("SOME_NEW_BINANCE_STATUS")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status != executionDomain.OrderStatusUnknown {
		t.Fatalf(
			"expected status Unknown, got %v",
			status,
		)
	}

	if !strings.Contains(err.Error(), "unsupported binance order status") {
		t.Fatalf(
			"expected unsupported status error, got %v",
			err,
		)
	}
}

func TestParseOrderFee(t *testing.T) {
	tests := []struct {
		name          string
		fills         []orderFill
		expectedFee   string
		expectedAsset string
	}{
		{
			name:          "single fill",
			fills: []orderFill{
				{
					Commission:      "0.001",
					CommissionAsset: "BTC",
				},
			},
			expectedFee:   "0.001",
			expectedAsset: "BTC",
		},
		{
			name: "multiple fills same asset",
			fills: []orderFill{
				{
					Commission:      "0.001",
					CommissionAsset: "BTC",
				},
				{
					Commission:      "0.002",
					CommissionAsset: "BTC",
				},
			},
			expectedFee:   "0.003",
			expectedAsset: "BTC",
		},
		{
			name: "zero commission is ignored",
			fills: []orderFill{
				{
					Commission:      "0",
					CommissionAsset: "",
				},
				{
					Commission:      "0.001",
					CommissionAsset: "BTC",
				},
			},
			expectedFee:   "0.001",
			expectedAsset: "BTC",
		},
		{
			name:          "no fills",
			fills:         nil,
			expectedFee:   "0",
			expectedAsset: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee, asset, err := parseOrderFee(tt.fills)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedFee := decimal.RequireFromString(tt.expectedFee)

			if !fee.Equal(expectedFee) {
				t.Fatalf(
					"expected fee %s, got %s",
					expectedFee,
					fee,
				)
			}

			if asset != tt.expectedAsset {
				t.Fatalf(
					"expected asset %q, got %q",
					tt.expectedAsset,
					asset,
				)
			}
		})
	}
}

func TestParseOrderFee_InvalidCommission(t *testing.T) {
	fills := []orderFill{
		{
			Commission:      "invalid",
			CommissionAsset: "BTC",
		},
	}

	_, _, err := parseOrderFee(fills)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "parse commission") {
		t.Fatalf(
			"expected parse commission error, got %v",
			err,
		)
	}
}

func TestParseOrderFee_NegativeCommission(t *testing.T) {
	fills := []orderFill{
		{
			Commission:      "-0.001",
			CommissionAsset: "BTC",
		},
	}

	_, _, err := parseOrderFee(fills)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "commission must not be negative") {
		t.Fatalf(
			"expected negative commission error, got %v",
			err,
		)
	}
}

func TestParseOrderFee_MissingAsset(t *testing.T) {
	fills := []orderFill{
		{
			Commission:      "0.001",
			CommissionAsset: "",
		},
	}

	_, _, err := parseOrderFee(fills)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "commission asset is empty") {
		t.Fatalf(
			"expected missing asset error, got %v",
			err,
		)
	}
}

func TestParseOrderFee_MultipleAssets(t *testing.T) {
	fills := []orderFill{
		{
			Commission:      "0.001",
			CommissionAsset: "BTC",
		},
		{
			Commission:      "1",
			CommissionAsset: "USDT",
		},
	}

	_, _, err := parseOrderFee(fills)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(
		err.Error(),
		"multiple commission assets are not supported",
	) {
		t.Fatalf(
			"expected multiple assets error, got %v",
			err,
		)
	}
}