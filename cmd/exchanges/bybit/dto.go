package bybit

type BybitOrderBookResponse struct {
	BybitOrderBookData `json:"result"`
}

type BybitOrderBookData struct {
	LastUpdateId int64      `json:"u"`
	Bids         [][]string `json:"b"` // Bids[0][0] - Price Bids[0][1] - Quantity
	Asks         [][]string `json:"a"`
}

// func (b BybitOrderBookData) ToDomain() domain.Data {
// 	return domain.Data{
// 		LastUpdateId: b.LastUpdateId,
// 		Bids:         b.Bids,
// 		Asks:         b.Asks,
// 	}
// }
