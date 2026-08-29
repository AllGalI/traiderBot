package tests

import (
	"context"
	"errors"
	"testing"

	"traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/domain"
)

type mockMarketCache struct {
	structure domain.MarketStructure
	ok        bool
	getErr    error
	setErr    error

	getCalled bool
	setCalled bool
}

func (m *mockMarketCache) Get(
	ctx context.Context,
) (domain.MarketStructure, bool, error) {
	m.getCalled = true

	return m.structure, m.ok, m.getErr
}

func (m *mockMarketCache) Set(
	ctx context.Context,
	structure domain.MarketStructure,
) error {
	m.setCalled = true
	m.structure = structure

	return m.setErr
}

type mockMarketStructureBuilder struct {
	structure domain.MarketStructure
	err       error

	called      bool
	startAssets map[string]struct{}
}

func (m *mockMarketStructureBuilder) Build(
	ctx context.Context,
	startAssets map[string]struct{},
) (domain.MarketStructure, error) {
	m.called = true
	m.startAssets = startAssets

	if m.err != nil {
		return domain.MarketStructure{}, m.err
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

type mockMarketStructureStore struct {
	structure *domain.MarketStructure

	getCalled bool
	setCalled bool
}

func (m *mockMarketStructureStore) Get() (*domain.MarketStructure, bool) {
	m.getCalled = true

	if m.structure == nil {
		return &domain.MarketStructure{}, false
	}

	return m.structure, true
}

func (m *mockMarketStructureStore) Set(
	structure *domain.MarketStructure,
) {
	m.setCalled = true
	m.structure = structure
}

func testMarketStructure() domain.MarketStructure {
	return domain.MarketStructure{
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

func newMarketService(
	cache application.MarketCache,
	builder application.MarketStructureBuilder,
	dataService application.MarketDataStarter,
	store application.MarketStructureStore,
) *application.MarketService {
	return application.NewMarketService(
		cache,
		builder,
		dataService,
		store,
	)
}

func TestMarketServiceStartCacheHit(t *testing.T) {
	structure := testMarketStructure()

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
		10,
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

	if !store.setCalled {
		t.Fatal("expected structureStore.Set to be called")
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
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	startAssets := map[string]struct{}{
		"USDT": {},
	}

	err := service.Start(
		context.Background(),
		startAssets,
		10,
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

	if !store.setCalled {
		t.Fatal("expected structureStore.Set to be called")
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
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
		10,
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

	if store.setCalled {
		t.Fatal("structureStore.Set should not be called")
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
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
		10,
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

	if store.setCalled {
		t.Fatal("structureStore.Set should not be called")
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
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
		10,
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

	if store.setCalled {
		t.Fatal("structureStore.Set should not be called after cache.Set error")
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceStartNoRouteSymbols(t *testing.T) {
	cache := &mockMarketCache{
		structure: domain.MarketStructure{},
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
		10,
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

	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		20,
		10,
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

func TestMarketServiceStartSymbolsLimitGreaterThanAvailable(t *testing.T) {
	structure := testMarketStructure()
	structureStore := &mockMarketStructureStore{}

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}

	service := application.NewMarketService(
		cache,
		builder,
		dataService,
		structureStore,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		10,
		10,
	)

	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if len(dataService.symbols) != 3 {
		t.Fatalf(
			"expected 3 symbols, got %d",
			len(dataService.symbols),
		)
	}
}

func TestMarketServiceStartSymbolsLimitRestrictsSymbols(t *testing.T) {
	structure := testMarketStructure()

	// В RouteIndex добавляем больше символов,
	// чем разрешает symbolsLimit.
	structure.RouteIndex = domain.RouteIndex{
		"BTCUSDT": {},
		"ETHBTC":  {},
		"ETHUSDT": {},
		"BNBUSDT": {},
		"SOLUSDT": {},
	}

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		10,
		3,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dataService.symbols) != 3 {
		t.Fatalf(
			"expected 3 symbols, got %d",
			len(dataService.symbols),
		)
	}

	allowed := map[string]struct{}{
		"BTCUSDT": {},
		"ETHBTC":  {},
		"ETHUSDT": {},
		"BNBUSDT": {},
		"SOLUSDT": {},
	}

	for _, symbol := range dataService.symbols {
		if _, ok := allowed[symbol]; !ok {
			t.Fatalf(
				"unexpected symbol %q",
				symbol,
			)
		}
	}

}

func TestMarketServiceStartSymbolsLimitZero(t *testing.T) {
	structure := testMarketStructure()

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		10,
		0,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "symbols limit must be greater than zero" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceStartSymbolsLimitNegative(t *testing.T) {
	structure := testMarketStructure()

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		10,
		-1,
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "symbols limit must be greater than zero" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if dataService.called {
		t.Fatal("MarketDataService.Start should not be called")
	}
}

func TestMarketServiceReadyClosedAfterStructureLoaded(t *testing.T) {
	structure := testMarketStructure()

	cache := &mockMarketCache{
		structure: structure,
		ok:        true,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}

	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		10,
		10,
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	select {
	case <-service.Ready():
		// expected
	default:
		t.Fatal("expected Ready() channel to be closed")
	}

	stored, ok := store.Get()
	if !ok {
		t.Fatal("expected market structure to be stored")
	}

	if len(stored.Routes) != len(structure.Routes) {
		t.Fatalf(
			"expected %d routes in store, got %d",
			len(structure.Routes),
			len(stored.Routes),
		)
	}
}

func TestMarketServiceReadyNotClosedWhenCacheGetFails(t *testing.T) {
	expectedErr := errors.New("cache get error")

	cache := &mockMarketCache{
		getErr: expectedErr,
	}

	builder := &mockMarketStructureBuilder{}
	dataService := &mockMarketDataStarter{}
	store := &mockMarketStructureStore{}

	service := newMarketService(
		cache,
		builder,
		dataService,
		store,
	)

	err := service.Start(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
		10,
		10,
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

	select {
	case <-service.Ready():
		t.Fatal("Ready() should not be closed when startup fails")
	default:
		// expected
	}
}
