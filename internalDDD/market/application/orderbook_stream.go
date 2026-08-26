package application

import (
	"context"

	"traiderBot/internalDDD/market/domain"
)

type OrderBookUpdateHandler func(orderBook domain.OrderBook)

type OrderBookStream interface {
	Subscribe(
		ctx context.Context,
		symbols []string,
		depth int,
		handler OrderBookUpdateHandler,
	) error
}
