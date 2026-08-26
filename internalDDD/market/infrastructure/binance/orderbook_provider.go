package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"traiderBot/internalDDD/market/domain"
)


type OrderBookProvider struct {
	client  *http.Client
	baseURL string
}

func NewOrderBookProvider() *OrderBookProvider {
	return &OrderBookProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
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

	reqURL := fmt.Sprintf(
		"%s/api/v3/depth?symbol=%s&limit=%d",
		p.baseURL,
		url.QueryEscape(symbol),
		depth,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"create request: %w",
			err,
		)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf(
			"request order book: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.OrderBook{}, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto orderBookDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
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
