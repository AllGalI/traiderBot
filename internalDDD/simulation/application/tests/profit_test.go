package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	marketDomain "traiderBot/internalDDD/market/domain"
	simulationApplication "traiderBot/internalDDD/simulation/application"
	simulationDomain "traiderBot/internalDDD/simulation/domain"
)

func TestSimulationService_ProfitCalculation_FullFill(t *testing.T) {

	store := &orderBookReaderStub{
		orderBooks: map[string]marketDomain.OrderBook{
			// 100 USDT -> 0.001 BTC
			"BTCUSDT": newOrderBook("BTCUSDT", "100000", "0.001"),

			// 0.001 BTC -> 0.02 ETH
			"ETHBTC": newOrderBook("ETHBTC", "0.05", "1"),

			// 0.02 ETH -> 102 USDT
			"ETHUSDT": newOrderBook("ETHUSDT", "5100", "1"),
		},
	}

	fees := &feeProviderStub{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate:  decimal.Zero,
				Asset: "BTC",
			},
			"ETHBTC": {
				Rate:  decimal.Zero,
				Asset: "ETH",
			},
			"ETHUSDT": {
				Rate:  decimal.Zero,
				Asset: "USDT",
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
		decimal.RequireFromString("0.3"),
		decimal.RequireFromString("0.01"),
	)

	result := service.Simulate(
		triangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK, got %v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity")
	}

	expectedExecuted := decimal.NewFromInt(100)

	if !result.Opportunity.ExecutedStartAmount.Equal(expectedExecuted) {
		t.Fatalf(
			"expected executed start amount %s, got %s",
			expectedExecuted,
			result.Opportunity.ExecutedStartAmount,
		)
	}

	expectedEnd := decimal.NewFromInt(102)

	if !result.Opportunity.EndAmount.Equal(expectedEnd) {
		t.Fatalf(
			"expected end amount %s, got %s",
			expectedEnd,
			result.Opportunity.EndAmount,
		)
	}

	expectedProfitAbs := decimal.NewFromInt(2)

	if !result.Opportunity.ProfitAbs.Equal(expectedProfitAbs) {
		t.Fatalf(
			"expected profit abs %s, got %s",
			expectedProfitAbs,
			result.Opportunity.ProfitAbs,
		)
	}

	expectedProfitPct := decimal.RequireFromString("0.02")

	if !result.Opportunity.ProfitPct.Equal(expectedProfitPct) {
		t.Fatalf(
			"expected profit pct %s, got %s",
			expectedProfitPct,
			result.Opportunity.ProfitPct,
		)
	}
}

func TestSimulationService_ProfitCalculation_PartialFill(t *testing.T) {

	store := &orderBookReaderStub{
		orderBooks: map[string]marketDomain.OrderBook{
			// Из 100 USDT исполняется только 60 USDT.
			"BTCUSDT": newOrderBook("BTCUSDT", "100000", "0.0006"),

			// 0.0006 BTC -> 0.012 ETH.
			"ETHBTC": newOrderBook("ETHBTC", "0.05", "1"),

			// 0.012 ETH -> 60.6 USDT.
			"ETHUSDT": newOrderBook("ETHUSDT", "5050", "1"),
		},
	}

	fees := &feeProviderStub{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate:  decimal.Zero,
				Asset: "BTC",
			},
			"ETHBTC": {
				Rate:  decimal.Zero,
				Asset: "ETH",
			},
			"ETHUSDT": {
				Rate:  decimal.Zero,
				Asset: "USDT",
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
		decimal.RequireFromString("0.3"),
		decimal.RequireFromString("0.005"),
	)

	result := service.Simulate(
		triangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK, got %v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity")
	}

	expectedFillRatio := decimal.RequireFromString("0.6")

	if !result.Opportunity.FillRatio.Equal(expectedFillRatio) {
		t.Fatalf(
			"expected fill ratio %s, got %s",
			expectedFillRatio,
			result.Opportunity.FillRatio,
		)
	}

	expectedExecuted := decimal.NewFromInt(60)

	if !result.Opportunity.ExecutedStartAmount.Equal(expectedExecuted) {
		t.Fatalf(
			"expected executed start amount %s, got %s",
			expectedExecuted,
			result.Opportunity.ExecutedStartAmount,
		)
	}

	expectedEnd := decimal.RequireFromString("60.6")

	if !result.Opportunity.EndAmount.Equal(expectedEnd) {
		t.Fatalf(
			"expected end amount %s, got %s",
			expectedEnd,
			result.Opportunity.EndAmount,
		)
	}

	// Прибыль считается относительно фактически
	// исполненного объёма:
	//
	// 60.6 - 60 = 0.6 USDT.
	expectedProfitAbs := decimal.RequireFromString("0.6")

	if !result.Opportunity.ProfitAbs.Equal(expectedProfitAbs) {
		t.Fatalf(
			"expected profit abs %s, got %s",
			expectedProfitAbs,
			result.Opportunity.ProfitAbs,
		)
	}

	// 0.6 / 60 = 0.01 = 1%.
	expectedProfitPct := decimal.RequireFromString("0.01")

	if !result.Opportunity.ProfitPct.Equal(expectedProfitPct) {
		t.Fatalf(
			"expected profit pct %s, got %s",
			expectedProfitPct,
			result.Opportunity.ProfitPct,
		)
	}
}

func TestSimulationService_ProfitTooLow(t *testing.T) {

	store := &orderBookReaderStub{
		orderBooks: map[string]marketDomain.OrderBook{
			// 100 USDT -> 0.001 BTC.
			"BTCUSDT": newOrderBook("BTCUSDT", "100000", "0.001"),

			// 0.001 BTC -> 0.02 ETH.
			"ETHBTC": newOrderBook("ETHBTC", "0.05", "1"),

			// 0.02 ETH -> 100.2 USDT.
			"ETHUSDT": newOrderBook("ETHUSDT", "5010", "1"),
		},
	}

	fees := &feeProviderStub{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate:  decimal.Zero,
				Asset: "BTC",
			},
			"ETHBTC": {
				Rate:  decimal.Zero,
				Asset: "ETH",
			},
			"ETHUSDT": {
				Rate:  decimal.Zero,
				Asset: "USDT",
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

	// Фактическая прибыль:
	//
	// 100.2 - 100 = 0.2
	//
	// ProfitPct = 0.2 / 100 = 0.002 = 0.2%.
	//
	// Требуем минимум 0.5%.
	service := simulationApplication.NewSimulationService(
		store,
		fees,
		symbols,
		decimal.RequireFromString("0.3"),
		decimal.RequireFromString("0.005"),
	)

	result := service.Simulate(
		triangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationProfitTooLow {
		t.Fatalf(
			"expected SimulationProfitTooLow, got %v",
			result.Reason,
		)
	}

	if result.Opportunity != nil {
		t.Fatal("expected no opportunity when profit is too low")
	}

	// Даже при отклонении маршрута мы должны знать,
	// какой fill ratio был рассчитан.
	expectedFillRatio := decimal.NewFromInt(1)

	if !result.RouteFillRatio.Equal(expectedFillRatio) {
		t.Fatalf(
			"expected route fill ratio %s, got %s",
			expectedFillRatio,
			result.RouteFillRatio,
		)
	}
}

func TestSimulationService_ProfitThresholdBoundary(t *testing.T) {

	store := &orderBookReaderStub{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT": newOrderBook("BTCUSDT", "100000", "0.001"),
			"ETHBTC":  newOrderBook("ETHBTC", "0.05", "1"),

			// 0.02 ETH * 5025 = 100.5 USDT.
			"ETHUSDT": newOrderBook("ETHUSDT", "5025", "1"),
		},
	}

	fees := &feeProviderStub{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate:  decimal.Zero,
				Asset: "BTC",
			},
			"ETHBTC": {
				Rate:  decimal.Zero,
				Asset: "ETH",
			},
			"ETHUSDT": {
				Rate:  decimal.Zero,
				Asset: "USDT",
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

	// Прибыль ровно 0.5%.
	minProfitPct := decimal.RequireFromString("0.005")

	service := simulationApplication.NewSimulationService(
		store,
		fees,
		symbols,
		decimal.RequireFromString("0.3"),
		minProfitPct,
	)

	result := service.Simulate(
		triangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected SimulationOK at profit threshold, got %v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity at exact profit threshold")
	}

	expectedProfitPct := decimal.RequireFromString("0.005")

	if !result.Opportunity.ProfitPct.Equal(expectedProfitPct) {
		t.Fatalf(
			"expected profit pct %s, got %s",
			expectedProfitPct,
			result.Opportunity.ProfitPct,
		)
	}
}

func TestSimulationService_ProfitCalculation_UsesExecutedAmount(t *testing.T) {
	store := &orderBookReaderStub{
		orderBooks: map[string]marketDomain.OrderBook{
			"BTCUSDT":  newOrderBook("BTCUSDT", "100000", "0.0005"),
			"ETHBTC":   newOrderBook("ETHBTC", "0.05", "1"),
			"ETHUSDT":  newOrderBook("ETHUSDT", "5100", "1"),
		},
	}

	fees := &feeProviderStub{
		fees: map[string]simulationApplication.FeeConfig{
			"BTCUSDT": {
				Rate:  decimal.Zero,
				Asset: "BTC",
			},
			"ETHBTC": {
				Rate:  decimal.Zero,
				Asset: "ETH",
			},
			"ETHUSDT": {
				Rate:  decimal.Zero,
				Asset: "USDT",
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
		decimal.RequireFromString("0.30"),
		decimal.RequireFromString("0.01"),
	)

	result := service.Simulate(
		triangleRoute(),
		decimal.NewFromInt(100),
	)

	if result.Reason != simulationDomain.SimulationOK {
		t.Fatalf(
			"expected simulation OK, got %v",
			result.Reason,
		)
	}

	if result.Opportunity == nil {
		t.Fatal("expected opportunity")
	}

	opportunity := result.Opportunity

	expectedRequested := decimal.NewFromInt(100)

	if !opportunity.RequestedStartAmount.Equal(expectedRequested) {
		t.Fatalf(
			"expected requested amount %s, got %s",
			expectedRequested,
			opportunity.RequestedStartAmount,
		)
	}

	expectedExecuted := decimal.NewFromInt(50)

	if !opportunity.ExecutedStartAmount.Equal(expectedExecuted) {
		t.Fatalf(
			"expected executed amount %s, got %s",
			expectedExecuted,
			opportunity.ExecutedStartAmount,
		)
	}

	expectedEnd := decimal.NewFromInt(51)

	if !opportunity.EndAmount.Equal(expectedEnd) {
		t.Fatalf(
			"expected end amount %s, got %s",
			expectedEnd,
			opportunity.EndAmount,
		)
	}

	expectedProfitAbs := decimal.NewFromInt(1)

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
}