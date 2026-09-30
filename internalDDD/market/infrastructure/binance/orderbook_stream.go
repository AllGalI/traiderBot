package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"traiderBot/internalDDD/exchange/infrastructure/binance/ws"
	"traiderBot/internalDDD/market/application"
)

type OrderBookStream struct {
	client *ws.Client
}

func NewOrderBookStream(client *ws.Client) *OrderBookStream {
	return &OrderBookStream{
		client: client,
	}
}

func (s *OrderBookStream) Subscribe(
	ctx context.Context,
	symbols []string,
	depth int,
	handler application.OrderBookUpdateHandler,
) error {
	if len(symbols) == 0 {
		return fmt.Errorf("symbols are empty")
	}

	if handler == nil {
		return fmt.Errorf("order book handler is nil")
	}

	if depth != 5 && depth != 10 && depth != 20 {
		return fmt.Errorf("unsupported order book depth: %d", depth)
	}

	url := s.buildWebSocketURL(symbols, depth)

	conn, err := s.client.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer conn.Close()

	return s.listen(ctx, conn, handler)
}

type combinedEventDTO struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

func (s *OrderBookStream) buildWebSocketURL(
	symbols []string,
	depth int,
) string {
	streams := make([]string, 0, len(symbols))

	for _, symbol := range symbols {
		streams = append(
			streams,
			fmt.Sprintf(
				"%s@depth%d@100ms",
				strings.ToLower(symbol),
				depth,
			),
		)
	}

	return fmt.Sprintf(
		"%s/stream?streams=%s",
		s.client.BaseURL(),
		strings.Join(streams, "/"),
	)
}

func (s *OrderBookStream) listen(
	ctx context.Context,
	conn *ws.Conn,
	handler application.OrderBookUpdateHandler,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		message, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			return fmt.Errorf("read websocket message: %w", err)
		}

		var event combinedEventDTO

		if err := json.Unmarshal(message, &event); err != nil {
			return fmt.Errorf("decode websocket event: %w", err)
		}

		symbol, err := symbolFromStream(event.Stream)
		if err != nil {
			return err
		}

		var orderBook orderBookDTO

		if err := json.Unmarshal(event.Data, &orderBook); err != nil {
			return fmt.Errorf(
				"decode order book event for %s: %w",
				symbol,
				err,
			)
		}

		result, err := mapOrderBook(orderBook, symbol)
		if err != nil {
			return fmt.Errorf(
				"map order book for %s: %w",
				symbol,
				err,
			)
		}

		handler(result)
	}
}

func symbolFromStream(stream string) (string, error) {
	parts := strings.Split(stream, "@")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("invalid websocket stream: %q", stream)
	}

	return strings.ToUpper(parts[0]), nil
}