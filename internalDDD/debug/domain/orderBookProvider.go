package domain

import (
	"time"
	marketDomain "traiderBot/internalDDD/market/domain"
)

type OrderBookReader interface {
	Get(symbol string) (marketDomain.OrderBook, bool)
	Symbols() []string
}

type OrderBookProvider struct {
	store OrderBookReader
}

func NewOrderBookProvider(
	store OrderBookReader,
) *OrderBookProvider {
	return &OrderBookProvider{
		store: store,
	}
}

func (p *OrderBookProvider) Name() string {
	return "OrderBookProvider"
}

func (p *OrderBookProvider) Snapshot() (any, error) {
	symbols := p.store.Symbols()

	result := map[string]any{
		"symbols": len(symbols),
	}

	books := make([]any, 0, len(symbols))

	for _, symbol := range symbols {
		orderBook, ok := p.store.Get(symbol)

		if !ok {
			continue
		}

		book := map[string]any{
			"symbol":         symbol,
			"lastUpdateID":   orderBook.LastUpdateID,
			"lastUpdateTime": formatTime(orderBook.LastUpdateTime),
			"bids":           len(orderBook.Bids),
			"asks":           len(orderBook.Asks),
		}

		if len(orderBook.Bids) > 0 {
			book["bestBid"] = map[string]any{
				"quantity": orderBook.Bids[0].Qty,
				"price":    orderBook.Bids[0].Price,
			}
		}

		if len(orderBook.Asks) > 0 {
			book["bestAsk"] = map[string]any{
				"quantity": orderBook.Asks[0].Qty,
				"price":    orderBook.Asks[0].Price,
			}
		}

		books = append(books, book)
	}

	result["orderBooks"] = books

	return result, nil
}

func (p *OrderBookProvider) Metrics() (any, error) {
	symbols := p.store.Symbols()

	metrics := map[string]any{
		"symbols":    len(symbols),
		"orderBooks": len(symbols),
		"bySymbol":   make(map[string]any),
	}

	bySymbol := metrics["bySymbol"].(map[string]any)

	for _, symbol := range symbols {
		orderBook, ok := p.store.Get(symbol)

		if !ok {
			continue
		}

		bySymbol[symbol] = map[string]any{
			"lastUpdateID":   orderBook.LastUpdateID,
			"lastUpdateTime": formatTime(orderBook.LastUpdateTime),
			"bids":           len(orderBook.Bids),
			"asks":           len(orderBook.Asks),
			"age":            orderBookAge(orderBook.LastUpdateTime),
		}
	}

	return metrics, nil
}

func orderBookAge(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return time.Since(value).String()
}
