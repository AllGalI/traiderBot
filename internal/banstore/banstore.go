package banstore

import (
	"context"
	"sync"
	"time"
	"traiderBot/stats"
)

type RouteKey string

type BanState struct {
	ExpiresAt time.Time
	Symbol    string
}

type BackoffState struct {
	Level  uint8
	Symbol string
}

type BanResult struct {
	WasBanned bool
	ExpiresAt time.Time
	Duration  time.Duration
	Level     uint8
	Symbol    string
}

type BanStore struct {
	mu sync.RWMutex

	bans     map[RouteKey]BanState
	backoffs map[RouteKey]BackoffState

	wheel *TimeWheel

	initialTTL time.Duration
	maxTTL     time.Duration
}

func NewBanStore(
	initialTTL time.Duration,
	maxTTL time.Duration,
	now time.Time,
) *BanStore {
	if initialTTL <= 0 {
		panic("initialTTL must be > 0")
	}

	if maxTTL < initialTTL {
		panic("maxTTL must be >= initialTTL")
	}

	const bucketDuration = 100 * time.Millisecond
	const bucketCount = 600

	return &BanStore{
		bans:     make(map[RouteKey]BanState),
		backoffs: make(map[RouteKey]BackoffState),

		wheel: NewTimeWheel(
			bucketDuration,
			bucketCount,
			now,
		),

		initialTTL: initialTTL,
		maxTTL:     maxTTL,
	}
}

// IsBanned только проверяет состояние.
// Expired записи пока не удаляет.
// Cleanup добавим позже через time wheel.
func (b *BanStore) IsBanned(
	key RouteKey,
	now time.Time,
) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	ban, ok := b.bans[key]
	if !ok {
		return false
	}

	return now.Before(ban.ExpiresAt)
}

// Ban создаёт новый бан.
//
// Если маршрут уже имеет действующий бан, Ban ничего не меняет.
// Для продления существующего бана используется Renew.
func (b *BanStore) Ban(
	key RouteKey,
	symbol string,
	now time.Time,
) BanResult {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Если cooldown ещё активен —
	// ничего не меняем.
	if existing, ok := b.bans[key]; ok {
		if now.Before(existing.ExpiresAt) {
			duration := existing.ExpiresAt.Sub(now)

			backoff := b.backoffs[key]

			return BanResult{
				WasBanned: true,
				ExpiresAt: existing.ExpiresAt,
				Duration:  duration,
				Level:     backoff.Level,
				Symbol:    existing.Symbol,
			}
		}
	}

	// Cooldown закончился.
	//
	// Берём текущий backoff level.
	backoff := b.backoffs[key]

	duration := b.durationForLevel(backoff.Level)

	expiresAt := now.Add(duration)

	b.bans[key] = BanState{
		ExpiresAt: expiresAt,
		Symbol:    symbol,
	}

	return BanResult{
		WasBanned: false,
		ExpiresAt: expiresAt,
		Duration:  duration,
		Level:     backoff.Level,
		Symbol:    symbol,
	}
}

func (b *BanStore) RecordFailure(
	key RouteKey,
	symbol string,
	now time.Time,
) BanResult {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Защита от случайного повторного вызова.
	if existing, ok := b.bans[key]; ok {
		if now.Before(existing.ExpiresAt) {
			backoff := b.backoffs[key]

			return BanResult{
				WasBanned: true,
				ExpiresAt: existing.ExpiresAt,
				Duration:  existing.ExpiresAt.Sub(now),
				Level:     backoff.Level,
				Symbol:    existing.Symbol,
			}
		}
	}

	backoff := b.backoffs[key]

	duration := b.durationForLevel(backoff.Level)
	expiresAt := now.Add(duration)

	b.bans[key] = BanState{
		ExpiresAt: expiresAt,
		Symbol:    symbol,
	}

	backoff.Symbol = symbol

	// Следующий failure будет использовать
	// следующий уровень.
	if duration < b.maxTTL {
		backoff.Level++
	}

	b.backoffs[key] = backoff

	b.wheel.Add(
		key,
		expiresAt,
	)

	return BanResult{
		WasBanned: false,
		ExpiresAt: expiresAt,
		Duration:  duration,
		Level:     backoff.Level - 1,
		Symbol:    symbol,
	}
}

func (b *BanStore) durationForLevel(
	level uint8,
) time.Duration {
	duration := b.initialTTL

	for i := uint8(0); i < level; i++ {
		if duration >= b.maxTTL/2 {
			return b.maxTTL
		}

		duration *= 2
	}

	if duration > b.maxTTL {
		return b.maxTTL
	}

	return duration
}

func (b *BanStore) Reset(
	key RouteKey,
	symbol string,
) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	backoff, ok := b.backoffs[key]
	if !ok {
		return false
	}

	if backoff.Symbol != symbol {
		return false
	}

	delete(b.bans, key)
	delete(b.backoffs, key)

	return true
}

func (b *BanStore) DeleteExpired(
	key RouteKey,
	expiresAt time.Time,
	now time.Time,
) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	ban, ok := b.bans[key]
	if !ok {
		return false
	}

	// Старый элемент wheel не должен удалить
	// новый cooldown.
	if !ban.ExpiresAt.Equal(expiresAt) {
		return false
	}

	if now.Before(ban.ExpiresAt) {
		return false
	}

	delete(b.bans, key)

	return true
}

func (b *BanStore) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.bans)
}

func (b *BanStore) cleanupExpired(
	key RouteKey,
	expiresAt time.Time,
	now time.Time,
) {
	ban, ok := b.bans[key]
	if !ok {
		return
	}

	// Старый wheel entry.
	if !ban.ExpiresAt.Equal(expiresAt) {
		return
	}

	if now.Before(ban.ExpiresAt) {
		return
	}

	delete(b.bans, key)
}

func (b *BanStore) Advance(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.wheel.Advance(
		now,
		func(key RouteKey, expiresAt time.Time) {
			ban, ok := b.bans[key]
			if !ok {
				return
			}

			// Старый wheel entry.
			//
			// Например:
			//
			// old ban → expires 10:01
			// new ban → expires 10:02
			//
			// Старый entry не имеет права удалить новый ban.
			if !ban.ExpiresAt.Equal(expiresAt) {
				return
			}

			if now.Before(ban.ExpiresAt) {
				return
			}

			delete(b.bans, key)
		},
	)
}

func (b *BanStore) Start(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case now := <-ticker.C:
				b.Advance(now)

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (bS *BanStore) SaveLog() error {
	err := stats.SaveObj("./stats/banstore.json", bS)
	return err
}
