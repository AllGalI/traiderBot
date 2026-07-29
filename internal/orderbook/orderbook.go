package orderbook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	restBaseURL = "https://api.binance.com"
	wsBaseURL   = "wss://stream.binance.com:9443/ws"
)


type PriceLevel struct {
	Price float64
	Qty   float64
}

func (p *PriceLevel) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) < 2 {
		return fmt.Errorf("invalid price level: %s", string(data))
	}

	price, err := strconv.ParseFloat(arr[0], 64)
	if err != nil {
		return fmt.Errorf("parse price: %w", err)
	}
	qty, err := strconv.ParseFloat(arr[1], 64)
	if err != nil {
		return fmt.Errorf("parse qty: %w", err)
	}

	p.Price = price
	p.Qty = qty
	return nil
}

// OrderBook хранит текущий стакан
type OrderBook struct {
	Mu           sync.RWMutex
	symbol       string
	lastUpdateID int64
	Asks         []PriceLevel // price -> qty
	Bids         []PriceLevel // price -> qty
	depth        int               // 5 / 10 / 20
}

// Market управляет несколькими стаканами
type Market struct {
	mu         sync.RWMutex
	orderBooks map[string]*OrderBook
	depth      int // глубина по умолчанию для всех книг
	cancel     context.CancelFunc
}

// -------------------- Конструкторы --------------------

func NewMarket(depth int) *Market {
	if depth != 5 && depth != 10 && depth != 20 {
		depth = 10 // безопасный дефолт
	}
	return &Market{
		orderBooks: make(map[string]*OrderBook),
		depth:      depth,
	}
}

func newOrderBook(symbol string, data restDepthResponse, depth int) *OrderBook {
	return &OrderBook{
		symbol: strings.ToUpper(symbol),
		lastUpdateID: data.LastUpdateID,
		Asks:   data.Asks,
		Bids:   data.Bids,
		depth:  depth,
	}
}

// -------------------- REST: первоначальная загрузка --------------------

type restDepthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

// LoadOrderBook загружает снимок стакана через REST
func LoadOrderBook(symbol string, depth int) (*OrderBook, error) {
	if depth != 5 && depth != 10 && depth != 20 {
		depth = 10
	}

	url := fmt.Sprintf("%s/api/v3/depth?symbol=%s&limit=%d",
		restBaseURL, strings.ToUpper(symbol), depth)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("rest depth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance rest error %d: %s", resp.StatusCode, string(body))
	}

	var data restDepthResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode rest depth: %w", err)
	}

	ob := newOrderBook(symbol, data, depth)

	return ob, nil
}

// LoadMarket создаёт Market и загружает несколько символов через REST
func LoadMarket(symbols []string, depth int) (*Market, error) {
	m := NewMarket(depth)

	for _, sym := range symbols {
		ob, err := LoadOrderBook(sym, depth)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", sym, err)
		}
		m.orderBooks[strings.ToUpper(sym)] = ob
	}

	return m, nil
}

// -------------------- WebSocket: Partial Book Depth --------------------

// partialDepthEvent — то, что приходит в @depthN@100ms
type partialDepthEvent struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

// UpdateOrderBook запускает WebSocket Partial Book Depth
// и постоянно обновляет локальный стакан.
// Возвращает указатель на OrderBook (уже существующий в Market или новый).
func (m *Market) UpdateOrderBook(symbol string) (*OrderBook, error) {
	symbol = strings.ToUpper(symbol)

	m.mu.Lock()
	ob, exists := m.orderBooks[symbol]
	if !exists {
		// если книги ещё нет — создаём и сразу грузим REST-снимок
		var err error
		ob, err = LoadOrderBook(symbol, m.depth)
		if err != nil {
			m.mu.Unlock()
			return nil, err
		}
		m.orderBooks[symbol] = ob
	}
	m.mu.Unlock()

	// запускаем WS в фоне
	go m.runPartialDepthWS(ob)

	return ob, nil
}

// runPartialDepthWS — основной цикл WebSocket
func (m *Market) runPartialDepthWS(ob *OrderBook) {
	stream := fmt.Sprintf("%s@depth%d",
		strings.ToLower(ob.symbol), ob.depth)

	url := fmt.Sprintf("%s/%s", wsBaseURL, stream)

	for {
		if err := m.connectAndListen(ob, url); err != nil {
			fmt.Printf("[%s] ws error: %v, reconnecting in 3s...\n", ob.symbol, err)
			time.Sleep(3 * time.Second)
			continue
		}
		// если вышли без ошибки — тоже переподключаемся
		time.Sleep(1 * time.Second)
	}
}

func (m *Market) connectAndListen(ob *OrderBook, url string) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Binance требует pong на ping
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var event partialDepthEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue // пропускаем мусор
		}

		ob.applyPartialSnapshot(event)
	}
}

// applyPartialSnapshot полностью заменяет топ-N уровней
func (ob *OrderBook) applyPartialSnapshot(event partialDepthEvent) {
	ob.Mu.Lock()
	defer ob.Mu.Unlock()

	// полностью очищаем и заполняем заново (Partial всегда полный топ)
	ob.Bids = event.Bids
	ob.Asks = event.Asks
	ob.lastUpdateID = event.LastUpdateID
}

// -------------------- Вспомогательные методы --------------------

// GetBestBid возвращает лучшую цену покупки и объём
func (ob *OrderBook) GetBestBid() PriceLevel {
	ob.Mu.RLock()
	defer ob.Mu.RUnlock()
	return ob.Bids[0]
}

// GetBestAsk возвращает лучшую цену продажи и объём
func (ob *OrderBook) GetBestAsk() PriceLevel {
	ob.Mu.RLock()
	defer ob.Mu.RUnlock()

	return ob.Asks[0]
}


// GetOrderBook возвращает книгу по символу
func (m *Market) GetOrderBook(symbol string) (*OrderBook, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ob, ok := m.orderBooks[strings.ToUpper(symbol)]
	return ob, ok
}
