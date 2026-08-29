package application

import (
	marketDomain "traiderBot/internalDDD/market/domain"
)

type SymbolReader struct {
	structure MarketStructureReader
}

func NewSymbolReader(
	structure MarketStructureReader,
) *SymbolReader {
	return &SymbolReader{
		structure: structure,
	}
}

func (r *SymbolReader) Get(
	symbol string,
) (marketDomain.Symbol, bool) {
	structure, ok := r.structure.Get()
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
