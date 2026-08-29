package application

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func LoadFreshnessConfig(
	path string,
) (FreshnessConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FreshnessConfig{}, fmt.Errorf(
			"read freshness config: %w",
			err,
		)
	}

	var config FreshnessConfig

	if err := json.Unmarshal(data, &config); err != nil {
		return FreshnessConfig{}, fmt.Errorf(
			"decode freshness config: %w",
			err,
		)
	}

	return config, nil
}

func (c FreshnessConfig) NewPolicy() *FixedTTLFreshnessPolicy {
	ttls := make(map[string]time.Duration, len(c.Symbols))

	for symbol, config := range c.Symbols {
		ttls[symbol] = time.Duration(config.TTLMS) * time.Millisecond
	}

	return NewFixedTTLFreshnessPolicy(ttls)
}
