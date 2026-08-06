package observe

import "sync/atomic"

type SIMCounters struct {
	InvalidRouteLength atomic.Int64
	OrderbookNotFound  atomic.Int64
	FeeNotFound        atomic.Int64
	FiltersNotFound    atomic.Int64
	NotEnoughLiquidity atomic.Int64
	LotSize            atomic.Int64
	MinNotional        atomic.Int64
	StartAsset         atomic.Int64
	ProfitTooLow       atomic.Int64
	OpportunityFound   atomic.Int64
}

func NewSIMCounters() *SIMCounters {
	return &SIMCounters{}
}
