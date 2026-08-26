package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type PriceLevel struct {
	Price decimal.Decimal
	Qty   decimal.Decimal
}

type OrderBook struct {
	Symbol         string
	LastUpdateID   int64
	LastUpdateTime time.Time
	Asks           []PriceLevel
	Bids           []PriceLevel
}

func (ob *OrderBook) GetBestBid() (PriceLevel, bool) {
	if len(ob.Bids) == 0 {
		return PriceLevel{}, false
	}
	return ob.Bids[0], true
}

func (ob *OrderBook) GetBestAsk() (PriceLevel, bool) {
	if len(ob.Asks) == 0 {
		return PriceLevel{}, false
	}
	return ob.Asks[0], true
}
