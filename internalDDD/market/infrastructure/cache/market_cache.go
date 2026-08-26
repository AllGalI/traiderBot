package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"traiderBot/internalDDD/market/application"
)

type MarketCache struct {
	path string
}

func NewMarketCache(path string) *MarketCache {
	return &MarketCache{
		path: path,
	}
}

func (c *MarketCache) Get(
	ctx context.Context,
) (application.MarketStructure, bool, error) {
	select {
	case <-ctx.Done():
		return application.MarketStructure{}, false, ctx.Err()
	default:
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return application.MarketStructure{}, false, nil
		}

		return application.MarketStructure{}, false, fmt.Errorf(
			"read cache file: %w",
			err,
		)
	}

	var structure application.MarketStructure

	if err := json.Unmarshal(data, &structure); err != nil {
		return application.MarketStructure{}, false, fmt.Errorf(
			"decode market structure cache: %w",
			err,
		)
	}

	return structure, true, nil
}

func (c *MarketCache) Set(
	ctx context.Context,
	structure application.MarketStructure,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := json.MarshalIndent(structure, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"encode market structure cache: %w",
			err,
		)
	}

	if err := os.WriteFile(
		c.path,
		data,
		0644,
	); err != nil {
		return fmt.Errorf(
			"write cache file: %w",
			err,
		)
	}

	return nil
}
