package application

import (
	"context"
	"log"
	"time"
)

type FeeRefreshWorker struct {
	loader          FeeLoader
	cache           *FeeCache
	marketStructure MarketStructureReader
	interval        time.Duration
}

func NewFeeRefreshWorker(
	loader FeeLoader,
	cache *FeeCache,
	marketStructure MarketStructureReader,
	interval time.Duration,
) *FeeRefreshWorker {
	return &FeeRefreshWorker{
		loader:          loader,
		cache:           cache,
		marketStructure: marketStructure,
		interval:        interval,
	}
}

func (w *FeeRefreshWorker) Run(ctx context.Context) error {
	if err := w.Refresh(ctx); err != nil {
		log.Printf(
			"initial fee refresh failed: %v",
			err,
		)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := w.Refresh(ctx); err != nil {
				log.Printf(
					"fee refresh failed: %v",
					err,
				)
			}
		}
	}
}

func (w *FeeRefreshWorker) Refresh(
	ctx context.Context,
) error {
	structure, ok := w.marketStructure.Get()
	if !ok {
		return nil
	}

	for _, symbol := range structure.Symbols {
		if err := ctx.Err(); err != nil {
			return err
		}

		fee, err := w.loader.Get(
			ctx,
			symbol.Name,
		)
		if err != nil {
			log.Printf(
				"fee refresh failed for %s: %v",
				symbol.Name,
				err,
			)

			continue
		}

		w.cache.Set(
			symbol.Name,
			fee,
		)
	}

	return nil
}