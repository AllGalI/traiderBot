package domain

import (
	"time"

	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type Opportunity struct {
	Route marketDomain.Route
	Steps []StepResult

	StartAsset string

	RequestedStartAmount decimal.Decimal
	ExecutedStartAmount  decimal.Decimal
	EndAmount            decimal.Decimal

	FillRatio decimal.Decimal

	ProfitAbs decimal.Decimal
	ProfitPct decimal.Decimal

	Timestamp time.Time
}
