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

	Bans     map[RouteKey]BanState     `json:"bans"`
	Backoffs map[RouteKey]BackoffState `json:"backoffs"`

	Wheel *TimeWheel `json:"wheel"`

	InitialTTL time.Duration `json:"initialttl"`
	MaxTTL     time.Duration `json:"maxttl"`
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
		Bans:     make(map[RouteKey]BanState),
		Backoffs: make(map[RouteKey]BackoffState),

		Wheel: NewTimeWheel(
			bucketDuration,
			bucketCount,
			now,
		),

		InitialTTL: initialTTL,
		MaxTTL:     maxTTL,
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

	ban, ok := b.Bans[key]
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
	if existing, ok := b.Bans[key]; ok {
		if now.Before(existing.ExpiresAt) {
			duration := existing.ExpiresAt.Sub(now)

			backoff := b.Backoffs[key]

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
	backoff := b.Backoffs[key]

	duration := b.durationForLevel(backoff.Level)

	expiresAt := now.Add(duration)

	b.Bans[key] = BanState{
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
	if existing, ok := b.Bans[key]; ok {
		if now.Before(existing.ExpiresAt) {
			backoff := b.Backoffs[key]

			return BanResult{
				WasBanned: true,
				ExpiresAt: existing.ExpiresAt,
				Duration:  existing.ExpiresAt.Sub(now),
				Level:     backoff.Level,
				Symbol:    existing.Symbol,
			}
		}
	}

	backoff := b.Backoffs[key]

	duration := b.durationForLevel(backoff.Level)
	expiresAt := now.Add(duration)

	b.Bans[key] = BanState{
		ExpiresAt: expiresAt,
		Symbol:    symbol,
	}

	backoff.Symbol = symbol

	// Следующий failure будет использовать
	// следующий уровень.
	if duration < b.MaxTTL {
		backoff.Level++
	}

	b.Backoffs[key] = backoff

	b.Wheel.Add(
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
	duration := b.InitialTTL

	for i := uint8(0); i < level; i++ {
		if duration >= b.MaxTTL/2 {
			return b.MaxTTL
		}

		duration *= 2
	}

	if duration > b.MaxTTL {
		return b.MaxTTL
	}

	return duration
}

func (b *BanStore) Reset(
	key RouteKey,
	symbol string,
) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	backoff, ok := b.Backoffs[key]
	if !ok {
		return false
	}

	if backoff.Symbol != symbol {
		return false
	}

	delete(b.Bans, key)
	delete(b.Backoffs, key)

	return true
}

func (b *BanStore) DeleteExpired(
	key RouteKey,
	expiresAt time.Time,
	now time.Time,
) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	ban, ok := b.Bans[key]
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

	delete(b.Bans, key)

	return true
}

func (b *BanStore) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.Bans)
}

func (b *BanStore) cleanupExpired(
	key RouteKey,
	expiresAt time.Time,
	now time.Time,
) {
	ban, ok := b.Bans[key]
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

	delete(b.Bans, key)
}

func (b *BanStore) Advance(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Wheel.Advance(
		now,
		func(key RouteKey, expiresAt time.Time) {
			ban, ok := b.Bans[key]
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

			delete(b.Bans, key)
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
