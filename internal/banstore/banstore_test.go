package banstore

import (
	"testing"
	"time"
)

func TestBanStore_FirstBan(t *testing.T) {
	store := NewBanStore(time.Second, time.Minute)

	now := time.Unix(1000, 0)
	key := RouteKey("BTC->USDT->ETH->BTC")

	result := store.Ban(key, now)

	if result.WasBanned {
		t.Fatal("first ban must not report WasBanned=true")
	}

	if result.Duration != time.Second {
		t.Fatalf("expected duration 1s, got %v", result.Duration)
	}

	if result.Level != 0 {
		t.Fatalf("expected level 0, got %d", result.Level)
	}

	if !result.ExpiresAt.Equal(now.Add(time.Second)) {
		t.Fatalf("unexpected expiration: %v", result.ExpiresAt)
	}

	if !store.IsBanned(key, now.Add(500*time.Millisecond)) {
		t.Fatal("route must be banned")
	}
}

func TestBanStore_Expiration(t *testing.T) {
	store := NewBanStore(time.Second, time.Minute)

	now := time.Unix(1000, 0)
	key := RouteKey("route-A")

	store.Ban(key, now)

	if !store.IsBanned(key, now.Add(999*time.Millisecond)) {
		t.Fatal("route should still be banned")
	}

	if store.IsBanned(key, now.Add(time.Second)) {
		t.Fatal("route should no longer be banned")
	}

	if store.Len() != 0 {
		t.Fatalf("expired ban should be removed, len=%d", store.Len())
	}
}

func TestBanStore_ExponentialBackoff(t *testing.T) {
	store := NewBanStore(time.Second, time.Minute)

	now := time.Unix(1000, 0)
	key := RouteKey("route-A")

	tests := []struct {
		expectedDuration time.Duration
		expectedLevel    uint8
	}{
		{time.Second, 0},
		{2 * time.Second, 1},
		{4 * time.Second, 2},
		{8 * time.Second, 3},
		{16 * time.Second, 4},
	}

	for i, tt := range tests {
		result := store.Ban(key, now)

		if result.Duration != tt.expectedDuration {
			t.Fatalf(
				"ban #%d: expected duration %v, got %v",
				i+1,
				tt.expectedDuration,
				result.Duration,
			)
		}

		if result.Level != tt.expectedLevel {
			t.Fatalf(
				"ban #%d: expected level %d, got %d",
				i+1,
				tt.expectedLevel,
				result.Level,
			)
		}

		now = now.Add(100 * time.Millisecond)
	}
}

func TestBanStore_MaxTTL(t *testing.T) {
	store := NewBanStore(time.Second, 4*time.Second)

	now := time.Unix(1000, 0)
	key := RouteKey("route-A")

	expected := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		4 * time.Second,
		4 * time.Second,
	}

	for i, expectedDuration := range expected {
		result := store.Ban(key, now)

		if result.Duration != expectedDuration {
			t.Fatalf(
				"ban #%d: expected %v, got %v",
				i+1,
				expectedDuration,
				result.Duration,
			)
		}

		now = now.Add(time.Second)
	}
}

func TestBanStore_Unban(t *testing.T) {
	store := NewBanStore(time.Second, time.Minute)

	now := time.Unix(1000, 0)
	key := RouteKey("route-A")

	store.Ban(key, now)

	if !store.IsBanned(key, now) {
		t.Fatal("route should be banned")
	}

	store.Unban(key)

	if store.IsBanned(key, now) {
		t.Fatal("route should be unbanned")
	}

	if store.Len() != 0 {
		t.Fatalf("expected empty store, len=%d", store.Len())
	}
}

func TestBanStore_UnbanResetsBackoff(t *testing.T) {
	store := NewBanStore(time.Second, time.Minute)

	now := time.Unix(1000, 0)
	key := RouteKey("route-A")

	store.Ban(key, now)
	store.Ban(key, now)
	store.Ban(key, now)

	result := store.Ban(key, now)

	if result.Duration != 8*time.Second {
		t.Fatalf("expected 8s, got %v", result.Duration)
	}

	store.Unban(key)

	result = store.Ban(key, now)

	if result.Duration != time.Second {
		t.Fatalf(
			"new ban after unban should start from initial TTL, got %v",
			result.Duration,
		)
	}

	if result.Level != 0 {
		t.Fatalf("new ban should have level 0, got %d", result.Level)
	}
}

func TestBanStore_RenewalIsFromNow(t *testing.T) {
	store := NewBanStore(time.Second, time.Minute)

	key := RouteKey("route-A")
	t0 := time.Unix(1000, 0)

	first := store.Ban(key, t0)

	if !first.ExpiresAt.Equal(t0.Add(time.Second)) {
		t.Fatalf("unexpected first expiration: %v", first.ExpiresAt)
	}

	// Повторный hit через 500ms.
	t1 := t0.Add(500 * time.Millisecond)

	second := store.Ban(key, t1)

	// TTL стал 2s и считается от t1.
	expected := t1.Add(2 * time.Second)

	if !second.ExpiresAt.Equal(expected) {
		t.Fatalf(
			"expected expiration %v, got %v",
			expected,
			second.ExpiresAt,
		)
	}
}