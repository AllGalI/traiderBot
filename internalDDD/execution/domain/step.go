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
	PlannedInput AssetAmount
}

type AssetAmount struct {
	Asset    string
	Quantity decimal.Decimal
}

type ActualStepResult struct {
	StepID int

	// PlannedInput — фактическая копия планового
	// входного объёма из Simulation.
	PlannedInput AssetAmount

	// RequestedInput — объём, который Execution
	// запросила у Executor.
	RequestedInput AssetAmount

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

	// FillRatio — отношение фактически исполненного
	// входного объёма к запрошенному.
	FillRatio decimal.Decimal

	// Result — итоговый статус ордера.
	Result OrderStatus

	// Error — ошибка, если шаг завершился ошибкой.
	Error string

	StartedAt   time.Time
	CompletedAt time.Time
}