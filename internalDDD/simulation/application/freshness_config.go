package application

import (
	"time"
)

type FreshnessConfig struct {
	GeneratedAt  time.Time                  `json:"generated_at"`
	DefaultTTLMS int64                      `json:"default_ttl_ms"`
	Symbols      map[string]FreshnessSymbol `json:"symbols"`
}

type FreshnessSymbol struct {
	TTLMS        int64  `json:"ttl_ms"`
	Percentile   string `json:"percentile"`
	PercentileMS int64  `json:"percentile_ms"`
	SampleCount  int64  `json:"sample_count"`
}

func (c *FreshnessConfig) TTLs() map[string]time.Duration {
	ttls := make(map[string]time.Duration, len(c.Symbols))

	for symbol, item := range c.Symbols {
		ttls[symbol] = time.Duration(item.TTLMS) * time.Millisecond
	}

	return ttls
}
