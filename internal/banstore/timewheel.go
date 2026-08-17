package banstore

import "time"

type wheelEntry struct {
	key       RouteKey
	expiresAt time.Time
	rounds    int
}

type TimeWheel struct {
	bucketDuration time.Duration
	buckets        [][]wheelEntry

	currentBucket int
	lastTick      time.Time
}

func NewTimeWheel(
	bucketDuration time.Duration,
	bucketCount int,
	startTime time.Time,
) *TimeWheel {
	if bucketDuration <= 0 {
		panic("bucketDuration must be > 0")
	}

	if bucketCount <= 0 {
		panic("bucketCount must be > 0")
	}

	return &TimeWheel{
		bucketDuration: bucketDuration,
		buckets:        make([][]wheelEntry, bucketCount),
		lastTick:       startTime,
	}
}

func (w *TimeWheel) Add(
	key RouteKey,
	expiresAt time.Time,
) {
	delta := expiresAt.Sub(w.lastTick)

	if delta < 0 {
		delta = 0
	}

	totalTicks := int(delta / w.bucketDuration)

	bucketOffset := totalTicks % len(w.buckets)
	rounds := totalTicks / len(w.buckets)

	bucket := (w.currentBucket + bucketOffset) % len(w.buckets)

	w.buckets[bucket] = append(
		w.buckets[bucket],
		wheelEntry{
			key:       key,
			expiresAt: expiresAt,
			rounds:    rounds,
		},
	)
}

func (w *TimeWheel) Advance(
	now time.Time,
	expire func(key RouteKey, expiresAt time.Time),
) {
	elapsed := now.Sub(w.lastTick)

	if elapsed < w.bucketDuration {
		return
	}

	ticks := int(elapsed / w.bucketDuration)

	// За один Advance не нужно проходить
	// больше одного полного круга.
	//
	// Если процесс был остановлен на 10 минут,
	// мы всё равно обойдём wheel один раз.
	if ticks > len(w.buckets) {
		ticks = len(w.buckets)
	}

	for i := 0; i < ticks; i++ {
		w.currentBucket++

		if w.currentBucket >= len(w.buckets) {
			w.currentBucket = 0
		}

		entries := w.buckets[w.currentBucket]

		// Bucket сразу очищаем.
		w.buckets[w.currentBucket] = nil

		for _, entry := range entries {
			if entry.rounds > 0 {
				entry.rounds--

				w.buckets[w.currentBucket] = append(
					w.buckets[w.currentBucket],
					entry,
				)

				continue
			}

			expire(
				entry.key,
				entry.expiresAt,
			)
		}
	}

	w.lastTick = w.lastTick.Add(
		time.Duration(ticks) * w.bucketDuration,
	)
}