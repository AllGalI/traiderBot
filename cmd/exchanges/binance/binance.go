package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"traiderBot/cmd/domain"
	"traiderBot/cmd/filesHandler"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var (
	pairsFile     = "pairs.json"
	baseUrl       = "https://api.binance.com/api/v3"
	baseWSUrl     = "wss://ws-api.binance.com:443/ws-api/v3"
	baseTestWSUrl = "wss://ws-api.testnet.binance.vision/ws-api/v3"
)

func getServerTime() (int64, error) {
	resp, err := http.Get("https://api.binance.com/api/v3/time")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		ServerTime int64 `json:"serverTime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	return result.ServerTime, nil
}

func FetchAccountInfo() (domain.AccountData, error) {
	err := godotenv.Load()
	if err != nil {
		return domain.AccountData{}, err
	}

	apiKey := os.Getenv("API_KEY")
	secretKey := os.Getenv("SECRET_KEY")

	timestamp, err := getServerTime()
	if err != nil {
		return domain.AccountData{}, err
	}

	query := fmt.Sprintf("omitZeroBalances=true&timestamp=%d", timestamp)

	// подпись
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(query))
	signature := hex.EncodeToString(mac.Sum(nil))

	url := fmt.Sprintf("%s/account?%s&signature=%s", baseUrl, query, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return domain.AccountData{}, err
	}

	req.Header.Set("X-MBX-APIKEY", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return domain.AccountData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return domain.AccountData{}, fmt.Errorf("binance error: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	if err != nil {
		return domain.AccountData{}, err
	}

	var result domain.AccountData
	if err := json.Unmarshal(body, &result); err != nil {
		return domain.AccountData{}, err
	}

	return result, nil
}

func FetchSymbols() error {
	resp, err := http.Get(baseUrl + "/ticker/24hr?symbolStatus=TRADING")
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var allTicks []domain.TickerData
	json.Unmarshal(
		body,
		&allTicks,
	)

	var ticks [15]string

	n := min(len(allTicks), 15)

	for i := range n {
		ticks[i] = allTicks[i].Symbol
	}

	filesHandler.SaveSymbols(ticks)

	return nil
}

type BinanceCombinedDepthEvent struct {
	Stream string `json:"stream"`
	Data   struct {
		Symbol       string     `json:"s"`
		LastUpdateID int64      `json:"u"`
		Bids         [][]string `json:"b"`
		Asks         [][]string `json:"a"`
	} `json:"data"`
}

// ==================== FETCH INITIAL SNAPSHOT ====================

func fetchVolume(symbol string, limit int) (domain.Data, error) {
	resp, err := http.Get(baseUrl + fmt.Sprintf("/depth?limit=%d&symbol=%s", limit, symbol))
	if err != nil {
		return domain.Data{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Data{}, fmt.Errorf("status: %s", resp.Status)
	}

	var result struct {
		LastUpdateId int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.Data{}, err
	}

	return convertToDomainData(result), nil
}

func convertToDomainData(src struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}) domain.Data {

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

func ConnectOrderBooks(orderBook *domain.OrderBook) {
	if len(orderBook.Symbols) == 0 {
		return
	}

	streams := make([]string, 0, len(orderBook.Symbols))
	for _, s := range orderBook.Symbols {
		streams = append(streams, strings.ToLower(s.Name)+"@depth@100ms")
	}

	wsURL := "wss://stream.binance.com:9443/stream?streams=" + strings.Join(streams, "/")

	go connectCombinedStream(wsURL, orderBook)
}

func connectCombinedStream(wsURL string, orderBook *domain.OrderBook) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("[Binance WS] Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("[Binance] Combined WS connected (%d symbols)\n", len(orderBook.Symbols))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("[Binance WS] Read error: %v\n", err)
			break
		}

		var event BinanceCombinedDepthEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		applyDepthUpdate(orderBook, event)
	}
}

func applyDepthUpdate(ob *domain.OrderBook, event BinanceCombinedDepthEvent) {
	for i := range ob.Symbols {
		if strings.EqualFold(ob.Symbols[i].Name, event.Data.Symbol) {
			s := &ob.Symbols[i]

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

			s.Data.LastUpdateId = event.Data.LastUpdateID
			s.Data.LastUpdateTime = time.Now().UnixMilli()

			// Лог
			// fmt.Printf("[%-7s] %s | Δ=%3dms | Bids:%2d | Asks:%2d\n",
			// 	ob.Exchange,
			// 	s.Name,
			// 	0, // можно посчитать разницу по времени
			// 	len(s.Data.Bids),
			// 	len(s.Data.Asks),
			// )
			return
		}
	}
}

// ==================== TestFuncs ====================

// TestAccountData возвращает искусственные тестовые данные
func TestAccountData() map[string]domain.AccountData {
	return map[string]domain.AccountData{
		"binance": {
			UID: 452108188,
			CommissionRates: map[string]string{
				"maker":  "0.00100000",
				"taker":  "0.00100000",
				"buyer":  "0.00000000",
				"seller": "0.00000000",
			},
			Balances: []domain.Balance{
				{Asset: "USDT", Free: "12450.75", Locked: "0.00"},
				{Asset: "BTC", Free: "0.1842", Locked: "0.00"},
				{Asset: "ETH", Free: "2.75", Locked: "0.00"},
				{Asset: "PEPE", Free: "1250000", Locked: "0.00"},
			},
			CanTrade:    true,
			CanWithdraw: true,
		},
		"bybit": {
			UID: 987654321,
			CommissionRates: map[string]string{
				"maker": "0.0009",
				"taker": "0.0010",
			},
			Balances: []domain.Balance{
				{Asset: "USDT", Free: "8750.25", Locked: "500.00"},
				{Asset: "BTC", Free: "0.0921", Locked: "0.00"},
			},
			CanTrade:    true,
			CanWithdraw: true,
		},
	}
}
