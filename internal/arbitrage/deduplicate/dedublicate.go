package deduplicate

import (
	"sync"
	"time"
)

type SeenOpportunity struct {
	LastSeen time.Time
}

type DedupStore struct {
	mu     sync.Mutex
	seen   map[string]SeenOpportunity
	window time.Duration
}

func NewDedupStore(window time.Duration) *DedupStore {
	return &DedupStore{
		seen:   make(map[string]SeenOpportunity),
		window: window,
	}
}

// IsDuplicate возвращает true, если маршрут уже встречался в пределах window.
func (d *DedupStore) IsDuplicate(routeKey string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	seen, ok := d.seen[routeKey]
	if !ok {
		return false
	}
	return now.Sub(seen.LastSeen) < d.window
}

// MarkSeen помечает маршрут как только что увиденный.
func (d *DedupStore) MarkSeen(routeKey string, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seen[routeKey] = SeenOpportunity{LastSeen: now}
}

// Cleanup удаляет записи старше maxAge.
func (d *DedupStore) Cleanup(maxAge time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for key, val := range d.seen {
		if val.LastSeen.Before(cutoff) {
			delete(d.seen, key)
		}
	}
}
