package executor

import (
	"context"
	"time"
	"traiderBot/internal/arbitrage"
)

type OrderSide uint8

const (
	Buy OrderSide = iota
	Sell
)

type OrderType uint8

const (
	LimitIOC OrderType = iota
	Market
)

type ExecutionResult struct {
	Opportunity arbitrage.Opportunity

	Status ExecutionStatus

	Steps []ExecutionStep

	StartedAt  time.Time
	FinishedAt time.Time

	Profit float64

	Err error
}

type Executor interface {
	Execute(ctx context.Context, opp arbitrage.Opportunity) (*ExecutionResult, error)
}

type ExecutionStatus string

const (
	ExecutionSuccess   ExecutionStatus = "SUCCESS"
	ExecutionFailed    ExecutionStatus = "FAILED"
	ExecutionRecovered ExecutionStatus = "RECOVERED"
)

type ExecutionStep struct {
	Planned arbitrage.StepResult

	Order OrderResult

	StartedAt  time.Time
	FinishedAt time.Time
}

type OrderResult struct {
	OrderID int64

	Symbol string

	Side string

	Status string

	Price float64

	ExecutedQty float64

	CummulativeQuoteQty float64

	Commission      float64
	CommissionAsset string
}

type OrderExecutor interface {
	PlaceLimitIOC(
		ctx context.Context,
		step arbitrage.StepResult,
	) (*OrderResult, error)

	PlaceMarket(
		ctx context.Context,
		symbol string,
		side string,
		quantity float64,
	) (*OrderResult, error)
}
