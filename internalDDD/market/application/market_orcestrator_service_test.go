package application

import (
	"context"
	"errors"
	"testing"

	"traiderBot/internalDDD/market/domain"
)

type mockMarketCache struct {
	structure MarketStructure
	ok        bool
	getErr    error
	setErr    error

	getCalled bool
	setCalled bool
}

func (m *mockMarketCache) Get(
	ctx context.Context,
) (MarketStructure, bool, error) {
	m.getCalled = true

	return m.structure, m.ok, m.getErr
}

func (m *mockMarketCache) Set(
	ctx context.Context,
	structure MarketStructure,
) error {
	m.setCalled = true
	m.structure = structure

	return m.setErr
}

type mockMarketStructureBuilder struct {
	structure MarketStructure
	err       error

	called      bool
	startAssets map[string]struct{}
}

func (m *mockMarketStructureBuilder) Build(
	ctx context.Context,
	startAssets map[string]struct{},
) (MarketStructure, error) {
	m.called = true
	m.startAssets = startAssets

	if m.err != nil {
		return MarketStructure{}, m.err
	}

	return m.structure, nil
}

type mockMarketDataStarter struct {
	called  bool
	symbols []string
	depth   int
	err     error
}

func (m *mockMarketDataStarter) Start(
	ctx context.Context,
	symbols []string,
	depth int,
) error {
	m.called = true
	m.symbols = symbols
	m.depth = depth

	return m.err
}

func testMarketStructure() MarketStructure {
	return MarketStructure{
		Symbols: []domain.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
			},
			{
				Name:       "ETHBTC",
				BaseAsset:  "ETH",
				QuoteAsset: "BTC",
			},
			{
				Name:       "ETHUSDT",
				BaseAsset:  "ETH",
				QuoteAsset: "USDT",
			},
		},
		RouteIndex: domain.RouteIndex{
			"BTCUSDT": {},
			"ETHBTC":  {},
			"ETHUSDT": {},
		},
	}
}

func TestMarketServiceStartCacheHit(t *testing.T) {
	structure := testMarketStructure()

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}

	dataService := &mockMarketDataStarter{}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !cache.getCalled {
		t.Fatal("expected cache.Get to be called")
	}

	if cache.setCalled {
		t.Fatal("cache.Set should not be called on cache hit")
	}

	if builder.called {
		t.Fatal("MarketStructureService should not be called on cache hit")
	}

	if !dataService.called {
		t.Fatal("expected MarketDataService.Start to be called")
	}

	if dataService.depth != 20 {
		t.Fatalf(
			"expected depth 20, got %d",
			dataService.depth,
		)
	}

	if len(dataService.symbols) != 3 {
		t.Fatalf(
			"expected 3 symbols, got %d",
			len(dataService.symbols),
		)
	}
}

func TestMarketServiceStartCacheMiss(t *testing.T) {
	structure := testMarketStructure()

	cache := &mockMarketCache{
		ok: false,
	}

	builder := &mockMarketStructureBuilder{
		structure: structure,
	}

	dataService := &mockMarketDataStarter{}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	startAssets := map[string]struct{}{
		"USDT": {},
	}

	err := service.Start(
		context.Background(),
		startAssets,
		10,
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !cache.getCalled {
		t.Fatal("expected cache.Get to be called")
	}

	if !builder.called {
		t.Fatal("expected MarketStructureService.Build to be called")
	}

	if !cache.setCalled {
		t.Fatal("expected cache.Set to be called")
	}

	if !dataService.called {
		t.Fatal("expected MarketDataService.Start to be called")
	}

	if len(dataService.symbols) != 3 {
		t.Fatalf(
			"expected 3 symbols, got %d",
			len(dataService.symbols),
		)
	}

	if dataService.depth != 10 {
		t.Fatalf(
			"expected depth 10, got %d",
			dataService.depth,
		)
	}
}

func TestMarketServiceStartCacheGetError(t *testing.T) {
	expectedErr := errors.New("cache get error")

	cache := &mockMarketCache{
		getErr: expectedErr,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped cache error, got %v",
			err,
		)
	}

	if builder.called {
		t.Fatal("MarketStructureService should not be called")
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceStartBuildError(t *testing.T) {
	expectedErr := errors.New("build error")

	cache := &mockMarketCache{
		ok: false,
	}

	builder := &mockMarketStructureBuilder{
		err: expectedErr,
	}

	dataService := &mockMarketDataStarter{}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped build error, got %v",
			err,
		)
	}

	if cache.setCalled {
		t.Fatal("cache.Set should not be called after build error")
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceStartCacheSetError(t *testing.T) {
	expectedErr := errors.New("cache set error")

	cache := &mockMarketCache{
		ok:     false,
		setErr: expectedErr,
	}

	builder := &mockMarketStructureBuilder{
		structure: testMarketStructure(),
	}

	dataService := &mockMarketDataStarter{}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped cache set error, got %v",
			err,
		)
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceStartNoRouteSymbols(t *testing.T) {
	cache := &mockMarketCache{
		structure: MarketStructure{},
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceStartMarketDataError(t *testing.T) {
	expectedErr := errors.New("market data error")

	cache := &mockMarketCache{
		structure: testMarketStructure(),
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}

	dataService := &mockMarketDataStarter{
		err: expectedErr,
	}

	service := NewMarketService(
		cache,
		builder,
		dataService,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped market data error, got %v",
			err,
		)
	}
}
