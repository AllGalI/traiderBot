package domain

import "github.com/shopspring/decimal"

type SimulationReason uint8

const (
	SimulationOK SimulationReason = iota
	SimulationInvalidRoute
	SimulationInvalidStartAmount
	SimulationOrderBookNotFound
	SimulationFeeNotFound
	SimulationInsufficientLiquidity
	SimulationFiltersNotFound
	SimulationProfitTooLow
	SimulationOrderBookStale
)

type SimulationResult struct {
	Opportunity *Opportunity
	Reason      SimulationReason

	// Доля исходного объёма, которая фактически
	// прошла через весь маршрут.
	//
	// 1.0 = весь объём.
	// 0.56 = через весь маршрут прошло 56%.
	RouteFillRatio decimal.Decimal
}
