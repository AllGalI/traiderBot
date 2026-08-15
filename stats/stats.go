package stats

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var lifetimeBuckets = []time.Duration{
	100 * time.Millisecond,
	200 * time.Millisecond,
	300 * time.Millisecond,
	500 * time.Millisecond,
	750 * time.Millisecond,
	1 * time.Second,
	1500 * time.Millisecond,
	2 * time.Second,
	3 * time.Second,
	5 * time.Second,
	7 * time.Second,
	10 * time.Second,
	15 * time.Second,
	20 * time.Second,
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

type Histogram struct {
	Buckets []uint64 `json:"buckets"`
	Count   uint64   `json:"count"`
}

func NewHistogram() Histogram {
	return Histogram{
		Buckets: make([]uint64, len(lifetimeBuckets)+1),
	}
}

func (h *Histogram) Add(value time.Duration) {
	index := sort.Search(
		len(lifetimeBuckets),
		func(i int) bool {
			return value <= lifetimeBuckets[i]
		},
	)

	h.Buckets[index]++
	h.Count++
}

func (h Histogram) Percentile(percentile float64) time.Duration {
	if h.Count == 0 {
		return 0
	}

	if percentile <= 0 {
		return 0
	}

	if percentile >= 1 {
		percentile = 1
	}

	target := uint64(float64(h.Count-1) * percentile)

	var cumulative uint64

	for i, count := range h.Buckets {
		cumulative += count

		if cumulative > target {
			if i >= len(lifetimeBuckets) {
				return lifetimeBuckets[len(lifetimeBuckets)-1]
			}

			return lifetimeBuckets[i]
		}
	}

	return lifetimeBuckets[len(lifetimeBuckets)-1]
}

type LifetimeStats struct {
	Histogram Histogram `json:"histogram"`

	P50 time.Duration `json:"p50"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`

	LastUpdateID uint64 `json:"last_update_id"`
}

func (s *LifetimeStats) Add(lifetime time.Duration) {
	s.Histogram.Add(lifetime)
}

func (s *LifetimeStats) Recalculate() {
	s.P50 = s.Histogram.Percentile(0.50)
	s.P95 = s.Histogram.Percentile(0.95)
	s.P99 = s.Histogram.Percentile(0.99)
}

type FreshnessStats struct {
	Symbols map[string]LifetimeStats `json:"symbols"`
}

func NewFreshnessStats() FreshnessStats {
	return FreshnessStats{
		Symbols: make(map[string]LifetimeStats),
	}
}

func AppendRowToCSV(pathToFolder, filename string, values ...any) error {
	if err := os.MkdirAll(pathToFolder, 0755); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	filePath := filepath.Join(pathToFolder, filename)

	file, err := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat csv file: %w", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Если файл новый — записываем header.
	if info.Size() == 0 {
		header := []string{
			"time",
			"symbol_checks_counter",
			"old_symbols_count",
			"total_simulations_started",
			"dubs_check_counter",
			"unfresh_routes_counter",
			"fresh_routes_counter",
			"wrong_start_amount",
			"ok_start_amount_counter",
			"total_simulations",
			"top_percent",
			"average_percent",
		}

		if err := writer.Write(header); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
	}

	// Преобразуем любые типы в string.
	dataToWrite := make([]string, 0, len(values))

	for _, value := range values {
		dataToWrite = append(dataToWrite, fmt.Sprint(value))
	}

	if err := writer.Write(dataToWrite); err != nil {
		return fmt.Errorf("write csv row: %w", err)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}
