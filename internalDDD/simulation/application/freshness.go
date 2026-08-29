package application

import "time"

type FreshnessPolicy interface {
	IsFresh(symbol string, updatedAt time.Time, now time.Time) bool
}

type FixedTTLFreshnessPolicy struct {
	ttls map[string]time.Duration
}

func NewFixedTTLFreshnessPolicy(
	ttls map[string]time.Duration,
) *FixedTTLFreshnessPolicy {
	copied := make(map[string]time.Duration, len(ttls))

	for symbol, ttl := range ttls {
		copied[symbol] = ttl
	}

	return &FixedTTLFreshnessPolicy{
		ttls: copied,
	}
}

func (p *FixedTTLFreshnessPolicy) IsFresh(
	symbol string,
	updatedAt time.Time,
	now time.Time,
) bool {
	ttl, ok := p.ttls[symbol]
	if !ok {
		return false
	}

	if updatedAt.IsZero() {
		return false
	}

	if now.Before(updatedAt) {
		return false
	}

	return now.Sub(updatedAt) <= ttl
}
