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

func TestExecutionManagerExecutesAllStepsAndPropagatesAmounts(t *testing.T) {
	opportunity := simulationDomain.Opportunity{
		StartAsset: "USDT",

		RequestedStartAmount: decimal.NewFromInt(100),
		ExecutedStartAmount:  decimal.NewFromInt(100),

		Steps: []simulationDomain.StepResult{
			{
				AmountIn:  decimal.NewFromInt(100),
				AmountOut: decimal.NewFromInt(101),
				Edge: marketDomain.GraphEdge{
					From:      "USDT",
					To:        "BTC",
					Symbol:    "BTCUSDT",
					Direction: marketDomain.Buy,
				},
			},
			{
				AmountIn:  decimal.NewFromInt(101),
				AmountOut: decimal.NewFromFloat(102.5),
				Edge: marketDomain.GraphEdge{
					From:      "BTC",
					To:        "ETH",
					Symbol:    "ETHBTC",
					Direction: marketDomain.Buy,
				},
			},
			{
				AmountIn:  decimal.NewFromFloat(102.5),
				AmountOut: decimal.NewFromFloat(105),
				Edge: marketDomain.GraphEdge{
					From:      "ETH",
					To:        "USDT",
					Symbol:    "ETHUSDT",
					Direction: marketDomain.Sell,
				},
			},
		},
	}

	executor := &fakeExchangeExecutor{
		results: []decimal.Decimal{
			decimal.NewFromInt(101),
			decimal.NewFromFloat(102.5),
			decimal.NewFromInt(105),
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
		t.Fatalf("unexpected execution error: %v", err)
	}

	if execution.Status != executionDomain.ExecutionCompleted {
		t.Fatalf(
			"expected completed execution, got %s",
			execution.Status,
		)
	}

	if len(execution.Orders) != 3 {
		t.Fatalf(
			"expected 3 orders, got %d",
			len(execution.Orders),
		)
	}

	expectedAmounts := []decimal.Decimal{
		decimal.NewFromInt(100),
		decimal.NewFromInt(101),
		decimal.NewFromFloat(102.5),
	}

	for i, expected := range expectedAmounts {
		if !executor.amountsIn[i].Equal(expected) {
			t.Fatalf(
				"step %d: expected input amount %s, got %s",
				i,
				expected,
				executor.amountsIn[i],
			)
		}
	}

	expectedFinalAmount := decimal.NewFromInt(105)

	if !execution.FinalAmount.Equal(expectedFinalAmount) {
		t.Fatalf(
			"expected final amount %s, got %s",
			expectedFinalAmount,
			execution.FinalAmount,
		)
	}

	if !execution.CurrentAmount.Equal(expectedFinalAmount) {
		t.Fatalf(
			"expected current amount %s, got %s",
			expectedFinalAmount,
			execution.CurrentAmount,
		)
	}

	for i, order := range execution.Orders {
		if order.StepIndex != i {
			t.Fatalf(
				"order %d: expected step index %d, got %d",
				i,
				i,
				order.StepIndex,
			)
		}

		if order.Status != executionDomain.OrderFilled {
			t.Fatalf(
				"order %d: expected filled status, got %s",
				i,
				order.Status,
			)
		}
	}
}
