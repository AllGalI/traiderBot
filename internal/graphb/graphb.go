package graphb

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// Edge — направленное ребро графа
type Edge struct {
	From   string // из какого актива
	To     string // в какой актив
	Symbol string // торговая пара (BTCUSDT)
	IsBuy  bool   // true  = покупаем To за From
	// false = продаём From за To
}

// Graph — adjacency list
type Graph map[string][]Edge

// Известные котировочные активы (отсортированы по длине убыванию — важно!)
var quoteAssets = []string{
	// Старые
	"USDT", "USDC", "BUSD", "TUSD", "FDUSD",
	"BTC", "ETH", "BNB", "EUR", "TRY", "BRL", "GBP",

	// Новые из переданных пар
	"USD", "USD1", "USDS", "U", "RLUSD",
	"XRP", "SOL", "EURI",
	"JPY", "MXN", "ARS", "COP", "IDR", "ZAR", "UAH", "KZT", "AED",
}

type UnknownAsset struct {
	symbol string
}

func (e UnknownAsset) Error() string {
	return fmt.Sprintf("unknown quote asset in symbol: %s", e.symbol)
}
// parseSymbol разбирает символ Binance на base и quote
func parseSymbol(symbol string) (base, quote string, err error) {
	for _, q := range quoteAssets {
		if strings.HasSuffix(symbol, q) {
			base = symbol[:len(symbol)-len(q)]
			if base == "" {
				return "", "", fmt.Errorf("invalid symbol: %s", symbol)
			}
			return base, q, nil
		}
	}
	return "", "", UnknownAsset{symbol: symbol}
}

// BuildEdges создаёт все направленные рёбра из списка торговых пар
func buildEdges(pairs []string) ([]Edge, error) {
	var edges []Edge

	for _, symbol := range pairs {
		base, quote, err := parseSymbol(symbol)
		if errors.Is(err, UnknownAsset{symbol}){
			log.Printf("%v", err.Error())
			continue
		}

		// Ребро 1: покупаем base за quote  (quote → base)
		edges = append(edges, Edge{
			From:   quote,
			To:     base,
			Symbol: symbol,
			IsBuy:  true,
		})

		// Ребро 2: продаём base за quote   (base → quote)
		edges = append(edges, Edge{
			From:   base,
			To:     quote,
			Symbol: symbol,
			IsBuy:  false,
		})
	}

	return edges, nil
}

// BuildGraph строит граф из списка рёбер
func buildGraph(edges []Edge) Graph {
	graph := make(Graph)

	for _, e := range edges {
		graph[e.From] = append(graph[e.From], e)
	}

	return graph
}

func NewGraph(pairs []string) (Graph, error){
	edges, err := buildEdges(pairs)
	if err != nil {
		return nil, err
	}

	graph := buildGraph(edges)
	return graph, nil
}