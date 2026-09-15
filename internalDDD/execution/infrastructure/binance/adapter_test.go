package binance

import (
	"context"
	"testing"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type mockAPIClient struct {
	executeRequest executionDomain.OrderRequest
	executeResult  executionDomain.OrderResult
	executeErr     error
}

func (m *mockAPIClient) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderResult, error) {
	m.executeRequest = request

	return m.executeResult, m.executeErr
}

func (m *mockAPIClient) GetOrder(
	ctx context.Context,
	symbol string,
	orderID string,
) (executionDomain.OrderResult, error) {
	return executionDomain.OrderResult{}, nil
}

func TestAdapterExecute(t *testing.T) {
	request, err := executionDomain.NewOrderRequest(
		marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		decimal.RequireFromString("100"),
		"client-order-123",
	)
	require.NoError(t, err)

	expectedResult := executionDomain.OrderResult{
		Status:          executionDomain.OrderStatusFilled,
		ExchangeOrderID: "123456",
		ClientOrderID:   "client-order-123",
		ExecutedAmount:  decimal.RequireFromString("100"),
		ReceivedAmount:  decimal.RequireFromString("0.001"),
		Fee:             decimal.RequireFromString("0.000001"),
		FeeAsset:        "BTC",
	}

	client := &mockAPIClient{
		executeResult: expectedResult,
	}

	adapter, err := NewAdapter(client)
	require.NoError(t, err)

	result, err := adapter.Execute(context.Background(), request)
	require.NoError(t, err)

	require.Equal(t, expectedResult, result)
	require.Equal(t, request, client.executeRequest)
}

func TestAdapterExecute_ReturnsClientError(t *testing.T) {
	request, err := executionDomain.NewOrderRequest(
		marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		decimal.RequireFromString("100"),
		"client-order-123",
	)
	require.NoError(t, err)

	expectedErr := context.DeadlineExceeded

	client := &mockAPIClient{
		executeErr: expectedErr,
	}

	adapter, err := NewAdapter(client)
	require.NoError(t, err)

	_, err = adapter.Execute(context.Background(), request)

	require.ErrorIs(t, err, expectedErr)
}
