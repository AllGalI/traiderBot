package domain

type TickerData struct {
	Symbol    string `json:"symbol"`
	CloseTime int64  `json:"closeTime"`
}

type OrderBook struct {
	Exchange string
	Symbols  []Symbol
}

type Symbol struct {
	Name string
	Data
}

type Data struct {
	LastUpdateId   int64              `json:"lastUpdateId"`
	Bids           map[string]float64 `json:"-"` // price -> quantity
	Asks           map[string]float64 `json:"-"` // price -> quantity
	LastUpdateTime int64              `json:"-"` // Unix Milli
}
