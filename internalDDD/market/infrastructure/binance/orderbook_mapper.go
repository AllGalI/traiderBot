package binance

import (
	"fmt"
	"time"
	"traiderBot/internalDDD/market/domain"

	"github.com/shopspring/decimal"
)

func mapOrderBook(dto orderBookDTO, symbol string) (domain.OrderBook, error) {
	bids, err := mapPriceLevels(dto.Bids)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("map bids: %w", err)
	}

	asks, err := mapPriceLevels(dto.Asks)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("map asks: %w", err)
	}

	return domain.OrderBook{
		Symbol:         symbol,
		LastUpdateID:   dto.LastUpdateID,
		LastUpdateTime: time.Now(),
		Bids:           bids,
		Asks:           asks,
	}, nil
}

func mapPriceLevels(levels [][]string) ([]domain.PriceLevel, error) {
	priceLevels := make([]domain.PriceLevel, 0, len(levels))

	for i, level := range levels {
		if len(level) != 2 {
			return nil, fmt.Errorf(
				"invalid price level at index %d: expected 2 values, got %d",
				i,
				len(level),
			)
		}

		price, err := decimal.NewFromString(level[0])
		if err != nil {
			return nil, fmt.Errorf(
				"parse price at index %d: %w",
				i,
				err,
			)
		}

		qty, err := decimal.NewFromString(level[1])
		if err != nil {
			return nil, fmt.Errorf(
				"parse quantity at index %d: %w",
				i,
				err,
			)
		}

		priceLevels = append(priceLevels, domain.PriceLevel{
			Price: price,
			Qty:   qty,
		})
	}

	return priceLevels, nil
}
