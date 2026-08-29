package tests

import (
	"context"
	"testing"
	"time"

	"traiderBot/internalDDD/market/application"
	"traiderBot/internalDDD/market/domain"
)

func TestMarketDataService_StreamUpdatePublishesSymbol(t *testing.T) {
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

	select {
	case symbol := <-service.Updates():
		if symbol != "BTCUSDT" {
			t.Fatalf(
				"expected update symbol BTCUSDT, got %q",
				symbol,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("expected order book update event")
	}
}

func TestMarketDataService_StreamUpdatePublishesEachSymbol(t *testing.T) {
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

	err := service.Start(
		context.Background(),
		[]string{"BTCUSDT", "ETHUSDT"},
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream.handler(domain.OrderBook{
		Symbol:         "BTCUSDT",
		LastUpdateID:   101,
		LastUpdateTime: time.Now(),
	})

	stream.handler(domain.OrderBook{
		Symbol:         "ETHUSDT",
		LastUpdateID:   201,
		LastUpdateTime: time.Now(),
	})

	select {
	case symbol := <-service.Updates():
		if symbol != "BTCUSDT" {
			t.Fatalf(
				"expected first update BTCUSDT, got %q",
				symbol,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("expected first order book update event")
	}

	select {
	case symbol := <-service.Updates():
		if symbol != "ETHUSDT" {
			t.Fatalf(
				"expected second update ETHUSDT, got %q",
				symbol,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("expected second order book update event")
	}
}
