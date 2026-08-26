package application

import (
	"context"
	"errors"
	"testing"

	"traiderBot/internalDDD/market/domain"
)

type mockSymbolProvider struct {
	symbols []domain.Symbol
	err     error
}

func (m *mockSymbolProvider) Get(
	ctx context.Context,
	symbols []string,
) ([]domain.Symbol, error) {
	return nil, nil
}

func (m *mockSymbolProvider) GetSymbols(
	ctx context.Context,
) ([]domain.Symbol, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.symbols, nil
}

func TestMarketStructureServiceBuild(t *testing.T) {
	provider := &mockSymbolProvider{
		symbols: []domain.Symbol{
			{
				Name:       "BTCUSDT",
				BaseAsset:  "BTC",
				QuoteAsset: "USDT",
			},
			{
				Name:       "ETHBTC",
				BaseAsset:  "ETH",
				QuoteAsset: "BTC",
			},
			{
				Name:       "ETHUSDT",
				BaseAsset:  "ETH",
				QuoteAsset: "USDT",
			},
		},
	}

	service := NewMarketStructureService(provider)

	structure, err := service.Build(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(structure.Symbols) != 3 {
		t.Fatalf(
			"expected 3 symbols, got %d",
			len(structure.Symbols),
		)
	}

	if len(structure.Graph) == 0 {
		t.Fatal("expected non-empty graph")
	}

	if len(structure.Routes) != 2 {
		t.Fatalf(
			"expected 2 routes, got %d",
			len(structure.Routes),
		)
	}

	for _, route := range structure.Routes {
		if !route.IsTriangle() {
			t.Fatalf(
				"expected triangle route, got %v",
				route.Symbols(),
			)
		}

		if route.Edges[0].From != "USDT" {
			t.Fatalf(
				"expected route to start from USDT, got %q",
				route.Edges[0].From,
			)
		}
	}
}

func TestMarketStructureServiceBuildProviderError(t *testing.T) {
	expectedErr := errors.New("provider error")

	provider := &mockSymbolProvider{
		err: expectedErr,
	}

	service := NewMarketStructureService(provider)

	_, err := service.Build(
		context.Background(),
		map[string]struct{}{
			"USDT": {},
		},
	)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped provider error, got %v",
			err,
		)
	}
}