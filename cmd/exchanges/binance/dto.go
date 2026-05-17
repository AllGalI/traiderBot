package binance

type AccountData struct {
	UID             int32
	ComissionsRates map[string]float64
	Balances        []Balance
	CanTrade        bool
	CanWithdraw     bool
}

type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}
