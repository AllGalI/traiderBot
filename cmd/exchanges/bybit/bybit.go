package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"traiderBot/cmd/domain"
	"traiderBot/cmd/filesHandler"

	bybit "github.com/bybit-exchange/bybit.go.api"
	"github.com/gorilla/websocket"
)

var (
	pairsFile = "pairs.json"
	baseUrl   = "https://api-testnet.bybit.com/v5"
	baseWSUrl = "wss://stream.bybit.com/v5/public/spot"
)

type TickerResult struct {
	Category string   `json:"category"`
	List     []Ticker `json:"list"`
}

type Ticker struct {
	Symbol string `json:"symbol"`
}

func FetchSymbols() error {
	client := bybit.NewBybitHttpClient("", "", bybit.WithBaseURL(bybit.TESTNET))
	existedTicks, err := filesHandler.ReadSymbols()
	if err != nil {
		return err
	}

	deleteMap := make(map[string]bool)
	for _, tick := range existedTicks {
		deleteMap[tick] = false
	}

	for _, tick := range existedTicks {
		params := map[string]any{"category": "spot", "symbol": tick}
		_, err := client.NewUtaBybitServiceWithParams(params).GetInstrumentInfo(context.Background())

		if err != nil {
			deleteMap[tick] = true
		}

	}

	filtered := make([]string, 0)

	for _, tick := range existedTicks {
		if !deleteMap[tick] {
			filtered = append(filtered, tick)
		}
	}

	err = filesHandler.SaveSymbols(filtered)
	if err != nil {
		return err
	}
	return nil
}

func fetchVolume(symbol string, limit int) (domain.Data, error) {
	resp, err := http.Get(baseUrl + fmt.Sprintf("/market/orderbook?category=spot&symbol=%s&limit=%d", symbol, limit))
	if err != nil {
		return domain.Data{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Data{}, fmt.Errorf("status: %s", resp.Status)
	}

	var result BybitOrderBookResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.Data{}, err
	}

	return convertToDomainData(result.BybitOrderBookData), nil
}

func convertToDomainData(src BybitOrderBookData) domain.Data {

	data := domain.Data{
		LastUpdateId:   src.LastUpdateId,
		Bids:           make(map[string]float64),
		Asks:           make(map[string]float64),
		LastUpdateTime: time.Now().UnixMilli(),
	}

	for _, b := range src.Bids {
		if price, qty := b[0], b[1]; qty != "0" && qty != "0.00000000" {
			data.Bids[price], _ = strconv.ParseFloat(qty, 64)
		}
	}

	for _, a := range src.Asks {
		if price, qty := a[0], a[1]; qty != "0" && qty != "0.00000000" {
			data.Asks[price], _ = strconv.ParseFloat(qty, 64)
		}
	}

	return data
}

func FetchVolumes(symbols []string, exchange string, limit int) (domain.OrderBook, error) {
	result := domain.OrderBook{}
	result.Exchange = exchange
	for _, symbol := range symbols {
		volume, err := fetchVolume(symbol, limit)
		if err != nil {
			return domain.OrderBook{}, err
		}
		s := domain.Symbol{Name: symbol, Data: volume}
		result.Symbols = append(result.Symbols, s)

	}

	return result, nil
}

// ==================== WEBSOCKET ====================

type BybitDepthEvent struct {
	Topic string `json:"topic"`
	Type  string `json:"type"` // "snapshot" или "delta"
	Data  struct {
		Symbol    string     `json:"s"`
		Bids      [][]string `json:"b"`
		Asks      [][]string `json:"a"`
		UpdateID  int64      `json:"u"` // seq для проверки
		Timestamp int64      `json:"ts"`
	} `json:"data"`
	Ts int64 `json:"ts"`
}

func ConnectOrderBooks(orderBook *domain.OrderBook) {
	if len(orderBook.Symbols) == 0 {
		return
	}

	go connectCombinedStream(orderBook)
}

func connectCombinedStream(orderBook *domain.OrderBook) {
	conn, _, err := websocket.DefaultDialer.Dial("wss://stream.bybit.com/v5/public/spot", nil)
	if err != nil {
		fmt.Printf("[Bybit WS] Ошибка подключения: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("[Bybit] WebSocket подключён (%d пар)\n", len(orderBook.Symbols))

	// === Подписываемся на несколько стаканов ===
	args := make([]string, 0, len(orderBook.Symbols))
	for _, s := range orderBook.Symbols {
		args = append(args, fmt.Sprintf("orderbook.50.%s", s.Name)) // 50 уровней
	}

	subscribeMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		fmt.Println("[Bybit WS] Ошибка подписки:", err)
		return
	}

	// Основной цикл чтения
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("[Bybit WS] Ошибка чтения: %v\n", err)
			break
		}

		var event BybitDepthEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		applyDepthUpdate(orderBook, event)
	}
}

func applyDepthUpdate(ob *domain.OrderBook, event BybitDepthEvent) {
	for i := range ob.Symbols {
		if ob.Symbols[i].Name == event.Data.Symbol {
			s := &ob.Symbols[i]

			// Bybit присылает snapshot при первой подписке и delta дальше
			if event.Type == "snapshot" {
				// Полная очистка и загрузка
				s.Data.Bids = make(map[string]float64)
				s.Data.Asks = make(map[string]float64)
			}

			// Применяем bids
			for _, level := range event.Data.Bids {
				price := level[0]
				qtyStr := level[1]
				if qtyStr == "0" || qtyStr == "0.00000000" {
					delete(s.Data.Bids, price)
				} else {
					s.Data.Bids[price], _ = strconv.ParseFloat(qtyStr, 64)
				}
			}

			// Применяем asks
			for _, level := range event.Data.Asks {
				price := level[0]
				qtyStr := level[1]
				if qtyStr == "0" || qtyStr == "0.00000000" {
					delete(s.Data.Asks, price)
				} else {
					s.Data.Asks[price], _ = strconv.ParseFloat(qtyStr, 64)
				}
			}

			s.Data.LastUpdateId = event.Data.UpdateID
			s.Data.LastUpdateTime = time.Now().UnixMilli()

			// fmt.Printf("[%-7s] %s | %s | Bids:%2d | Asks:%2d\n",
			// 	ob.Exchange, s.Name, event.Type, len(s.Data.Bids), len(s.Data.Asks))

			return
		}
	}
}
