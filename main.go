package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"traiderBot/cmd/domain"
	"traiderBot/cmd/exchanges/binance"
	"traiderBot/cmd/exchanges/bybit"
	"traiderBot/cmd/filesHandler"
)

type Config struct {
	UpdateIntervalSec int      `json:"update_interval_sec"`
	MinVolumeUSDT     float64  `json:"min_volume_usdt"`
	Exchanges         []string `json:"exchanges"`
	TelegramToken     string   `json:"telegram_token"`
	TelegramChatID    string   `json:"telegram_chat_id"`
}

type PriceInfo struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume24h float64   `json:"volume24h"`
	Timestamp time.Time `json:"timestamp"`
}

type ArbitrageOpp struct {
	Symbol    string
	BuyEx     string
	SellEx    string
	BuyPrice  float64
	SellPrice float64
	SpreadPct float64
}

var (
	prices   = make(map[string]map[string]PriceInfo) // exchange -> symbol -> PriceInfo
	pricesMu sync.RWMutex
	config   Config
)

func app() {
	loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("🚀 Запуск мониторинга крипто-арбитража...")

	// Запускаем мониторинг для топ-4 бирж
	for _, ex := range config.Exchanges {
		go monitorExchange(ctx, ex)
		time.Sleep(500 * time.Millisecond) // небольшая задержка при старте
	}

	// Сканер арбитража
	go arbitrageScanner()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	cancel()
	fmt.Println("\n👋 Мониторинг остановлен")
}

func loadConfig() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Println("config.json не найден, создаём дефолтный")
		config = Config{
			UpdateIntervalSec: 5,
			MinVolumeUSDT:     100000,
			Exchanges:         []string{"binance", "bybit", "okx", "kucoin"},
		}
		saveConfig()
		return
	}

	json.Unmarshal(data, &config)
	fmt.Printf("✅ Загружено бирж: %d\n", len(config.Exchanges))
}

func saveConfig() {
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("config.json", data, 0644)
}

// ====================== Мониторинг одной биржи ======================
func monitorExchange(ctx context.Context, exchange string) {
	exchangeUpper := strings.ToUpper(exchange)
	log.Printf("📡 Мониторинг %s запущен (интервал: %d сек)", exchangeUpper, config.UpdateIntervalSec)

	ticker := time.NewTicker(time.Duration(config.UpdateIntervalSec) * time.Second)
	defer ticker.Stop()

	updateCount := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Мониторинг %s остановлен", exchangeUpper)
			return

		case <-ticker.C:
			start := time.Now()
			updateExchangePrices(exchange)
			elapsed := time.Since(start)

			updateCount++

			// Логируем подробно каждые 10 обновлений или при долгом запросе
			if updateCount%10 == 0 || elapsed > 800*time.Millisecond {
				log.Printf("[%s] ✅ Обновлено | Время: %v | Обновлений: %d",
					exchangeUpper, elapsed.Round(time.Millisecond), updateCount)
				updateCount = 0
			}
		}
	}
}

// ====================== Обновление цен (будет расширяться) ======================
func updateExchangePrices(exchange string) {
	var tickers map[string]PriceInfo

	switch exchange {
	case "binance":
		tickers = fetchBinanceUSDT()
	case "bybit":
		tickers = fetchBybitUSDT()
	case "okx":
		tickers = fetchOKXUSDT()
	case "kucoin":
		tickers = fetchKuCoinUSDT()
	default:
		return
	}

	pricesMu.Lock()
	if prices[exchange] == nil {
		prices[exchange] = make(map[string]PriceInfo)
	}
	for k, v := range tickers {
		prices[exchange][k] = v
	}
	pricesMu.Unlock()
}

// ====================== Арбитражный сканер ======================
func arbitrageScanner() {
	for {
		time.Sleep(8 * time.Second)

		pricesMu.RLock()
		opportunities := findArbitrage()
		pricesMu.RUnlock()

		if len(opportunities) > 0 {
			fmt.Printf("\n🔥 Найдено арбитражных возможностей: %d\n", len(opportunities))
			for i, opp := range opportunities {
				if i >= 8 {
					break // показываем топ-8
				}
				fmt.Printf("%s | %s %.6f → %s %.6f | +%.2f%%\n",
					opp.Symbol, opp.BuyEx, opp.BuyPrice, opp.SellEx, opp.SellPrice, opp.SpreadPct)
			}
		}
	}
}

func findArbitrage() []ArbitrageOpp {
	var opps []ArbitrageOpp

	// Берём все символы из Binance как базу (самая большая ликвидность)
	baseSymbols := prices["binance"]

	for symbol, basePrice := range baseSymbols {
		if basePrice.Price == 0 {
			continue
		}

		for ex, exPrices := range prices {
			if ex == "binance" {
				continue
			}

			if p, exists := exPrices[symbol]; exists && p.Price > 0 {
				spreadPct := (p.Price - basePrice.Price) / basePrice.Price * 100

				if spreadPct > 0.3 { // покупаем на binance, продаём дороже
					opps = append(opps, ArbitrageOpp{
						Symbol:    symbol,
						BuyEx:     "binance",
						SellEx:    ex,
						BuyPrice:  basePrice.Price,
						SellPrice: p.Price,
						SpreadPct: spreadPct,
					})
				} else if spreadPct < -0.3 { // наоборот
					opps = append(opps, ArbitrageOpp{
						Symbol:    symbol,
						BuyEx:     ex,
						SellEx:    "binance",
						BuyPrice:  p.Price,
						SellPrice: basePrice.Price,
						SpreadPct: -spreadPct,
					})
				}
			}
		}
	}

	return opps
}

// Binance
func fetchBinanceUSDT() map[string]PriceInfo {
	tickers := make(map[string]PriceInfo)

	resp, err := http.Get("https://api.binance.com/api/v3/ticker/24hr")
	if err != nil {
		log.Printf("Binance error: %v", err)
		return tickers
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data []struct {
		Symbol    string `json:"symbol"`
		LastPrice string `json:"lastPrice"`
		Volume    string `json:"volume"`
	}

	json.Unmarshal(body, &data)

	for _, t := range data {
		if !strings.HasSuffix(t.Symbol, "USDT") {
			continue
		}
		price, _ := strconv.ParseFloat(t.LastPrice, 64)
		volume, _ := strconv.ParseFloat(t.Volume, 64)

		if volume < config.MinVolumeUSDT {
			continue
		}

		tickers[t.Symbol] = PriceInfo{
			Exchange:  "binance",
			Symbol:    t.Symbol,
			Price:     price,
			Volume24h: volume,
			Timestamp: time.Now(),
		}
	}
	return tickers
}

// Bybit
func fetchBybitUSDT() map[string]PriceInfo {
	tickers := make(map[string]PriceInfo)

	resp, err := http.Get("https://api.bybit.com/v5/market/tickers?category=spot")
	if err != nil {
		return tickers
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Result struct {
			List []struct {
				Symbol    string `json:"symbol"`
				LastPrice string `json:"lastPrice"`
				Volume    string `json:"volume24h"`
			} `json:"list"`
		} `json:"result"`
	}

	json.Unmarshal(body, &result)

	for _, t := range result.Result.List {
		if !strings.HasSuffix(t.Symbol, "USDT") {
			continue
		}
		price, _ := strconv.ParseFloat(t.LastPrice, 64)
		volume, _ := strconv.ParseFloat(t.Volume, 64)

		if volume < config.MinVolumeUSDT {
			continue
		}

		tickers[t.Symbol] = PriceInfo{
			Exchange:  "bybit",
			Symbol:    t.Symbol,
			Price:     price,
			Volume24h: volume,
			Timestamp: time.Now(),
		}
	}
	return tickers
}

// OKX
func fetchOKXUSDT() map[string]PriceInfo {
	tickers := make(map[string]PriceInfo)

	resp, err := http.Get("https://www.okx.com/api/v5/market/tickers?instType=SPOT")
	if err != nil {
		return tickers
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			InstID string `json:"instId"`
			Last   string `json:"last"`
			VolCcy string `json:"volCcy24h"`
		} `json:"data"`
	}

	json.Unmarshal(body, &result)

	for _, t := range result.Data {
		if !strings.HasSuffix(t.InstID, "-USDT") {
			continue
		}
		symbol := strings.Replace(t.InstID, "-", "", -1)
		price, _ := strconv.ParseFloat(t.Last, 64)
		volume, _ := strconv.ParseFloat(t.VolCcy, 64)

		if volume < config.MinVolumeUSDT {
			continue
		}

		tickers[symbol] = PriceInfo{
			Exchange:  "okx",
			Symbol:    symbol,
			Price:     price,
			Volume24h: volume,
			Timestamp: time.Now(),
		}
	}
	return tickers
}

// KuCoin
func fetchKuCoinUSDT() map[string]PriceInfo {
	tickers := make(map[string]PriceInfo)

	resp, err := http.Get("https://api.kucoin.com/api/v1/market/allTickers")
	if err != nil {
		return tickers
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Ticker []struct {
				Symbol string `json:"symbol"`
				Last   string `json:"last"`
				Vol    string `json:"vol"`
			} `json:"ticker"`
		} `json:"data"`
	}

	json.Unmarshal(body, &result)

	for _, t := range result.Data.Ticker {
		if !strings.HasSuffix(t.Symbol, "-USDT") {
			continue
		}
		symbol := strings.Replace(t.Symbol, "-", "", -1)
		price, _ := strconv.ParseFloat(t.Last, 64)
		volume, _ := strconv.ParseFloat(t.Vol, 64)

		if volume < config.MinVolumeUSDT {
			continue
		}

		tickers[symbol] = PriceInfo{
			Exchange:  "kucoin",
			Symbol:    symbol,
			Price:     price,
			Volume24h: volume,
			Timestamp: time.Now(),
		}
	}
	return tickers
}

// func updatePairsData() error {
// 	err := binance.FetchSymbols()
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Binance fetched")

// 	err = bybit.FetchSymbols()
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Bybit fetched")

// 	return nil
// }

func fetchExchVolumes() ([]domain.OrderBook, error) {
	orderbooks := []domain.OrderBook{}
	limit := 100

	symbols, err := filesHandler.ReadSymbols()
	if err != nil {
		return []domain.OrderBook{}, err
	}

	binanceVolumes, err := binance.FetchVolumes(symbols, "binance", limit)
	if err != nil {
		return []domain.OrderBook{}, err
	}
	orderbooks = append(orderbooks, binanceVolumes)

	bybitVolumes, err := bybit.FetchVolumes(symbols, "bybit", limit)
	if err != nil {
		return []domain.OrderBook{}, err
	}
	orderbooks = append(orderbooks, bybitVolumes)

	return orderbooks, nil
}

func connectOrderBooks(exchVolumes *[]domain.OrderBook) error {
	// Обновлять данную хрень
	for _, orderBook := range *exchVolumes {
		switch orderBook.Exchange {
		case "binance":
			binance.ConnectOrderBooks(&orderBook)
		case "bybit":
			bybit.ConnectOrderBooks(&orderBook)
		}
	}

	return nil
}

func fetchAccountsInfo() (map[string]domain.AccountData, error) {
	result, err := binance.FetchAccountInfo()
	if err != nil {
		return map[string]domain.AccountData{}, nil
	}
	return map[string]domain.AccountData{"binance": result}, nil
}

// OPUSDT | binance 0.144300 → okx 0.144400 | +1.00%func main() {
func main() {

	exchVolumes, err := fetchExchVolumes()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("orderbooks are loaded")
	connectOrderBooks(&exchVolumes)

	fmt.Println(exchVolumes)

	exchAccounts, err := fetchAccountsInfo()
	if err != nil {
		log.Fatal(err)
	}

	hh := exchVolumes[0].Symbols[0]
	bb := exchVolumes[1].Symbols[0]
}
