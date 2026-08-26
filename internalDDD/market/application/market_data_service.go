package application

import (
	"context"
	"fmt"

	"traiderBot/internalDDD/market/domain"
)

type OrderBookProvider interface {
	Get(
		ctx context.Context,
		symbol string,
		depth int,
	) (domain.OrderBook, error)
}

type OrderBookStore interface {
	Get(symbol string) (domain.OrderBook, bool)
	Set(orderBook domain.OrderBook)
	Show()
}

type MarketDataService struct {
	provider OrderBookProvider
	stream   OrderBookStream
	store    OrderBookStore
}

func NewMarketDataService(
	provider OrderBookProvider,
	stream OrderBookStream,
	store OrderBookStore,
) *MarketDataService {
	return &MarketDataService{
		provider: provider,
		stream:   stream,
		store:    store,
	}
}

func (s *MarketDataService) Start(
	ctx context.Context,
	symbols []string,
	depth int,
) error {
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	// Сначала получаем первоначальные snapshot через REST.
	for _, symbol := range symbols {
		orderBook, err := s.provider.Get(
			ctx,
			symbol,
			depth,
		)
		if err != nil {
			return fmt.Errorf(
				"load order book %s: %w",
				symbol,
				err,
			)
		}

		s.store.Set(orderBook)
	}

	// После первоначальной загрузки начинаем получать
	// обновления через WebSocket.
	return s.stream.Subscribe(
		ctx,
		symbols,
		depth,
		func(orderBook domain.OrderBook) {
			s.store.Set(orderBook)
		},
	)
}

func (s *MarketDataService) GetOrderBook(
	symbol string,
) (domain.OrderBook, bool) {
	return s.store.Get(symbol)
}

func (s *MarketDataService) ShowStore() {
	s.store.Show()
}
