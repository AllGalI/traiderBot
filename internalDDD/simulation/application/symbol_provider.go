package application

import (
	marketDomain "traiderBot/internalDDD/market/domain"
)

type MarketStructureStore interface {
	Get() (*marketDomain.MarketStructure, bool)
}

type MarketStructureSymbolProvider struct {
	store MarketStructureStore
}

func NewMarketStructureSymbolProvider(
	store MarketStructureStore,
) *MarketStructureSymbolProvider {
	return &MarketStructureSymbolProvider{
		store: store,
	}
}

func (p *MarketStructureSymbolProvider) Get(
	symbol string,
) (marketDomain.Symbol, bool) {
	structure, ok := p.store.Get()
	if !ok {
		return marketDomain.Symbol{}, false
	}

	for _, item := range structure.Symbols {
		if item.Name == symbol {
			return item, true
		}
	}

	return marketDomain.Symbol{}, false
}
