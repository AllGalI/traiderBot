package domain

import (
	"time"

	marketDomain "traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

type ExecutionStep struct {
	ID   int
	Edge marketDomain.GraphEdge

	// PlannedInput — объём входного актива,
	// который Simulation первоначально рассчитала
	// как исполнимый для этого шага.
	//
	// Используется Execution для сравнения
	// плановой и фактической исполнимости.
	PlannedInput AssetAmount
}

type AssetAmount struct {
	Asset    string
	Quantity decimal.Decimal
}

type ActualStepResult struct {
	StepID int

	// ActualInput — фактически исполненный
	// входной объём до комиссии.
	ActualInput AssetAmount

	// GrossOutput — фактически полученный
	// выходной объём до комиссии.
	GrossOutput AssetAmount

	// Fee — фактически удержанная комиссия.
	Fee AssetAmount

	// NetOutput — фактически оставшийся
	// выходной объём после комиссии.
	NetOutput AssetAmount

	CompletedAt time.Time
}