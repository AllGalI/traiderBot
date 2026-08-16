package banstore

import (
	"sync"
	"time"
	"traiderBot/stats"
)

type RouteKey string

type Ban struct {
	ExpiresAt time.Time
	Duration  time.Duration
	Level     uint8
}

type BanResult struct {
	WasBanned bool
	ExpiresAt time.Time
	Duration  time.Duration
	Level     uint8
}

type BanStore struct {
	mu sync.RWMutex

	bans map[RouteKey]Ban

	initialTTL time.Duration
	maxTTL     time.Duration
}

func NewBanStore(initialTTL, maxTTL time.Duration) *BanStore {
	if initialTTL <= 0 {
		panic("initialTTL must be > 0")
	}

	if maxTTL < initialTTL {
		panic("maxTTL must be >= initialTTL")
	}

	return &BanStore{
		bans:       make(map[RouteKey]Ban),
		initialTTL: initialTTL,
		maxTTL:     maxTTL,
	}
}

func (bS *BanStore) SaveLog() error {
	err := stats.SaveObj("./stats/banstore.json", bS)
	return err
}

// IsBanned возвращает true, если маршрут находится под действующим баном.
//
// Просроченный бан удаляется сразу при обращении к нему.
// Это пока простая реализация. На следующем этапе cleanup будет
// вынесен в отдельный механизм.
func (b *BanStore) IsBanned(key RouteKey, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	ban, ok := b.bans[key]
	if !ok {
		return false
	}

	if !now.Before(ban.ExpiresAt) {
		delete(b.bans, key)
		return false
	}

	return true
}

// Ban создаёт новый бан либо продлевает существующий.
//
// Для нового маршрута:
//
//	TTL = initialTTL
//
// Для уже забаненного:
//
//	TTL = previous TTL * 2
//
// TTL ограничивается maxTTL.
//
// Продление считается от now, а не от старого ExpiresAt.
func (b *BanStore) Ban(key RouteKey, now time.Time) BanResult {
	b.mu.Lock()
	defer b.mu.Unlock()

	existing, ok := b.bans[key]

	if !ok || !now.Before(existing.ExpiresAt) {
		ban := Ban{
			Duration:  b.initialTTL,
			ExpiresAt: now.Add(b.initialTTL),
			Level:     0,
		}

		b.bans[key] = ban

		return BanResult{
			WasBanned: false,
			ExpiresAt: ban.ExpiresAt,
			Duration:  ban.Duration,
			Level:     ban.Level,
		}
	}

	duration := existing.Duration * 2

	if duration > b.maxTTL {
		duration = b.maxTTL
	}

	level := existing.Level
	if existing.Duration < b.maxTTL {
		level++
	}

	ban := Ban{
		Duration:  duration,
		ExpiresAt: now.Add(duration),
		Level:     level,
	}

	b.bans[key] = ban

	return BanResult{
		WasBanned: true,
		ExpiresAt: ban.ExpiresAt,
		Duration:  ban.Duration,
		Level:     ban.Level,
	}
}

// Unban немедленно снимает бан.
func (b *BanStore) Unban(key RouteKey) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.bans, key)
}

// Len возвращает количество текущих записей.
//
// Пока это просто полезный диагностический метод.
// Позже пригодится для проверки эффективности cleanup.
func (b *BanStore) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.bans)
}
