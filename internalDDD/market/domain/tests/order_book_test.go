package tests

import (
	"testing"

	"github.com/shopspring/decimal"

	"traiderBot/internalDDD/market/domain"
)

func TestOrderBookGetBestBid(t *testing.T) {
	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []domain.PriceLevel{
			{
				Price: decimal.RequireFromString("50000"),
				Qty:   decimal.RequireFromString("0.5"),
			},
			{
				Price: decimal.RequireFromString("49900"),
				Qty:   decimal.RequireFromString("1.0"),
			},
		},
	}

	priceLevel, ok := orderBook.GetBestBid()

	if !ok {
		t.Fatal("expected best bid to exist")
	}

	if !priceLevel.Price.Equal(decimal.RequireFromString("50000")) {
		t.Fatalf(
			"expected best bid price 50000, got %s",
			priceLevel.Price,
		)
	}

	if !priceLevel.Qty.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf(
			"expected best bid quantity 0.5, got %s",
			priceLevel.Qty,
		)
	}
}

func TestOrderBookGetBestAsk(t *testing.T) {
	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []domain.PriceLevel{
			{
				Price: decimal.RequireFromString("50100"),
				Qty:   decimal.RequireFromString("0.4"),
			},
			{
				Price: decimal.RequireFromString("50200"),
				Qty:   decimal.RequireFromString("0.8"),
			},
		},
	}

	priceLevel, ok := orderBook.GetBestAsk()

	if !ok {
		t.Fatal("expected best ask to exist")
	}

	if !priceLevel.Price.Equal(decimal.RequireFromString("50100")) {
		t.Fatalf(
			"expected best ask price 50100, got %s",
			priceLevel.Price,
		)
	}

	if !priceLevel.Qty.Equal(decimal.RequireFromString("0.4")) {
		t.Fatalf(
			"expected best ask quantity 0.4, got %s",
			priceLevel.Qty,
		)
	}
}

func TestOrderBookGetBestBidEmpty(t *testing.T) {
	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
		Bids:   nil,
	}

	priceLevel, ok := orderBook.GetBestBid()

	if ok {
		t.Fatal("expected no best bid for empty order book")
	}

	if priceLevel != (domain.PriceLevel{}) {
		t.Fatalf(
			"expected zero price level, got %+v",
			priceLevel,
		)
	}
}

func TestOrderBookGetBestAskEmpty(t *testing.T) {
	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
		Asks:   nil,
	}

	priceLevel, ok := orderBook.GetBestAsk()

	if ok {
		t.Fatal("expected no best ask for empty order book")
	}

	if priceLevel != (domain.PriceLevel{}) {
		t.Fatalf(
			"expected zero price level, got %+v",
			priceLevel,
		)
	}
}

func TestOrderBookGetBestBidUsesFirstLevel(t *testing.T) {
	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
		Bids: []domain.PriceLevel{
			{
				Price: decimal.RequireFromString("50000"),
				Qty:   decimal.RequireFromString("0.5"),
			},
			{
				Price: decimal.RequireFromString("51000"),
				Qty:   decimal.RequireFromString("1.0"),
			},
		},
	}

	priceLevel, ok := orderBook.GetBestBid()

	if !ok {
		t.Fatal("expected best bid to exist")
	}

	if !priceLevel.Price.Equal(decimal.RequireFromString("50000")) {
		t.Fatalf(
			"expected first bid level to be returned, got %s",
			priceLevel.Price,
		)
	}
}

func TestOrderBookGetBestAskUsesFirstLevel(t *testing.T) {
	orderBook := domain.OrderBook{
		Symbol: "BTCUSDT",
		Asks: []domain.PriceLevel{
			{
				Price: decimal.RequireFromString("50100"),
				Qty:   decimal.RequireFromString("0.4"),
			},
			{
				Price: decimal.RequireFromString("50000"),
				Qty:   decimal.RequireFromString("1.0"),
			},
		},
	}

	priceLevel, ok := orderBook.GetBestAsk()

	if !ok {
		t.Fatal("expected best ask to exist")
	}

	if !priceLevel.Price.Equal(decimal.RequireFromString("50100")) {
		t.Fatalf(
			"expected first ask level to be returned, got %s",
			priceLevel.Price,
		)
	}
}
