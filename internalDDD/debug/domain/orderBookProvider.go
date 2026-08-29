package domain

import (
	"fmt"
	"strings"
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

func (p *OrderBookProvider) Snapshot() (string, error) {
	symbols := p.store.Symbols()

	if len(symbols) == 0 {
		return "No order books loaded.", nil
	}

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"Order books: %d\n\n",
		len(symbols),
	)

	for _, symbol := range symbols {
		orderBook, ok := p.store.Get(symbol)
		if !ok {
			continue
		}

		fmt.Fprintf(
			&builder,
			"%s\n",
			symbol,
		)

		fmt.Fprintf(
			&builder,
			"  LastUpdateID: %d\n",
			orderBook.LastUpdateID,
		)

		fmt.Fprintf(
			&builder,
			"  LastUpdateTime: %s\n",
			formatTime(orderBook.LastUpdateTime),
		)

		fmt.Fprintf(
			&builder,
			"  Bids: %d\n",
			len(orderBook.Bids),
		)

		fmt.Fprintf(
			&builder,
			"  Asks: %d\n",
			len(orderBook.Asks),
		)

		if len(orderBook.Bids) > 0 {
			fmt.Fprintf(
				&builder,
				"  BestBid: %s @ %s\n",
				orderBook.Bids[0].Qty,
				orderBook.Bids[0].Price,
			)
		}

		if len(orderBook.Asks) > 0 {
			fmt.Fprintf(
				&builder,
				"  BestAsk: %s @ %s\n",
				orderBook.Asks[0].Qty,
				orderBook.Asks[0].Price,
			)
		}

		fmt.Fprintln(&builder)
	}

	return builder.String(), nil
}

func (p *OrderBookProvider) Metrics() (any, error) {
	symbols := p.store.Symbols()

	metrics := map[string]any{
		"symbols":     len(symbols),
		"orderBooks":  len(symbols),
		"bySymbol":    make(map[string]any),
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

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(time.RFC3339Nano)
}

func orderBookAge(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return time.Since(value).String()
}