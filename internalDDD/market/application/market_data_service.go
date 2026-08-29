package application

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

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
	Symbols() []string
}

type MarketDataService struct {
	provider OrderBookProvider
	stream   OrderBookStream
	store    OrderBookStore

	updates chan string
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
		updates:  make(chan string, 1024),
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
		log.Printf("MarketData: requesting snapshot %s", symbol)

		started := time.Now()

		orderBook, err := s.provider.Get(
			ctx,
			symbol,
			depth,
		)

		log.Printf(
			"MarketData: snapshot %s finished in %s",
			symbol,
			time.Since(started),
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
	log.Printf(
		"MarketData: starting websocket for %d symbols",
		len(symbols),
	)

	return s.stream.Subscribe(
		ctx,
		symbols,
		depth,
		func(orderBook domain.OrderBook) {
			s.store.Set(orderBook)
			s.publishUpdate(orderBook.Symbol)
		},
	)

}

func (s *MarketDataService) GetOrderBook(
	symbol string,
) (domain.OrderBook, bool) {
	return s.store.Get(symbol)
}

func (s *MarketDataService) Updates() <-chan string {
	return s.updates
}

func (s *MarketDataService) publishUpdate(symbol string) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if symbol == "" {
		return
	}

	// Канал является сигналом, что symbol изменился.
	//
	// Сам актуальный OrderBook уже находится в store.
	//
	// Если worker временно не успевает обработать событие,
	// не блокируем WebSocket. При следующем событии symbol
	// worker снова получит актуальное состояние из store.
	select {
	case s.updates <- symbol:
	default:
		log.Printf(
			"MarketData: updates channel full, dropping update %s", symbol,
		)
	}

}
