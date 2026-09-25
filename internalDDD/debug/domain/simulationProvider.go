package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	simulationApplication "traiderBot/internalDDD/simulation/application"
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

func (p *SimulationProvider) Snapshot() (string, error) {
	metrics := p.reader.Snapshot()

	var builder strings.Builder

	fmt.Fprintln(&builder, "Simulation Worker")
	fmt.Fprintln(&builder, "-----------------")

	fmt.Fprintf(
		&builder,
		"Updates received: %d\n",
		metrics.UpdatesReceived,
	)

	fmt.Fprintf(
		&builder,
		"Routes simulated: %d\n",
		metrics.RoutesSimulated,
	)

	fmt.Fprintf(
		&builder,
		"Opportunities: %d\n",
		metrics.Opportunities,
	)

	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "Simulation results")
	fmt.Fprintln(&builder, "------------------")

	fmt.Fprintf(
		&builder,
		"OK: %d\n",
		metrics.SimulationOK,
	)

	fmt.Fprintf(
		&builder,
		"Invalid route: %d\n",
		metrics.SimulationInvalidRoute,
	)

	fmt.Fprintf(
		&builder,
		"Invalid start amount: %d\n",
		metrics.SimulationInvalidStartAmount,
	)

	fmt.Fprintf(
		&builder,
		"Order book not found: %d\n",
		metrics.SimulationOrderBookNotFound,
	)

	fmt.Fprintf(
		&builder,
		"Order book stale: %d\n",
		metrics.SimulationOrderBookStale,
	)

	fmt.Fprintf(
		&builder,
		"Filters not found: %d\n",
		metrics.SimulationFiltersNotFound,
	)

	fmt.Fprintf(
		&builder,
		"Fee not found: %d\n",
		metrics.SimulationFeeNotFound,
	)

	fmt.Fprintf(
		&builder,
		"Insufficient liquidity: %d\n",
		metrics.SimulationInsufficientLiquidity,
	)

	fmt.Fprintf(
		&builder,
		"Profit too low: %d\n",
		metrics.SimulationProfitTooLow,
	)

	fmt.Fprintln(&builder)

	fmt.Fprintf(
		&builder,
		"Last update: %s\n",
		formatTime(metrics.LastUpdateAt),
	)

	fmt.Fprintf(
		&builder,
		"Last simulation: %s\n",
		formatTime(metrics.LastSimulationAt),
	)

	if metrics.HasProfitRange {
		fmt.Fprintf(
			&builder,
			"Min profit: %s%%\n",
			metrics.MinProfitPct.Mul(decimal.NewFromInt(100)),
		)

		fmt.Fprintf(
			&builder,
			"Max profit: %s%%\n",
			metrics.MaxProfitPct.Mul(decimal.NewFromInt(100)),
		)
	} else {
		fmt.Fprintln(
			&builder,
			"Profit range: no opportunities",
		)
	}

	return builder.String(), nil
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
			"feeNotFound":            metrics.SimulationFeeNotFound,
			"insufficientLiquidity": metrics.SimulationInsufficientLiquidity,
			"profitTooLow":           metrics.SimulationProfitTooLow,
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