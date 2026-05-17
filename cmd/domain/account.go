package domain

type AccountData struct {
	UID             int32
	CommissionRates map[string]string
	Balances        []Balance
	CanTrade        bool
	CanWithdraw     bool
}

type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}
