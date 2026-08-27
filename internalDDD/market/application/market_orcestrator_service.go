package application

import (
	"context"
	"fmt"
	"log"
	"traiderBot/internalDDD/market/domain"
)

type MarketStructureBuilder interface {
	Build(
		ctx context.Context,
		startAssets map[string]struct{},
	) (domain.MarketStructure, error)
}

type MarketDataStarter interface {
	Start(
		ctx context.Context,
		symbols []string,
		depth int,
	) error
}

type MarketService struct {
	cache             MarketCache
	structureService  MarketStructureBuilder
	marketDataService MarketDataStarter
	structureStore    MarketStructureStore

	ready chan struct{}
}

func NewMarketService(
	cache MarketCache,
	structureService MarketStructureBuilder,
	marketDataService MarketDataStarter,
	structureStore MarketStructureStore,
) *MarketService {
	return &MarketService{
		cache:             cache,
		structureService:  structureService,
		marketDataService: marketDataService,
		structureStore:    structureStore,
		ready:             make(chan struct{}),
	}
}
func (s *MarketService) Start(
	ctx context.Context,
	startAssets map[string]struct{},
	depth, symbolsLimit int,
) error {
	log.Println("MarketService: START")

	structure, ok, err := s.cache.Get(ctx)
	if err != nil {
		return fmt.Errorf(
			"get market structure from cache: %w",
			err,
		)
	}

	log.Printf("MarketService: cache hit=%v", ok)

	if !ok {
		log.Println("MarketService: building structure")

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

		log.Println("MarketService: structure built")

		if err := s.cache.Set(ctx, structure); err != nil {
			return fmt.Errorf(
				"cache market structure: %w",
				err,
			)
		}

		log.Println("MarketService: structure cached")
	}
	s.structureStore.Set(&structure)
	close(s.ready)

	symbols := structure.RouteIndex.Symbols()

	log.Printf(
		"MarketService: total symbols=%d, limit=%d",
		len(symbols),
		symbolsLimit,
	)

	symbols = symbols[:symbolsLimit]

	log.Printf(
		"MarketService: selected symbols=%v",
		symbols,
	)

	if len(symbols) == 0 {
		return fmt.Errorf("no symbols found in market routes")
	}

	log.Println("MarketService: starting market data")

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

	log.Println("MarketService: market data stopped")

	return nil
}

func (s *MarketService) Ready() <-chan struct{} {
	return s.ready
}
