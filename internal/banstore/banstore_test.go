package banstore

import (
	"testing"
	"time"
)

func TestBanStore_FirstBanUsesInitialTTL(t *testing.T) {
	initialTTL := time.Second
	maxTTL := 60 * time.Second

	store := NewBanStore(initialTTL, maxTTL, time.Now())

	now := time.Unix(1000, 0)

	result := store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if result.WasBanned {
		t.Fatal("first Ban() must not report existing ban")
	}

	if result.Level != 0 {
		t.Fatalf("expected level 0, got %d", result.Level)
	}

	expectedExpiresAt := now.Add(initialTTL)

	if !result.ExpiresAt.Equal(expectedExpiresAt) {
		t.Fatalf(
			"expected expiration %v, got %v",
			expectedExpiresAt,
			result.ExpiresAt,
		)
	}

	if result.Symbol != "ETHUSDT" {
		t.Fatalf(
			"expected symbol ETHUSDT, got %s",
			result.Symbol,
		)
	}

	if !store.IsBanned("route-1", now) {
		t.Fatal("route must be banned immediately after Ban()")
	}
}


func TestBanStore_ActiveBanIsNotExtended(t *testing.T) {
	initialTTL := time.Second
	maxTTL := 60 * time.Second

	store := NewBanStore(initialTTL, maxTTL, time.Now())

	now := time.Unix(1000, 0)

	first := store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	// Через 100ms маршрут снова попал в обработку.
	later := now.Add(100 * time.Millisecond)

	second := store.Ban(
		"route-1",
		"ETHUSDT",
		later,
	)

	if !second.WasBanned {
		t.Fatal("expected existing ban")
	}

	// ExpiresAt должен остаться от первого Ban().
	if !second.ExpiresAt.Equal(first.ExpiresAt) {
		t.Fatalf(
			"active ban was unexpectedly extended: first=%v second=%v",
			first.ExpiresAt,
			second.ExpiresAt,
		)
	}

	if second.Level != 0 {
		t.Fatalf(
			"expected level 0, got %d",
			second.Level,
		)
	}
}


func TestBanStore_IsBannedExpiresAtTTL(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if !store.IsBanned("route-1", now.Add(999*time.Millisecond)) {
		t.Fatal("route should still be banned before expiration")
	}

	if store.IsBanned("route-1", now.Add(time.Second)) {
		t.Fatal("route should not be banned exactly at expiration")
	}

	if store.IsBanned("route-1", now.Add(2*time.Second)) {
		t.Fatal("route should not be banned after expiration")
	}
}

func TestBanStore_BackoffDoublesAfterRepeatedFailure(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	// Первый failure.
	first := store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if first.Level != 0 {
		t.Fatalf("expected level 0, got %d", first.Level)
	}

	if first.Duration != time.Second {
		t.Fatalf(
			"expected 1s duration, got %v",
			first.Duration,
		)
	}

	// Первый ban закончился.
	now = now.Add(time.Second)

	// Второй failure.
	second := store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if second.Level != 1 {
		t.Fatalf("expected level 1, got %d", second.Level)
	}

	if second.Duration != 2*time.Second {
		t.Fatalf(
			"expected 2s duration, got %v",
			second.Duration,
		)
	}

	// Второй ban закончился.
	now = now.Add(2 * time.Second)

	// Третий failure.
	third := store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if third.Level != 2 {
		t.Fatalf("expected level 2, got %d", third.Level)
	}

	if third.Duration != 4*time.Second {
		t.Fatalf(
			"expected 4s duration, got %v",
			third.Duration,
		)
	}
}

func TestBanStore_BackoffStopsAtMaxTTL(t *testing.T) {
	store := NewBanStore(
		time.Second,
		4*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	// 1s
	result := store.Ban("route-1", "ETHUSDT", now)

	if result.Duration != time.Second {
		t.Fatalf("expected 1s, got %v", result.Duration)
	}

	// 2s
	now = result.ExpiresAt

	result = store.Ban("route-1", "ETHUSDT", now)

	if result.Duration != 2*time.Second {
		t.Fatalf("expected 2s, got %v", result.Duration)
	}

	// 4s
	now = result.ExpiresAt

	result = store.Ban("route-1", "ETHUSDT", now)

	if result.Duration != 4*time.Second {
		t.Fatalf("expected 4s, got %v", result.Duration)
	}

	// Ещё один failure.
	// Должно остаться 4s.
	now = result.ExpiresAt

	result = store.Ban("route-1", "ETHUSDT", now)

	if result.Duration != 4*time.Second {
		t.Fatalf(
			"expected max TTL 4s, got %v",
			result.Duration,
		)
	}
}

func TestBanStore_ResetWrongSymbolDoesNothing(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	ok := store.Reset(
		"route-1",
		"BTCUSDT",
	)

	if ok {
		t.Fatal("Reset with wrong symbol must return false")
	}

	if !store.IsBanned("route-1", now) {
		t.Fatal("route must still be banned")
	}
}

func TestBanStore_ResetCorrectSymbol(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	ok := store.Reset(
		"route-1",
		"ETHUSDT",
	)

	if !ok {
		t.Fatal("Reset with correct symbol must return true")
	}

	if store.IsBanned("route-1", now) {
		t.Fatal("route must no longer be banned after Reset")
	}
}

func TestBanStore_ResetResetsBackoff(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	// 1s
	result := store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	now = result.ExpiresAt

	// 2s
	result = store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if result.Level != 1 {
		t.Fatalf("expected level 1, got %d", result.Level)
	}

	if result.Duration != 2*time.Second {
		t.Fatalf(
			"expected 2s, got %v",
			result.Duration,
		)
	}

	// ETHUSDT восстановился.
	if !store.Reset("route-1", "ETHUSDT") {
		t.Fatal("expected Reset to succeed")
	}

	// Следующий failure должен снова начать с 1s.
	result = store.Ban(
		"route-1",
		"ETHUSDT",
		now,
	)

	if result.Level != 0 {
		t.Fatalf(
			"expected level 0 after Reset, got %d",
			result.Level,
		)
	}

	if result.Duration != time.Second {
		t.Fatalf(
			"expected 1s after Reset, got %v",
			result.Duration,
		)
	}
}

func TestBanStore_RoutesAreIndependent(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	resultA := store.Ban(
		"route-A",
		"ETHUSDT",
		now,
	)

	resultB := store.Ban(
		"route-B",
		"BTCUSDT",
		now,
	)

	if resultA.Level != 0 {
		t.Fatalf("route A expected level 0, got %d", resultA.Level)
	}

	if resultB.Level != 0 {
		t.Fatalf("route B expected level 0, got %d", resultB.Level)
	}

	// Увеличиваем backoff только A.
	now = resultA.ExpiresAt

	resultA = store.Ban(
		"route-A",
		"ETHUSDT",
		now,
	)

	if resultA.Level != 1 {
		t.Fatalf("route A expected level 1, got %d", resultA.Level)
	}

	// B всё ещё имеет свой собственный ban.
	if !store.IsBanned("route-B", now) {
		t.Fatal("route B state must be independent")
	}
}

func TestBanStore_ConcurrentAccess(t *testing.T) {
	store := NewBanStore(
		time.Second,
		60*time.Second,
		time.Now(),
	)

	now := time.Unix(1000, 0)

	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func(i int) {
			key := RouteKey("route-1")

			store.IsBanned(key, now)

			store.Ban(
				key,
				"ETHUSDT",
				now,
			)

			store.Reset(
				key,
				"ETHUSDT",
			)

			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}