package ws

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	binance "traiderBot/internalDDD/exchange/infrastructure/binance"
)

const defaultHandshakeTimeout = 10 * time.Second

type Client struct {
	baseURL string
	dialer  *websocket.Dialer
}

func NewClient(cfg binance.Config) *Client {
	return &Client{
		baseURL: cfg.WSBaseURL,
		dialer: &websocket.Dialer{
			HandshakeTimeout: defaultHandshakeTimeout,
		},
	}
}

func (c *Client) Connect(ctx context.Context, endpoint string) (*Conn, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("websocket endpoint is empty")
	}

	conn, _, err := c.dialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	result := &Conn{
		conn: conn,
	}

	go func() {
		<-ctx.Done()
		_ = result.Close()
	}()

	return result, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

type Conn struct {
	conn *websocket.Conn
}

func (c *Conn) ReadMessage() ([]byte, error) {
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (c *Conn) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}