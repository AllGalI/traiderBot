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