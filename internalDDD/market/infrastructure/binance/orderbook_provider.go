package binance

import (
	"context"
	"encoding/json"
	"fmt"

	binanceRest "traiderBot/internalDDD/exchange/infrastructure/binance/rest"
	"traiderBot/internalDDD/market/domain"
)

type OrderBookProvider struct {
	client *binanceRest.Client
}

func NewOrderBookProvider(
	client *binanceRest.Client,
) *OrderBookProvider {
	return &OrderBookProvider{
		client: client,
	}
}

func (p *OrderBookProvider) Get(
	ctx context.Context,
	symbol string,
	depth int,
) (domain.OrderBook, error) {
	if depth != 5 && depth != 10 && depth != 20 {
		return domain.OrderBook{}, fmt.Errorf(
			"unsupported order book depth: %d",
			depth,
		)
	}

	body, err := p.client.Request(
		ctx,
		"GET",
		orderBookPath,
		map[string][]string{
			"symbol": {symbol},
			"limit":  {fmt.Sprintf("%d", depth)},
		},
	)

	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"request order book: %w",
			err,
		)
	}

	var dto orderBookDTO

	if err := json.Unmarshal(body, &dto); err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"decode order book response: %w",
			err,
		)
	}

	orderBook, err := mapOrderBook(dto, symbol)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"map order book: %w",
			err,
		)
	}

	return orderBook, nil
}