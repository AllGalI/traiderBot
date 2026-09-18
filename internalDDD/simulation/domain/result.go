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


func (r SimulationReason) String() string {
	switch r {
	case SimulationOK:
		return "OK"
	case SimulationInvalidRoute:
		return "InvalidRoute"
	case SimulationInvalidStartAmount:
		return "InvalidStartAmount"
	case SimulationOrderBookNotFound:
		return "OrderBookNotFound"
	case SimulationFeeNotFound:
		return "FeeNotFound"
	case SimulationInsufficientLiquidity:
		return "InsufficientLiquidity"
	case SimulationFiltersNotFound:
		return "FiltersNotFound"
	case SimulationProfitTooLow:
		return "ProfitTooLow"
	case SimulationOrderBookStale:
		return "OrderBookStale"
	default:
		return "Unknown"
	}
}