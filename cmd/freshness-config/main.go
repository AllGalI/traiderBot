package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type Statistics struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	Symbols     map[string]SymbolStat     `json:"symbols"`
}

type SymbolStat struct {
	Count  int64   `json:"count"`
	P50MS  float64 `json:"p50_ms"`
	P75MS  float64 `json:"p75_ms"`
	P90MS  float64 `json:"p90_ms"`
	P95MS  float64 `json:"p95_ms"`
	P99MS  float64 `json:"p99_ms"`
	P999MS float64 `json:"p999_ms"`
	MaxMS  float64 `json:"max_ms"`
	MeanMS float64 `json:"mean_ms"`
}

type FreshnessConfig struct {
	GeneratedAt  time.Time              `json:"generated_at"`
	DefaultTTLMS int64                  `json:"default_ttl_ms"`
	Symbols      map[string]SymbolTTL   `json:"symbols"`
}

type SymbolTTL struct {
	TTLMS       int64   `json:"ttl_ms"`
	Percentile  string  `json:"percentile"`
	PercentileMS float64 `json:"percentile_ms"`
	SampleCount int64   `json:"sample_count"`
}

func main() {
	input := flag.String("input", "", "statistics JSON file")
	output := flag.String("output", "", "freshness config JSON file")

	safety := flag.Float64(
		"safety",
		1.25,
		"TTL multiplier applied to p99",
	)

	minTTL := flag.Duration(
		"min-ttl",
		time.Second,
		"minimum TTL",
	)

	maxTTL := flag.Duration(
		"max-ttl",
		time.Minute,
		"maximum TTL",
	)

	defaultTTL := flag.Duration(
		"default-ttl",
		30*time.Second,
		"default TTL for unknown symbols",
	)

	flag.Parse()

	if *input == "" || *output == "" {
		flag.Usage()
		os.Exit(2)
	}

	if *safety <= 0 {
		fatal("safety must be greater than zero")
	}

	if *minTTL <= 0 {
		fatal("min-ttl must be greater than zero")
	}

	if *maxTTL < *minTTL {
		fatal("max-ttl must be greater than or equal to min-ttl")
	}

	stats, err := loadStatistics(*input)
	if err != nil {
		fatal("load statistics: %v", err)
	}

	config := FreshnessConfig{
		GeneratedAt:  stats.GeneratedAt,
		DefaultTTLMS: defaultTTL.Milliseconds(),
		Symbols:      make(map[string]SymbolTTL, len(stats.Symbols)),
	}

	for symbol, stat := range stats.Symbols {
		if stat.Count <= 0 {
			continue
		}

		ttl := time.Duration(stat.P99MS*float64(time.Millisecond) * (*safety))

		if ttl < *minTTL {
			ttl = *minTTL
		}

		if ttl > *maxTTL {
			ttl = *maxTTL
		}

		config.Symbols[symbol] = SymbolTTL{
			TTLMS:        ttl.Milliseconds(),
			Percentile:   "p99",
			PercentileMS: stat.P99MS,
			SampleCount:  stat.Count,
		}
	}

	if err := writeConfig(*output, config); err != nil {
		fatal("write config: %v", err)
	}

	fmt.Printf(
		"generated freshness config: %s (%d symbols)\n",
		*output,
		len(config.Symbols),
	)
}

func loadStatistics(path string) (Statistics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Statistics{}, err
	}

	var stats Statistics

	if err := json.Unmarshal(data, &stats); err != nil {
		return Statistics{}, err
	}

	if len(stats.Symbols) == 0 {
		return Statistics{}, fmt.Errorf("statistics contain no symbols")
	}

	return stats, nil
}

func writeConfig(path string, config FreshnessConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	return os.WriteFile(path, data, 0644)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}