package application

import (
	"context"
	"fmt"
)

type MarketStructureBuilder interface {
	Build(
		ctx context.Context,
		startAssets map[string]struct{},
	) (MarketStructure, error)
}

type MarketDataStarter interface {
	Start(
		ctx context.Context,
		symbols []string,
		depth int,
	) error
}

type MarketService struct {
	cache            MarketCache
	structureService MarketStructureBuilder
	marketDataService MarketDataStarter
}

func NewMarketService(
	cache MarketCache,
	structureService MarketStructureBuilder,
	marketDataService MarketDataStarter,
) *MarketService {
	return &MarketService{
		cache:             cache,
		structureService:  structureService,
		marketDataService: marketDataService,
	}
}

func (s *MarketService) Start(
	ctx context.Context,
	startAssets map[string]struct{},
	depth int,
) error {
	structure, ok, err := s.cache.Get(ctx)
	if err != nil {
		return fmt.Errorf(
			"get market structure from cache: %w",
			err,
		)
	}

	if !ok {
		structure, err = s.structureService.Build(
			ctx,
			startAssets,
		)
		if err != nil {
			return fmt.Errorf(
				"build market structure: %w",
				err,
			)
		}

		if err := s.cache.Set(ctx, structure); err != nil {
			return fmt.Errorf(
				"cache market structure: %w",
				err,
			)
		}
	}

	symbols := structure.RouteIndex.Symbols()

	if len(symbols) == 0 {
		return fmt.Errorf("no symbols found in market routes")
	}

	if err := s.marketDataService.Start(
		ctx,
		symbols,
		depth,
	); err != nil {
		return fmt.Errorf(
			"start market data service: %w",
			err,
		)
	}

	return nil
}