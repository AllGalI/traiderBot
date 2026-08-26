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
	"sync/atomic"
	"time"
	"traiderBot/internal/freshness"

	"github.com/gorilla/websocket"
)

const (
	restBaseURL = "https://api.binance.com"
	wsBaseURL   = "wss://stream.binance.com:9443"
)

type PriceLevel struct {
	Price float64
	Qty   float64
}



type OrderBook struct {
	Mu             sync.RWMutex
	symbol         string
	LastUpdateID   int64
	LastUpdateTime time.Time
	Asks           []PriceLevel
	Bids           []PriceLevel
	depth          int
}

func (ob *OrderBook) GetBestBid() (PriceLevel, bool) {
	ob.Mu.RLock()
	defer ob.Mu.RUnlock()
	if len(ob.Bids) == 0 {
		return PriceLevel{}, false
	}
	return ob.Bids[0], true
}

func (ob *OrderBook) GetBestAsk() (PriceLevel, bool) {
	ob.Mu.RLock()
	defer ob.Mu.RUnlock()
	if len(ob.Asks) == 0 {
		return PriceLevel{}, false
	}
	return ob.Asks[0], true
}

func (ob *OrderBook) Symbol() string {
	return ob.symbol
}

// applyPartialSnapshot полностью заменяет топ-N
func (ob *OrderBook) applyPartialSnapshot(lastUpdateID int64, bids, asks []PriceLevel) {
	ob.Mu.Lock()
	defer ob.Mu.Unlock()

	ob.Bids = bids
	ob.Asks = asks
	ob.LastUpdateID = lastUpdateID
	ob.LastUpdateTime = time.Now()
}
func (ob *OrderBook) Age() time.Duration {
	ob.Mu.RLock()
	defer ob.Mu.RUnlock()

	return time.Since(ob.LastUpdateTime)
}

type Market struct {
	mu             sync.RWMutex
	orderBooks     map[string]*OrderBook
	depth          int
	updatesCounter atomic.Int64

	// управление WS
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// можно будет использовать потом для маршрутов
	onUpdate         func(symbol string)
	FreshnessTracker *freshness.FreshnessTracker
}

func (m *Market) CalculateSAV(ustdStartAmount float64, startAmounts map[string]struct{}) map[string]float64 {
	result := make(map[string]float64, len(startAmounts))

	// USDT является базовой валютой расчёта.
	result["USDT"] = ustdStartAmount

	for symbol := range startAmounts {
		if symbol == "USDT" {
			continue
		}

		// Например:
		// BTC -> BTCUSDT
		// ETH -> ETHUSDT
		// BNB -> BNBUSDT
		// XRP -> XRPUSDT
		marketSymbol := symbol + "USDT"

		ob, ok := m.GetOrderBook(marketSymbol)
		if !ok {
			continue
		}

		priceLevel, ok := ob.GetBestBid()

		if !ok || priceLevel.Price <= 0 {
			continue
		}

		// Сколько монет нужно купить на $100.
		result[symbol] = ustdStartAmount / priceLevel.Price
	}

	return result
}

func (m *Market) GetOrderbooks() map[string]*OrderBook {
	return m.orderBooks
}

func (m *Market) SetOrderbooks(orderbooks map[string]*OrderBook) {
	m.orderBooks = orderbooks
}

func (m *Market) GetUpdatesCounter() int64 {
	return m.updatesCounter.Load()
}

func NewMarket(depth int) *Market {
	if depth != 5 && depth != 10 && depth != 20 {
		depth = 10
	}

	ctx, cancel := context.WithCancel(context.Background())

	freshnessTracker := freshness.NewFreshnessTracker(freshness.DefaultConfig())
	freshness.SeedTrackerFromFile(freshnessTracker, "analysis.json")

	return &Market{
		orderBooks:       make(map[string]*OrderBook),
		depth:            depth,
		ctx:              ctx,
		cancel:           cancel,
		FreshnessTracker: freshnessTracker,
	}
}

// SetOnUpdate позволяет подписаться на обновление любой книги
func (m *Market) SetOnUpdate(fn func(symbol string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onUpdate = fn
}

type restDepthResponse struct {
	LastUpdateID int64        `json:"lastUpdateId"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

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

	return &OrderBook{
		symbol:         strings.ToUpper(symbol),
		LastUpdateID:   data.LastUpdateID,
		LastUpdateTime: time.Now(),
		Asks:           data.Asks,
		Bids:           data.Bids,
		depth:          depth,
	}, nil
}

// Start загружает книги и запускает один combined WebSocket
func (m *Market) Start(symbols []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	// 1. Загружаем REST-снимки
	for _, sym := range symbols {
		sym = strings.ToUpper(sym)
		if _, exists := m.orderBooks[sym]; exists {
			continue
		}

		ob, err := LoadOrderBook(sym, m.depth)
		if err != nil {
			return fmt.Errorf("load %s: %w", sym, err)
		}
		m.orderBooks[sym] = ob
	}

	// 2. Запускаем combined WS
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.runCombinedWS(symbols)
	}()

	return nil
}

func (m *Market) StartCombinedSymbols(symbols []string) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.runCombinedWS(symbols)
	}()
}

func (m *Market) Stop() {
	m.cancel()
	m.wg.Wait()
}

func (m *Market) GetOrderBook(symbol string) (*OrderBook, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ob, ok := m.orderBooks[strings.ToUpper(symbol)]
	return ob, ok
}

// -------------------- Combined WebSocket --------------------

type combinedEvent struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type partialDepthEvent struct {
	LastUpdateID int64        `json:"lastUpdateId"`
	Bids         []PriceLevel `json:"bids"`
	Asks         []PriceLevel `json:"asks"`
}

func (m *Market) runCombinedWS(symbols []string) {
	// Собираем combined URL один раз
	streams := make([]string, 0, len(symbols))
	for _, sym := range symbols {
		streams = append(streams, fmt.Sprintf("%s@depth%d@100ms",
			strings.ToLower(sym), m.depth))
	}

	url := fmt.Sprintf("%s/stream?streams=%s", wsBaseURL, strings.Join(streams, "/"))

	backoff := time.Second

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
		}

		err := m.connectAndListen(url)
		if err != nil {
			fmt.Printf("[combined] ws error: %v, reconnecting in %v...\n", err, backoff)
		}

		select {
		case <-m.ctx.Done():
			return
		case <-time.After(backoff):
		}

		// простой backoff с потолком
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
}

func (m *Market) connectAndListen(url string) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Сбрасываем backoff при успешном коннекте можно снаружи,
	// но для простоты пока оставляем как есть.

	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	for {
		select {
		case <-m.ctx.Done():
			return nil
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var wrapper combinedEvent
		if err := json.Unmarshal(message, &wrapper); err != nil {
			continue
		}

		// stream выглядит как "btcusdt@depth10@100ms"
		parts := strings.Split(wrapper.Stream, "@")
		if len(parts) < 1 {
			continue
		}
		symbol := strings.ToUpper(parts[0])

		var event partialDepthEvent
		if err := json.Unmarshal(wrapper.Data, &event); err != nil {
			continue
		}

		m.mu.RLock()
		ob, ok := m.orderBooks[symbol]
		onUpdate := m.onUpdate
		m.updatesCounter.Add(1)
		m.mu.RUnlock()

		if !ok {
			continue
		}

		ob.applyPartialSnapshot(event.LastUpdateID, event.Bids, event.Asks)

		if onUpdate != nil {
			now := time.Now()
			updateEvent := freshness.UpdateEvent{
				Symbol:    symbol,
				Timestamp: now,
				SeqID:     uint64(event.LastUpdateID),
			}
			heartBeatEvent := freshness.HeartbeatEvent{
				Symbol:    symbol,
				Timestamp: now,
			}
			m.FreshnessTracker.RecordUpdate(updateEvent)
			m.FreshnessTracker.RecordHeartbeat(heartBeatEvent)
			onUpdate(symbol)
		}

	}
}
