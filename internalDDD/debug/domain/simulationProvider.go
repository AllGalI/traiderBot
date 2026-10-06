package domain

import (
	"time"

	simulationApplication "traiderBot/internalDDD/simulation/application"

	"github.com/shopspring/decimal"
)

type SimulationProvider struct {
	reader interface {
		Snapshot() simulationApplication.WorkerMetricsSnapshot
	}
}

func NewSimulationProvider(
	reader interface {
		Snapshot() simulationApplication.WorkerMetricsSnapshot
	},
) *SimulationProvider {
	return &SimulationProvider{
		reader: reader,
	}
}

func (p *SimulationProvider) Name() string {
	return "SimulationProvider"
}

func (p *SimulationProvider) Snapshot() (any, error) {
	metrics := p.reader.Snapshot()

	response := map[string]any{
		"updatesReceived":  metrics.UpdatesReceived,
		"routesSimulated":  metrics.RoutesSimulated,
		"opportunities":    metrics.Opportunities,
		"lastUpdateAt":     formatTime(metrics.LastUpdateAt),
		"lastSimulationAt": formatTime(metrics.LastSimulationAt),

		"simulationResults": map[string]any{
			"ok":                    metrics.SimulationOK,
			"invalidRoute":          metrics.SimulationInvalidRoute,
			"invalidStartAmount":    metrics.SimulationInvalidStartAmount,
			"orderBookNotFound":     metrics.SimulationOrderBookNotFound,
			"orderBookStale":        metrics.SimulationOrderBookStale,
			"filtersNotFound":       metrics.SimulationFiltersNotFound,
			"feeNotFound":           metrics.SimulationFeeNotFound,
			"insufficientLiquidity": metrics.SimulationInsufficientLiquidity,
			"profitTooLow":          metrics.SimulationProfitTooLow,
		},
	}

	if metrics.HasProfitRange {
		response["profitRange"] = map[string]any{
			"minPct": metrics.MinProfitPct.Mul(
				decimal.NewFromInt(100),
			),
			"maxPct": metrics.MaxProfitPct.Mul(
				decimal.NewFromInt(100),
			),
		}
	} else {
		response["profitRange"] = nil
	}

	return response, nil
}

func (p *SimulationProvider) Metrics() (any, error) {
	metrics := p.reader.Snapshot()

	response := map[string]any{
		"updatesReceived":     metrics.UpdatesReceived,
		"routesSimulated":     metrics.RoutesSimulated,
		"opportunities":       metrics.Opportunities,
		"totalSimulationTime": metrics.TotalSimulationTime,
		"lastUpdateAt":        metrics.LastUpdateAt,
		"lastSimulationAt":    metrics.LastSimulationAt,

		"simulationResults": map[string]uint64{
			"ok":                    metrics.SimulationOK,
			"invalidRoute":          metrics.SimulationInvalidRoute,
			"invalidStartAmount":    metrics.SimulationInvalidStartAmount,
			"orderBookNotFound":     metrics.SimulationOrderBookNotFound,
			"orderBookStale":        metrics.SimulationOrderBookStale,
			"filtersNotFound":       metrics.SimulationFiltersNotFound,
			"feeNotFound":           metrics.SimulationFeeNotFound,
			"insufficientLiquidity": metrics.SimulationInsufficientLiquidity,
			"profitTooLow":          metrics.SimulationProfitTooLow,
		},
	}

	if metrics.HasProfitRange {
		response["minProfitPct"] = metrics.MinProfitPct
		response["maxProfitPct"] = metrics.MaxProfitPct
	} else {
		response["minProfitPct"] = nil
		response["maxProfitPct"] = nil
	}

	if metrics.RoutesSimulated > 0 {
		response["averageSimulationTime"] =
			metrics.TotalSimulationTime /
				time.Duration(metrics.RoutesSimulated)
	} else {
		response["averageSimulationTime"] = time.Duration(0)
	}

	return response, nil
}
