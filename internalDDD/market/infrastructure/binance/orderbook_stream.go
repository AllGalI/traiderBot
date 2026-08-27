package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"traiderBot/internalDDD/market/application"
)

type OrderBookStream struct {
}

func NewOrderBookStream() *OrderBookStream {
	return &OrderBookStream{}
}

func (s *OrderBookStream) Subscribe(
	ctx context.Context,
	symbols []string,
	depth int,
	handler application.OrderBookUpdateHandler,
) error {
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	if handler == nil {
		return fmt.Errorf("update handler is nil")
	}

	if depth != 5 && depth != 10 && depth != 20 {
		return fmt.Errorf("unsupported order book depth: %d", depth)
	}

	url := buildWebSocketURL(symbols, depth)

	return s.connectAndListen(ctx, url, handler)
}

type combinedEventDTO struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

func buildWebSocketURL(symbols []string, depth int) string {
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
		wsBaseURL,
		strings.Join(streams, "/"),
	)
}

func (s *OrderBookStream) connectAndListen(
	ctx context.Context,
	url string,
	handler application.OrderBookUpdateHandler,
) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	log.Printf("WebSocket: dialing %s", url)

	conn, _, err := dialer.DialContext(ctx, url, nil)

	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	log.Println("WebSocket: connected")

	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read websocket message: %w", err)
		}

		var event combinedEventDTO

		if err := json.Unmarshal(message, &event); err != nil {
			return fmt.Errorf("decode combined event: %w", err)
		}

		symbol, err := symbolFromStream(event.Stream)
		if err != nil {
			return fmt.Errorf("parse stream symbol: %w", err)
		}

		var dto orderBookDTO

		if err := json.Unmarshal(event.Data, &dto); err != nil {
			return fmt.Errorf(
				"decode order book update for %s: %w",
				symbol,
				err,
			)
		}

		orderBook, err := mapOrderBook(dto, symbol)
		if err != nil {
			return fmt.Errorf(
				"map order book update for %s: %w",
				symbol,
				err,
			)
		}

		handler(orderBook)
	}
}

func symbolFromStream(stream string) (string, error) {
	parts := strings.Split(stream, "@")

	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("invalid stream: %q", stream)
	}

	return strings.ToUpper(parts[0]), nil
}
