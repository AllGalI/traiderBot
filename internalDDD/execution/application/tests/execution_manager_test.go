package tests

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	executionApplication "traiderBot/internalDDD/execution/application"
	executionDomain "traiderBot/internalDDD/execution/domain"
	marketDomain "traiderBot/internalDDD/market/domain"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)




type fakeExecutionOrderQueryWithRequest struct {
	result executionDomain.OrderExecutionResult

	calls   int
	symbol  string
	orderID string
}

func (f *fakeExecutionOrderQueryWithRequest) GetOrder(
	ctx context.Context,
	symbol string,
	orderID string,
) (executionDomain.OrderExecutionResult, error) {
	f.calls++
	f.symbol = symbol
	f.orderID = orderID

	return f.result, nil
}

func TestExecutionManagerBuildsOrderRequestWithExecutionMetadata(t *testing.T) {
	opportunity := simulationDomain.Opportunity{
		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(100),
		ExecutedStartAmount:  decimal.NewFromInt(100),

		Steps: []simulationDomain.StepResult{
			{
				AmountIn:  decimal.NewFromInt(100),
				AmountOut: decimal.NewFromFloat(0.0018),
				Edge: marketDomain.GraphEdge{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: marketDomain.Buy,
				},
			},
			{
				AmountIn:  decimal.NewFromFloat(0.0018),
				AmountOut: decimal.NewFromInt(5),
				Edge: marketDomain.GraphEdge{
					From:      "BTC",
					To:        "ETH",
					Symbol:    "ETHBTC",
					Direction: marketDomain.Buy,
				},
			},
		},
	}

	executor := &fakeExchangeExecutorCaptureRequest{
		results: []decimal.Decimal{
			decimal.NewFromFloat(0.0018),
			decimal.NewFromInt(5),
		},
	}

	manager := executionApplication.NewExecutionManager(
		executor,
		nil,
		fakeExecutionPolicy{},
	)

	execution, err := manager.Execute(
		context.Background(),
		&opportunity,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if execution.Status != executionDomain.ExecutionCompleted {
		t.Fatalf(
			"expected completed execution, got %s",
			execution.Status,
		)
	}

	if len(executor.requests) != 2 {
		t.Fatalf(
			"expected 2 executor requests, got %d",
			len(executor.requests),
		)
	}

	first := executor.requests[0]

	if first.StepIndex != 0 {
		t.Fatalf(
			"expected first step index 0, got %d",
			first.StepIndex,
		)
	}

	if first.ClientOrderID == "" {
		t.Fatal("expected first client order ID to be generated")
	}

	if !first.Amount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf(
			"expected first amount 100, got %s",
			first.Amount,
		)
	}

	if !first.ExpectedAmountOut.Equal(
		decimal.NewFromFloat(0.0018),
	) {
		t.Fatalf(
			"expected first expected amount out 0.0018, got %s",
			first.ExpectedAmountOut,
		)
	}

	second := executor.requests[1]

	if second.StepIndex != 1 {
		t.Fatalf(
			"expected second step index 1, got %d",
			second.StepIndex,
		)
	}

	if second.ClientOrderID == "" {
		t.Fatal("expected second client order ID to be generated")
	}

	if second.ClientOrderID == first.ClientOrderID {
		t.Fatal("expected unique client order IDs")
	}

	if !second.Amount.Equal(
		decimal.NewFromFloat(0.0018),
	) {
		t.Fatalf(
			"expected second amount 0.0018, got %s",
			second.Amount,
		)
	}

	if !second.ExpectedAmountOut.Equal(
		decimal.NewFromInt(5),
	) {
		t.Fatalf(
			"expected second expected amount out 5, got %s",
			second.ExpectedAmountOut,
		)
	}
}

type fakeExchangeExecutorCaptureRequest struct {
	requests []executionDomain.OrderRequest

	results []decimal.Decimal
}

func (f *fakeExchangeExecutorCaptureRequest) Execute(
	ctx context.Context,
	request executionDomain.OrderRequest,
) (executionDomain.OrderExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return executionDomain.OrderExecutionResult{}, err
	}

	index := len(f.requests)

	f.requests = append(
		f.requests,
		request,
	)

	now := time.Now()

	return executionDomain.OrderExecutionResult{
		StepIndex: request.StepIndex,

		Symbol:    request.Edge.Symbol,
		Direction: request.Edge.Direction,

		ClientOrderID: request.ClientOrderID,

		RequestedAmount:   request.Amount,
		ExpectedAmountOut: request.ExpectedAmountOut,
		ExecutedAmount:    request.Amount,
		ReceivedAmount:    f.results[index],

		Status: executionDomain.OrderFilled,

		CreatedAt:   now,
		SubmittedAt: now,
		FilledAt:    now,
	}, nil
}
