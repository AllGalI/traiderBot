package banstore

import (
	"testing"
	"time"
)

func TestTimeWheel_ExpiresEntry(t *testing.T) {
	start := time.Unix(1000, 0)

	wheel := NewTimeWheel(
		100*time.Millisecond,
		10,
		start,
	)

	expiresAt := start.Add(300 * time.Millisecond)

	wheel.Add(
		"route-1",
		expiresAt,
	)

	var expired bool

	wheel.Advance(
		start.Add(100*time.Millisecond),
		func(key RouteKey, expiresAt time.Time) {
			expired = true
		},
	)

	if expired {
		t.Fatal("entry expired too early")
	}

	wheel.Advance(
		start.Add(300*time.Millisecond),
		func(key RouteKey, expiresAt time.Time) {
			expired = true
		},
	)

	if !expired {
		t.Fatal("entry did not expire")
	}
}

func TestTimeWheel_MultipleRounds(t *testing.T) {
	start := time.Unix(1000, 0)

	// 100ms × 10 = 1 second per full round.
	wheel := NewTimeWheel(
		100*time.Millisecond,
		10,
		start,
	)

	// 2.3 секунды = два полных оборота + 300ms.
	expiresAt := start.Add(2300 * time.Millisecond)

	wheel.Add(
		"route-1",
		expiresAt,
	)

	expired := false

	// Через первый оборот.
	wheel.Advance(
		start.Add(1000*time.Millisecond),
		func(key RouteKey, expiresAt time.Time) {
			expired = true
		},
	)

	if expired {
		t.Fatal("entry expired after first round")
	}

	// Через второй оборот.
	wheel.Advance(
		start.Add(2000*time.Millisecond),
		func(key RouteKey, expiresAt time.Time) {
			expired = true
		},
	)

	if expired {
		t.Fatal("entry expired after second round")
	}

	// Теперь должно истечь.
	wheel.Advance(
		start.Add(2300*time.Millisecond),
		func(key RouteKey, expiresAt time.Time) {
			expired = true
		},
	)

	if !expired {
		t.Fatal("entry did not expire")
	}
}

func TestTimeWheel_OldEntryCannotDeleteNewBan(t *testing.T) {
	start := time.Unix(1000, 0)

	store := NewBanStore(
		time.Second,
		60*time.Second,
		start,
	)

	// Первый ban.
	first := store.RecordFailure(
		"route-1",
		"ETHUSDT",
		start,
	)

	// Дошли до expiration.
	now := first.ExpiresAt

	if store.IsBanned("route-1", now) {
		t.Fatal("first ban should be expired")
	}

	// Представим, что до Advance() произошёл новый failure.
	second := store.RecordFailure(
		"route-1",
		"ETHUSDT",
		now,
	)

	// Теперь старый wheel entry может сработать.
	store.Advance(now)

	// Новый ban не должен быть удалён.
	if !store.IsBanned(
		"route-1",
		second.ExpiresAt.Add(-time.Millisecond),
	) {
		t.Fatal("new ban was incorrectly deleted by old wheel entry")
	}
}

func TestBanStore_Reset(t *testing.T) {
	start := time.Unix(1000, 0)

	store := NewBanStore(
		time.Second,
		60*time.Second,
		start,
	)

	first := store.RecordFailure(
		"route-1",
		"ETHUSDT",
		start,
	)

	// Второй failure → level 1.
	now := first.ExpiresAt

	second := store.RecordFailure(
		"route-1",
		"ETHUSDT",
		now,
	)

	if second.Level != 1 {
		t.Fatalf("expected level 1, got %d", second.Level)
	}

	// ETHUSDT восстановился.
	if !store.Reset(
		"route-1",
		"ETHUSDT",
	) {
		t.Fatal("expected Reset to succeed")
	}

	// Следующий failure должен снова быть level 0 / 1s.
	third := store.RecordFailure(
		"route-1",
		"ETHUSDT",
		now,
	)

	if third.Level != 0 {
		t.Fatalf(
			"expected level 0 after reset, got %d",
			third.Level,
		)
	}

	if third.Duration != time.Second {
		t.Fatalf(
			"expected 1s after reset, got %v",
			third.Duration,
		)
	}
}