package tests

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestSimulationService_Opportunity_FullFill(t *testing.T) {
	store := &orderBookReaderStub{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": newOrderBook(
				"BTCUSDT",
				"100000",
				"0.001",
			),
			"ETHBTC": newOrderBook(
				"ETHBTC",
				"0.05",
				"1",
			),
			"ETHUSDT": newOrderBook(
				"ETHUSDT",
				"5100",
				"1",
			),
		},
	}

	fees := &feeProviderStub{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate: decimal.Zero,
			},
			"ETHBTC": {
				Rate: decimal.Zero,
			},
			"ETHUSDT": {
				Rate: decimal.Zero,
			},
		},
	}

	symbols := &symbolProviderStub{
		symbols: map[string]marketDomain.Symbol{
			"BTCUSDT": {},
			"ETHBTC":  {},
			"ETHUSDT": {},
		},
	}

	service := simulationApplication.NewSimulationService(
		store,
		fees,
		symbols,
		alwaysFreshPolicy{},
		decimal.RequireFromString("0.30"),
		decimal.Zero,
	)

	route := triangleRoute()
	startAmount := decimal.NewFromInt(100)

	before := time.Now()

	result := service.Simulate(context.Background(), route, startAmount)

	after := time.Now()

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK, got %v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity, got nil")
	}

	opportunity := result.Opportunity

	if len(opportunity.Steps) != 3 {
		t.Fatalf(
			"expected 3 steps, got %d",
			len(opportunity.Steps),
		)
	}

	if len(opportunity.Route.Edges) != len(route.Edges) {
		t.Fatalf(
			"expected route with %d edges, got %d",
			len(route.Edges),
			len(opportunity.Route.Edges),
		)
	}

	if opportunity.StartAsset != "USDT" {
		t.Fatalf(
			"expected start asset USDT, got %s",
			opportunity.StartAsset,
		)
	}

	if !opportunity.RequestedStartAmount.Equal(startAmount) {
		t.Fatalf(
			"expected requested start amount %s, got %s",
			startAmount,
			opportunity.RequestedStartAmount,
		)
	}

	expectedFillRatio := decimal.NewFromInt(1)

	if !opportunity.FillRatio.Equal(expectedFillRatio) {
		t.Fatalf(
			"expected fill ratio %s, got %s",
			expectedFillRatio,
			opportunity.FillRatio,
		)
	}

	if !opportunity.ExecutedStartAmount.Equal(startAmount) {
		t.Fatalf(
			"expected executed start amount %s, got %s",
			startAmount,
			opportunity.ExecutedStartAmount,
		)
	}

	expectedEndAmount := decimal.NewFromInt(102)

	if !opportunity.EndAmount.Equal(expectedEndAmount) {
		t.Fatalf(
			"expected end amount %s, got %s",
			expectedEndAmount,
			opportunity.EndAmount,
		)
	}

	expectedProfitAbs := decimal.NewFromInt(2)

	if !opportunity.ProfitAbs.Equal(expectedProfitAbs) {
		t.Fatalf(
			"expected profit abs %s, got %s",
			expectedProfitAbs,
			opportunity.ProfitAbs,
		)
	}

	expectedProfitPct := decimal.RequireFromString("0.02")

	if !opportunity.ProfitPct.Equal(expectedProfitPct) {
		t.Fatalf(
			"expected profit pct %s, got %s",
			expectedProfitPct,
			opportunity.ProfitPct,
		)
	}

	if opportunity.Timestamp.Before(before) ||
		opportunity.Timestamp.After(after) {
		t.Fatalf(
			"expected timestamp between %s and %s, got %s",
			before,
			after,
			opportunity.Timestamp,
		)
	}

	if result.RouteFillRatio.IsZero() ||
		!result.RouteFillRatio.Equal(opportunity.FillRatio) {
		t.Fatalf(
			"expected result route fill ratio %s to equal opportunity fill ratio %s",
			result.RouteFillRatio,
			opportunity.FillRatio,
		)
	}
}
