package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	binanceRest "traiderBot/internalDDD/exchange/infrastructure/binance/rest"
	"traiderBot/internalDDD/market/domain"
)

type SymbolProvider struct {
	client *binanceRest.Client
}

func NewSymbolProvider(
	client *binanceRest.Client,
) *SymbolProvider {
	return &SymbolProvider{
		client: client,
	}
}

func (p *SymbolProvider) Get(
	ctx context.Context,
	symbols []string,
) ([]domain.Symbol, error) {

	if len(symbols) == 0 {
		return nil, fmt.Errorf(
			"no symbols provided",
		)
	}

	requested := make(map[string]struct{}, len(symbols))

	for _, symbol := range symbols {

		symbol = strings.ToUpper(
			strings.TrimSpace(symbol),
		)

		if symbol == "" {
			continue
		}

		requested[symbol] = struct{}{}
	}

	if len(requested) == 0 {
		return nil, fmt.Errorf(
			"no valid symbols provided",
		)
	}

	symbolList := make([]string, 0, len(requested))

	for symbol := range requested {
		symbolList = append(
			symbolList,
			symbol,
		)
	}

	symbolsJSON, err := json.Marshal(
		symbolList,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"encode symbols: %w",
			err,
		)
	}

	body, err := p.client.Request(
		ctx,
		"GET",
		exchangeInfoPath,
		map[string][]string{
			"symbols": {
				string(symbolsJSON),
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf(
			"request exchange info: %w",
			err,
		)
	}

	return p.parseSymbols(
		body,
		requested,
	)
}

func (p *SymbolProvider) GetSymbols(
	ctx context.Context,
) ([]domain.Symbol, error) {

	body, err := p.client.Request(
		ctx,
		"GET",
		exchangeInfoPath,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"request exchange info: %w",
			err,
		)
	}

	return p.parseSymbols(
		body,
		nil,
	)
}

func (p *SymbolProvider) parseSymbols(
	body []byte,
	requested map[string]struct{},
) ([]domain.Symbol, error) {

	var dto exchangeInfoDTO

	if err := json.Unmarshal(
		body,
		&dto,
	); err != nil {

		return nil, fmt.Errorf(
			"decode exchange info response: %w",
			err,
		)
	}

	result := make(
		[]domain.Symbol,
		0,
		len(dto.Symbols),
	)

	for _, symbolDTO := range dto.Symbols {

		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbolName := strings.ToUpper(
			symbolDTO.Symbol,
		)

		if requested != nil {

			if _, ok := requested[symbolName]; !ok {
				continue
			}
		}

		symbol, err := mapSymbol(
			symbolDTO,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"map symbol %q: %w",
				symbolDTO.Symbol,
				err,
			)
		}

		result = append(
			result,
			symbol,
		)
	}

	return result, nil
}
