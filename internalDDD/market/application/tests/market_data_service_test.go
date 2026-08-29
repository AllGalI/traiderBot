package tests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/domain"
)

type fakeOrderBookProvider struct {
	orderBooks map[string]domain.OrderBook
	err        error

	calls []string
}

func (f *fakeOrderBookProvider) Get(
	ctx context.Context,
	symbol string,
	depth int,
) (domain.OrderBook, error) {
	f.calls = append(f.calls, symbol)

	if f.err != nil {
		return domain.OrderBook{}, f.err
	}

	orderBook, ok := f.orderBooks[symbol]
	if !ok {
		return domain.OrderBook{}, errors.New("order book not found")
	}

	return orderBook, nil
}

type fakeOrderBookStream struct {
	calls     int
	symbols   []string
	depth     int
	handler   application.OrderBookUpdateHandler
	returnErr error
}

func (f *fakeOrderBookStream) Subscribe(
	ctx context.Context,
	symbols []string,
	depth int,
	handler application.OrderBookUpdateHandler,
) error {
	f.calls++
	f.symbols = symbols
	f.depth = depth
	f.handler = handler

	return f.returnErr
}

type fakeOrderBookStore struct {
	orderBooks map[string]domain.OrderBook
}

func newFakeOrderBookStore() *fakeOrderBookStore {
	return &fakeOrderBookStore{
		orderBooks: make(map[string]domain.OrderBook),
	}
}

func (f *fakeOrderBookStore) Get(
	symbol string,
) (domain.OrderBook, bool) {
	orderBook, ok := f.orderBooks[symbol]
	return orderBook, ok
}

func (f *fakeOrderBookStore) Set(
	orderBook domain.OrderBook,
) {
	f.orderBooks[orderBook.Symbol] = orderBook
}

func (f *fakeOrderBookStore) Symbols() []string {
	symbols := make([]string, 0, len(f.orderBooks))

	for symbol := range f.orderBooks {
		symbols = append(symbols, symbol)
	}

	sort.Strings(symbols)

	return symbols
}

func (f *fakeOrderBookStore) Show() {
	fmt.Println("Show orderbook")
}

func TestMarketDataService_StartLoadsInitialOrderBooks(t *testing.T) {
	provider := &fakeOrderBookProvider{
		orderBooks: map[string]domain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
			"ETHUSDT": {
				Symbol:       "ETHUSDT",
				LastUpdateID: 200,
			},
		},
	}

	stream := &fakeOrderBookStream{}
	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	ctx := context.Background()

	err := service.Start(
		ctx,
		[]string{"BTCUSDT", "ETHUSDT"},
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(provider.calls) != 2 {
		t.Fatalf(
			"expected 2 provider calls, got %d",
			len(provider.calls),
		)
	}

	if stream.calls != 1 {
		t.Fatalf(
			"expected 1 stream subscription, got %d",
			stream.calls,
		)
	}

	if stream.depth != 10 {
		t.Fatalf(
			"expected depth 10, got %d",
			stream.depth,
		)
	}

	if _, ok := store.Get("BTCUSDT"); !ok {
		t.Fatal("expected BTCUSDT to be stored")
	}

	if _, ok := store.Get("ETHUSDT"); !ok {
		t.Fatal("expected ETHUSDT to be stored")
	}
}

func TestMarketDataService_StreamUpdateReplacesOrderBook(t *testing.T) {
	provider := &fakeOrderBookProvider{
		orderBooks: map[string]domain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
		},
	}

	stream := &fakeOrderBookStream{}
	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	err := service.Start(
		context.Background(),
		[]string{"BTCUSDT"},
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stream.handler == nil {
		t.Fatal("expected stream handler to be set")
	}

	stream.handler(domain.OrderBook{
		Symbol:         "BTCUSDT",
		LastUpdateID:   200,
		LastUpdateTime: time.Now(),
	})

	got, ok := store.Get("BTCUSDT")
	if !ok {
		t.Fatal("expected BTCUSDT to exist")
	}

	if got.LastUpdateID != 200 {
		t.Fatalf(
			"expected update ID 200, got %d",
			got.LastUpdateID,
		)
	}
}

func TestMarketDataService_StartProviderError(t *testing.T) {
	expectedErr := errors.New("provider failed")

	provider := &fakeOrderBookProvider{
		err: expectedErr,
	}

	stream := &fakeOrderBookStream{}
	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	err := service.Start(
		context.Background(),
		[]string{"BTCUSDT"},
		10,
	)
	if err == nil {
		t.Fatal("expected error")
	}

	if stream.calls != 0 {
		t.Fatal(
			"stream should not start when initial snapshot loading fails",
		)
	}
}

func TestMarketDataService_StartStreamError(t *testing.T) {
	expectedErr := errors.New("stream failed")

	provider := &fakeOrderBookProvider{
		orderBooks: map[string]domain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
		},
	}

	stream := &fakeOrderBookStream{
		returnErr: expectedErr,
	}

	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	err := service.Start(
		context.Background(),
		[]string{"BTCUSDT"},
		10,
	)
	if err == nil {
		t.Fatal("expected error")
	}

	if _, ok := store.Get("BTCUSDT"); !ok {
		t.Fatal(
			"initial order book should be stored before stream starts",
		)
	}
}

func TestMarketDataService_StartWithoutSymbols(t *testing.T) {
	provider := &fakeOrderBookProvider{}
	stream := &fakeOrderBookStream{}
	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	err := service.Start(
		context.Background(),
		nil,
		10,
	)
	if err == nil {
		t.Fatal("expected error")
	}

	if provider.calls != nil {
		t.Fatal("provider should not be called")
	}

	if stream.calls != 0 {
		t.Fatal("stream should not start")
	}
}

func TestMarketDataService_StartSubscribesWithRequestedSymbols(t *testing.T) {
	provider := &fakeOrderBookProvider{
		orderBooks: map[string]domain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
			"ETHUSDT": {
				Symbol:       "ETHUSDT",
				LastUpdateID: 200,
			},
		},
	}

	stream := &fakeOrderBookStream{}
	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	symbols := []string{
		"BTCUSDT",
		"ETHUSDT",
	}

	err := service.Start(
		context.Background(),
		symbols,
		20,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stream.symbols) != len(symbols) {
		t.Fatalf(
			"expected %d symbols, got %d",
			len(symbols),
			len(stream.symbols),
		)
	}

	for i, symbol := range symbols {
		if stream.symbols[i] != symbol {
			t.Fatalf(
				"expected symbol %q at index %d, got %q",
				symbol,
				i,
				stream.symbols[i],
			)
		}
	}

	if stream.depth != 20 {
		t.Fatalf(
			"expected depth 20, got %d",
			stream.depth,
		)
	}
}

func TestMarketDataService_GetOrderBook(t *testing.T) {
	provider := &fakeOrderBookProvider{
		orderBooks: map[string]domain.OrderBook{
			"BTCUSDT": {
				Symbol:       "BTCUSDT",
				LastUpdateID: 100,
			},
		},
	}

	stream := &fakeOrderBookStream{}
	store := newFakeOrderBookStore()

	service := application.NewMarketDataService(
		provider,
		stream,
		store,
	)

	err := service.Start(
		context.Background(),
		[]string{"BTCUSDT"},
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	orderBook, ok := service.GetOrderBook("BTCUSDT")
	if !ok {
		t.Fatal("expected BTCUSDT order book to exist")
	}

	if orderBook.Symbol != "BTCUSDT" {
		t.Fatalf(
			"expected symbol BTCUSDT, got %s",
			orderBook.Symbol,
		)
	}

	if orderBook.LastUpdateID != 100 {
		t.Fatalf(
			"expected update ID 100, got %d",
			orderBook.LastUpdateID,
		)
	}

	_, ok = service.GetOrderBook("ETHUSDT")
	if ok {
		t.Fatal("expected ETHUSDT order book to be absent")
	}
}
