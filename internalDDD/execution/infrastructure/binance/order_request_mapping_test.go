package binance

import (
	"testing"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestToCreateOrderRequest_Buy(t *testing.T) {
	request, err := executionDomain.NewOrderRequest(
		marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},
		decimal.RequireFromString("100.50"),
		"client-buy-123",
	)
	require.NoError(t, err)

	result, err := toCreateOrderRequest(request)
	require.NoError(t, err)

	require.Equal(t, "BTCUSDT", result.Symbol)
	require.Equal(t, "BUY", result.Side)
	require.Equal(t, "MARKET", result.Type)
	require.Equal(t, "100.5", result.QuoteOrderQty)
	require.Empty(t, result.Quantity)
	require.Equal(t, "client-buy-123", result.NewClientOrderID)
}

func TestToCreateOrderRequest_Sell(t *testing.T) {
	request, err := executionDomain.NewOrderRequest(
		marketDomain.GraphEdge{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Sell,
		},
		decimal.RequireFromString("0.025"),
		"client-sell-123",
	)
	require.NoError(t, err)

	result, err := toCreateOrderRequest(request)
	require.NoError(t, err)

	require.Equal(t, "BTCUSDT", result.Symbol)
	require.Equal(t, "SELL", result.Side)
	require.Equal(t, "MARKET", result.Type)
	require.Equal(t, "0.025", result.Quantity)
	require.Empty(t, result.QuoteOrderQty)
	require.Equal(t, "client-sell-123", result.NewClientOrderID)
}

func TestToCreateOrderRequest_UnsupportedDirection(t *testing.T) {
	request, err := executionDomain.NewOrderRequest(
		marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.TradeDirection(99),
		},
		decimal.RequireFromString("100"),
		"client-invalid",
	)
	require.NoError(t, err)

	_, err = toCreateOrderRequest(request)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported trade direction")
}
