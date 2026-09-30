package rest


type orderResponse struct {

	Symbol string `json:"symbol"`

	OrderID int64 `json:"orderId"`

	ClientOrderID string `json:"clientOrderId"`


	Status string `json:"status"`


	ExecutedQty string `json:"executedQty"`

	CumulativeQuoteQty string `json:"cummulativeQuoteQty"`


	Fills []fill `json:"fills"`
}



type fill struct {

	Price string `json:"price"`

	Qty string `json:"qty"`

	Commission string `json:"commission"`

	CommissionAsset string `json:"commissionAsset"`
}