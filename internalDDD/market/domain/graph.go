package domain

import "fmt"

type TradeDirection int

const (
	Buy TradeDirection = iota
	Sell
)

type GraphEdge struct {
	From      string
	To        string
	Symbol    string
	Direction TradeDirection
}

type Graph map[string][]GraphEdge

func NewGraph(symbols []Symbol) (Graph, error) {
	graph := make(Graph)

	for _, symbol := range symbols {
		if symbol.BaseAsset == "" {
			return nil, fmt.Errorf(
				"symbol %q has empty base asset",
				symbol.Name,
			)
		}

		if symbol.QuoteAsset == "" {
			return nil, fmt.Errorf(
				"symbol %q has empty quote asset",
				symbol.Name,
			)
		}

		// Покупаем BaseAsset за QuoteAsset:
		//
		// USDT -> BTC
		//
		graph[symbol.QuoteAsset] = append(
			graph[symbol.QuoteAsset],
			GraphEdge{
				From:      symbol.QuoteAsset,
				To:        symbol.BaseAsset,
				Symbol:    symbol.Name,
				Direction: Buy,
			},
		)

		// Продаём BaseAsset за QuoteAsset:
		//
		// BTC -> USDT
		//
		graph[symbol.BaseAsset] = append(
			graph[symbol.BaseAsset],
			GraphEdge{
				From:      symbol.BaseAsset,
				To:        symbol.QuoteAsset,
				Symbol:    symbol.Name,
				Direction: Sell,
			},
		)
	}

	return graph, nil
}

func (e *GraphEdge) IsBuy() bool {
	return e.Direction == Buy
}
