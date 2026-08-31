package binance

type orderResponse struct {
	Symbol             string `json:"symbol"`
	OrderID            int64  `json:"orderId"`
	ClientOrderID      string `json:"clientOrderId"`
	TransactTime       int64  `json:"transactTime"`
	UpdateTime         int64  `json:"updateTime"`
	Status             string `json:"status"`
	Side               string `json:"side"`
	ExecutedQty        string `json:"executedQty"`
	CumulativeQuoteQty string `json:"cummulativeQuoteQty"`

	Fills []orderFill `json:"fills"`
}

type orderFill struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}
