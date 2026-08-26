package infrastructure

import (
	"fmt"
	"strings"
	"sync"

	"traiderBot/internalDDD/market/domain"
)

type OrderBookStore struct {
	mu         sync.RWMutex
	orderBooks map[string]domain.OrderBook
}

func NewOrderBookStore() *OrderBookStore {
	return &OrderBookStore{
		orderBooks: make(map[string]domain.OrderBook),
	}
}

func (s *OrderBookStore) Get(symbol string) (domain.OrderBook, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderBook, ok := s.orderBooks[strings.ToUpper(symbol)]

	return orderBook, ok
}

func (s *OrderBookStore) Set(orderBook domain.OrderBook) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orderBooks[strings.ToUpper(orderBook.Symbol)] = orderBook
}

func (s *OrderBookStore) Show() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fmt.Println(s.orderBooks)

}
