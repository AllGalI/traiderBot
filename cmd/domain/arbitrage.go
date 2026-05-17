package domain

// type ArbitrageOpportunity struct {
// 	Symbol        string
// 	BuyExchange   string
// 	SellExchange  string
// 	BuyPrice      float64
// 	SellPrice     float64
// 	ProfitPercent float64
// 	ProfitUSDT    float64
// 	Volume        float64 // сколько базовой монеты торгуем
// 	Reason        string  // почему выгодно / ограничение
// }

// type ArbitrageResult struct {
// 	Opportunities []ArbitrageOpportunity
// 	Timestamp     int64
// }

type ArbitrageResult struct {
	Volume      float64
	MinAskPrice float64
	MaxBidPrice float64
}
