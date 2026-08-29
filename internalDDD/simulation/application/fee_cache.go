package application

import (
	"context"
	"sync"
)

type FeeCache struct {
	mu   sync.RWMutex
	fees map[string]FeeConfig
}

func NewFeeCache() *FeeCache {
	return &FeeCache{
		fees: make(map[string]FeeConfig),
	}
}

func (c *FeeCache) Get(
	ctx context.Context,
	symbol string,
) (FeeConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fee, ok := c.fees[symbol]

	return fee, ok
}

func (c *FeeCache) Set(
	symbol string,
	fee FeeConfig,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.fees[symbol] = fee
}
