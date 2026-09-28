package domain

import (
	"time"

	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type OpportunityID string

type Opportunity struct {
	ID OpportunityID

	Route marketDomain.Route
	Steps []StepResult

	StartAsset string

	// RequestedStartAmount — сколько входного актива
	// изначально хотели использовать для opportunity.
	RequestedStartAmount decimal.Decimal

	// ExecutedStartAmount — сколько входного актива
	// реально может быть исполнено на первом шаге
	// согласно Simulation.
	ExecutedStartAmount decimal.Decimal

	// EndAmount — конечный NET результат маршрута
	// после комиссий всех шагов.
	EndAmount decimal.Decimal

	FillRatio decimal.Decimal

	ProfitAbs decimal.Decimal
	ProfitPct decimal.Decimal

	Timestamp time.Time
}