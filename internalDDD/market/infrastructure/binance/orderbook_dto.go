package binance

type orderBookDTO struct {
    LastUpdateID int64            `json:"lastUpdateId"`
    Bids         [][]string       `json:"bids"`
    Asks         [][]string       `json:"asks"`
}