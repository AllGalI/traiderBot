package domain

import (
	"time"

	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type ExecutionStep struct {
	ID   int
	Edge marketDomain.GraphEdge
}

type AssetAmount struct {
	Asset    string
	Quantity decimal.Decimal
}

type ActualStepResult struct {
	StepID      int
	Input       AssetAmount
	Output      AssetAmount
	CompletedAt time.Time
}
