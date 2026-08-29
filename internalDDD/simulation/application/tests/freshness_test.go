package tests

import (
	"testing"
	"time"

	simulationApplication "traiderBot/internalDDD/simulation/application"
)

func TestFixedTTLFreshnessPolicyFresh(t *testing.T) {
	now := time.Date(
		2026,
		time.January,
		1,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	policy := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Second,
		},
	)

	updatedAt := now.Add(-500 * time.Millisecond)

	if !policy.IsFresh("BTCUSDT", updatedAt, now) {
		t.Fatal("expected order book to be fresh")
	}
}

func TestFixedTTLFreshnessPolicyExpired(t *testing.T) {
	now := time.Date(
		2026,
		time.January,
		1,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	policy := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Second,
		},
	)

	updatedAt := now.Add(-1500 * time.Millisecond)

	if policy.IsFresh("BTCUSDT", updatedAt, now) {
		t.Fatal("expected order book to be stale")
	}
}

func TestFixedTTLFreshnessPolicyAtTTLBoundary(t *testing.T) {
	now := time.Date(
		2026,
		time.January,
		1,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	policy := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Second,
		},
	)

	updatedAt := now.Add(-time.Second)

	if !policy.IsFresh("BTCUSDT", updatedAt, now) {
		t.Fatal("expected order book to be fresh at TTL boundary")
	}
}

func TestFixedTTLFreshnessPolicyUnknownSymbol(t *testing.T) {
	now := time.Now()

	policy := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Second,
		},
	)

	if policy.IsFresh(
		"ETHUSDT",
		now.Add(-100*time.Millisecond),
		now,
	) {
		t.Fatal("expected unknown symbol to be stale")
	}
}

func TestFixedTTLFreshnessPolicyZeroTimestamp(t *testing.T) {
	now := time.Now()

	policy := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Second,
		},
	)

	if policy.IsFresh(
		"BTCUSDT",
		time.Time{},
		now,
	) {
		t.Fatal("expected zero timestamp to be stale")
	}
}

func TestFixedTTLFreshnessPolicyFutureTimestamp(t *testing.T) {
	now := time.Now()

	policy := simulationApplication.NewFixedTTLFreshnessPolicy(
		map[string]time.Duration{
			"BTCUSDT": time.Second,
		},
	)

	if policy.IsFresh(
		"BTCUSDT",
		now.Add(time.Second),
		now,
	) {
		t.Fatal("expected future timestamp to be stale")
	}
}
