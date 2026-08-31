package binance_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"

	binance "traiderBot/internalDDD/execution/infrastructure/binance"
)

const (
	testAPIKey    = "qNOarfUGAJ2EFuOX5RezAnXBrHgFDtelPRpwKZkGArribqD787uQPUGQL5p3De13"
	testAPISecret = "zK8mwqF6KN2em4ibmrp4K6tjh1X4Q2hMJepdEu4hOm1GlUV9nwUf5TJjp7H11T4c"
)

func TestClientExecuteMarketBuy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf(
					"expected POST, got %s",
					r.Method,
				)
			}

			if r.URL.Path != "/api/v3/order" {
				t.Fatalf(
					"expected /api/v3/order, got %s",
					r.URL.Path,
				)
			}

			if got := r.Header.Get("X-MBX-APIKEY"); got != testAPIKey {
				t.Fatalf(
					"expected api key %q, got %q",
					testAPIKey,
					got,
				)
			}

			query := r.URL.Query()

			assertQueryValue(
				t,
				query,
				"symbol",
				"BTCUSDT",
			)

			assertQueryValue(
				t,
				query,
				"side",
				"BUY",
			)

			assertQueryValue(
				t,
				query,
				"type",
				"MARKET",
			)

			assertQueryValue(
				t,
				query,
				"quoteOrderQty",
				"100",
			)

			assertQueryValue(
				t,
				query,
				"newClientOrderId",
				"execution-1-order-0-2",
			)

			assertSignedRequest(
				t,
				query,
				testAPISecret,
			)

			response := map[string]any{
				"symbol":              "BTCUSDT",
				"orderId":             int64(12345),
				"clientOrderId":       "execution-1-order-0-2",
				"transactTime":        int64(1700000000000),
				"updateTime":          int64(1700000001500),
				"status":              "FILLED",
				"side":                "BUY",
				"executedQty":         "0.00179",
				"cummulativeQuoteQty": "99.80",
				"fills": []map[string]any{
					{
						"price":           "55700",
						"qty":             "0.00179",
						"commission":      "0.00000179",
						"commissionAsset": "BTC",
					},
				},
			}

			writeJSON(
				t,
				w,
				response,
			)
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	request := executionDomain.OrderRequest{
		StepIndex: 0,

		Edge: marketDomain.GraphEdge{
			From:      "USDT",
			To:        "BTC",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Buy,
		},

		Amount:            decimal.NewFromInt(100),
		ExpectedAmountOut: decimal.RequireFromString("0.0018"),

		ClientOrderID: "execution-1-order-0-2",
	}

	result, err := client.Execute(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf(
			"unexpected execute error: %v",
			err,
		)
	}

	if result.StepIndex != 0 {
		t.Fatalf(
			"expected step index 0, got %d",
			result.StepIndex,
		)
	}

	if result.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected BTCUSDT, got %s",
			result.Symbol,
		)
	}

	if result.Direction != marketDomain.Buy {
		t.Fatalf(
			"expected buy direction, got %v",
			result.Direction,
		)
	}

	if result.ClientOrderID != request.ClientOrderID {
		t.Fatalf(
			"expected client order id %q, got %q",
			request.ClientOrderID,
			result.ClientOrderID,
		)
	}

	if result.ExchangeOrderID != "12345" {
		t.Fatalf(
			"expected exchange order id 12345, got %s",
			result.ExchangeOrderID,
		)
	}

	if !result.RequestedAmount.Equal(
		decimal.NewFromInt(100),
	) {
		t.Fatalf(
			"expected requested amount 100, got %s",
			result.RequestedAmount,
		)
	}

	if !result.ExecutedAmount.Equal(
		decimal.RequireFromString("99.80"),
	) {
		t.Fatalf(
			"expected executed amount 99.80, got %s",
			result.ExecutedAmount,
		)
	}

	if !result.ReceivedAmount.Equal(
		decimal.RequireFromString("0.00179"),
	) {
		t.Fatalf(
			"expected received amount 0.00179, got %s",
			result.ReceivedAmount,
		)
	}

	if result.ExpectedAmountOut != request.ExpectedAmountOut {
		t.Fatalf(
			"expected expected amount out %s, got %s",
			request.ExpectedAmountOut,
			result.ExpectedAmountOut,
		)
	}

	if result.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected filled status, got %s",
			result.Status,
		)
	}

	if !result.Fee.Equal(
		decimal.RequireFromString("0.00000179"),
	) {
		t.Fatalf(
			"expected fee 0.00000179, got %s",
			result.Fee,
		)
	}

	if result.FeeAsset != "BTC" {
		t.Fatalf(
			"expected fee asset BTC, got %s",
			result.FeeAsset,
		)
	}

	if result.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if result.SubmittedAt.IsZero() {
		t.Fatal("expected SubmittedAt to be set")
	}

	if result.FilledAt.IsZero() {
		t.Fatal("expected FilledAt to be set")
	}
}

func TestClientExecuteMarketSell(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			assertQueryValue(
				t,
				query,
				"symbol",
				"BTCUSDT",
			)

			assertQueryValue(
				t,
				query,
				"side",
				"SELL",
			)

			assertQueryValue(
				t,
				query,
				"type",
				"MARKET",
			)

			assertQueryValue(
				t,
				query,
				"quantity",
				"0.001",
			)

			if query.Get("quoteOrderQty") != "" {
				t.Fatal(
					"SELL order must not contain quoteOrderQty",
				)
			}

			response := map[string]any{
				"symbol":              "BTCUSDT",
				"orderId":             int64(54321),
				"clientOrderId":       "sell-order-1",
				"transactTime":        int64(1700000000000),
				"updateTime":          int64(1700000001500),
				"status":              "FILLED",
				"side":                "SELL",
				"executedQty":         "0.001",
				"cummulativeQuoteQty": "99.50",
			}

			writeJSON(
				t,
				w,
				response,
			)
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	request := executionDomain.OrderRequest{
		StepIndex: 1,

		Edge: marketDomain.GraphEdge{
			From:      "BTC",
			To:        "USDT",
			Symbol:    "BTCUSDT",
			Direction: marketDomain.Sell,
		},

		Amount:            decimal.RequireFromString("0.001"),
		ExpectedAmountOut: decimal.RequireFromString("100"),

		ClientOrderID: "sell-order-1",
	}

	result, err := client.Execute(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf(
			"unexpected execute error: %v",
			err,
		)
	}

	if !result.ExecutedAmount.Equal(
		decimal.RequireFromString("0.001"),
	) {
		t.Fatalf(
			"expected executed amount 0.001, got %s",
			result.ExecutedAmount,
		)
	}

	if !result.ReceivedAmount.Equal(
		decimal.RequireFromString("99.50"),
	) {
		t.Fatalf(
			"expected received amount 99.50, got %s",
			result.ReceivedAmount,
		)
	}

	if result.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected filled status, got %s",
			result.Status,
		)
	}
}

func TestClientGetOrderByClientOrderID(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf(
					"expected GET, got %s",
					r.Method,
				)
			}

			query := r.URL.Query()

			assertQueryValue(
				t,
				query,
				"symbol",
				"BTCUSDT",
			)

			assertQueryValue(
				t,
				query,
				"origClientOrderId",
				"execution-1-order-0-2",
			)

			assertSignedRequest(
				t,
				query,
				testAPISecret,
			)

			response := map[string]any{
				"symbol":              "BTCUSDT",
				"orderId":             int64(12345),
				"clientOrderId":       "execution-1-order-0-2",
				"transactTime":        int64(1700000000000),
				"updateTime":          int64(1700000001500),
				"status":              "FILLED",
				"side":                "BUY",
				"executedQty":         "0.00179",
				"cummulativeQuoteQty": "99.80",
			}

			writeJSON(
				t,
				w,
				response,
			)
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	result, err := client.GetOrder(
		context.Background(),
		"BTCUSDT",
		"execution-1-order-0-2",
	)
	if err != nil {
		t.Fatalf(
			"unexpected get order error: %v",
			err,
		)
	}

	if result.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected BTCUSDT, got %s",
			result.Symbol,
		)
	}

	if result.ClientOrderID != "execution-1-order-0-2" {
		t.Fatalf(
			"unexpected client order id: %s",
			result.ClientOrderID,
		)
	}

	if result.ExchangeOrderID != "12345" {
		t.Fatalf(
			"expected exchange order id 12345, got %s",
			result.ExchangeOrderID,
		)
	}

	if result.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected filled status, got %s",
			result.Status,
		)
	}

	if !result.ReceivedAmount.Equal(
		decimal.RequireFromString("0.00179"),
	) {
		t.Fatalf(
			"expected received amount 0.00179, got %s",
			result.ReceivedAmount,
		)
	}
}

func TestClientReturnsBinanceAPIError(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			w.WriteHeader(http.StatusBadRequest)

			_, _ = w.Write([]byte(
				`{"code":-1013,"msg":"Invalid quantity"}`,
			))
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	_, err = client.Execute(
		context.Background(),
		executionDomain.OrderRequest{
			Edge: marketDomain.GraphEdge{
				Symbol:    "BTCUSDT",
				Direction: marketDomain.Buy,
			},
			Amount:            decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.RequireFromString("0.001"),
			ClientOrderID:     "test-order",
		},
	)
	if err == nil {
		t.Fatal("expected Binance API error")
	}

	if !strings.Contains(
		err.Error(),
		"Invalid quantity",
	) {
		t.Fatalf(
			"expected error to contain Binance message, got %v",
			err,
		)
	}
}

func TestNewClientRejectsMissingCredentials(t *testing.T) {
	_, err := binance.NewClient(
		binance.Config{},
		http.DefaultClient,
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientExecuteRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	cancel()

	_, err = client.Execute(
		ctx,
		executionDomain.OrderRequest{
			Edge: marketDomain.GraphEdge{
				Symbol:    "BTCUSDT",
				Direction: marketDomain.Buy,
			},
			Amount:            decimal.NewFromInt(100),
			ExpectedAmountOut: decimal.RequireFromString("0.001"),
			ClientOrderID:     "test-order",
		},
	)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}

func TestClientGetOrderByClientOrderIDPreservesSellDirection(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			assertQueryValue(
				t,
				query,
				"symbol",
				"BTCUSDT",
			)

			assertQueryValue(
				t,
				query,
				"origClientOrderId",
				"sell-order-1",
			)

			response := map[string]any{
				"symbol":              "BTCUSDT",
				"orderId":             int64(54321),
				"clientOrderId":       "sell-order-1",
				"transactTime":        int64(1700000000000),
				"updateTime":          int64(1700000001500),
				"status":              "FILLED",
				"side":                "SELL",
				"executedQty":         "0.001",
				"cummulativeQuoteQty": "99.50",
			}

			writeJSON(t, w, response)
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	result, err := client.GetOrder(
		context.Background(),
		"BTCUSDT",
		"sell-order-1",
	)
	if err != nil {
		t.Fatalf(
			"unexpected get order error: %v",
			err,
		)
	}

	if result.Direction != marketDomain.Sell {
		t.Fatalf(
			"expected sell direction, got %v",
			result.Direction,
		)
	}

	if result.Status != executionDomain.OrderFilled {
		t.Fatalf(
			"expected filled status, got %s",
			result.Status,
		)
	}

	if result.ClientOrderID != "sell-order-1" {
		t.Fatalf(
			"expected client order id sell-order-1, got %s",
			result.ClientOrderID,
		)
	}

	if result.ExchangeOrderID != "54321" {
		t.Fatalf(
			"expected exchange order id 54321, got %s",
			result.ExchangeOrderID,
		)
	}
}

func TestClientRejectsUnknownBinanceOrderSide(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"symbol":              "BTCUSDT",
				"orderId":             int64(12345),
				"clientOrderId":       "test-order",
				"transactTime":        int64(1700000000000),
				"updateTime":          int64(1700000001500),
				"status":              "FILLED",
				"side":                "UNKNOWN",
				"executedQty":         "0.001",
				"cummulativeQuoteQty": "100",
			}

			writeJSON(t, w, response)
		}),
	)

	defer server.Close()

	client, err := binance.NewClient(
		binance.Config{
			APIKey:    testAPIKey,
			APISecret: testAPISecret,
			BaseURL:   server.URL,
		},
		http.DefaultClient,
	)
	if err != nil {
		t.Fatalf(
			"unexpected client creation error: %v",
			err,
		)
	}

	_, err = client.GetOrder(
		context.Background(),
		"BTCUSDT",
		"test-order",
	)
	if err == nil {
		t.Fatal("expected unknown side error")
	}
}

func assertQueryValue(
	t *testing.T,
	query url.Values,
	key string,
	expected string,
) {
	t.Helper()

	actual := query.Get(key)

	if actual != expected {
		t.Fatalf(
			"expected query %s=%q, got %q",
			key,
			expected,
			actual,
		)
	}
}

func assertSignedRequest(
	t *testing.T,
	query url.Values,
	secret string,
) {
	t.Helper()

	signature := query.Get("signature")

	if signature == "" {
		t.Fatal("signature is missing")
	}

	values := url.Values{}

	for key, items := range query {
		if key == "signature" {
			continue
		}

		for _, item := range items {
			values.Add(key, item)
		}
	}

	payload := values.Encode()

	mac := hmac.New(
		sha256.New,
		[]byte(secret),
	)

	_, _ = mac.Write([]byte(payload))

	expected := hex.EncodeToString(
		mac.Sum(nil),
	)

	if signature != expected {
		t.Fatalf(
			"invalid signature: expected %s, got %s",
			expected,
			signature,
		)
	}
}

func writeJSON(
	t *testing.T,
	w http.ResponseWriter,
	value any,
) {
	t.Helper()

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf(
			"encode response: %v",
			err,
		)
	}
}

// func TestNewClientRejectsNilHTTPClient(t *testing.T) {
// 	_, err := binance.NewClient(
// 		binance.Config{
// 			APIKey:    testAPIKey,
// 			APISecret: testAPISecret,
// 			BaseURL:   server.URL,
// 		},
// 		http.DefaultClient,
// 	)

// 	if err == nil {
// 		t.Fatal("expected error for nil http client")
// 	}
// }
