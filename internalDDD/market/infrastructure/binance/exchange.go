package binance

type exchangeInfoDTO struct {
	Symbols []symbolDTO `json:"symbols"`
}

type symbolDTO struct {
	Symbol     string      `json:"symbol"`
	BaseAsset  string      `json:"baseAsset"`
	QuoteAsset string      `json:"quoteAsset"`
	Status     string      `json:"status"`
	Filters    []filterDTO `json:"filters"`
}

type filterDTO struct {
	FilterType  string `json:"filterType"`
	MinQty      string `json:"minQty,omitempty"`
	MaxQty      string `json:"maxQty,omitempty"`
	StepSize    string `json:"stepSize,omitempty"`
	MinNotional string `json:"minNotional,omitempty"`
	MinPrice    string `json:"minPrice,omitempty"`
	MaxPrice    string `json:"maxPrice,omitempty"`
	TickSize    string `json:"tickSize,omitempty"`
}