package application

import (
	marketDomain "traiderBot/internalDDD/market/domain"
)

type MarketStructureReader interface {
	Get() (*marketDomain.MarketStructure, bool)
}
