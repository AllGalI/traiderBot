package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"traiderBot/internalDDD/market/domain"
)


type SymbolProvider struct {
	client  *http.Client
	baseURL string
}


func NewSymbolProvider() *SymbolProvider {
	return &SymbolProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
	}
}

func (p *SymbolProvider) Get(
	ctx context.Context,
	symbols []string,
) ([]domain.Symbol, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("no symbols provided")
	}

	requested := make(map[string]struct{}, len(symbols))

	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))

		if symbol == "" {
			continue
		}

		requested[symbol] = struct{}{}
	}

	if len(requested) == 0 {
		return nil, fmt.Errorf("no valid symbols provided")
	}

	symbolList := make([]string, 0, len(requested))

	for symbol := range requested {
		symbolList = append(symbolList, symbol)
	}

	symbolsJSON, err := json.Marshal(symbolList)
	if err != nil {
		return nil, fmt.Errorf("encode symbols: %w", err)
	}

	reqURL := fmt.Sprintf(
		"%s%s?symbols=%s",
		p.baseURL,
		exchangeInfoPath,
		url.QueryEscape(string(symbolsJSON)),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto exchangeInfoDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf(
			"decode exchange info response: %w",
			err,
		)
	}

	result := make([]domain.Symbol, 0, len(dto.Symbols))

	for _, symbolDTO := range dto.Symbols {
		symbolName := strings.ToUpper(symbolDTO.Symbol)

		if _, ok := requested[symbolName]; !ok {
			continue
		}

		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbol, err := mapSymbol(symbolDTO)
		if err != nil {
			return nil, fmt.Errorf(
				"map symbol %q: %w",
				symbolDTO.Symbol,
				err,
			)
		}

		result = append(result, symbol)
	}

	return result, nil
}

func (p *SymbolProvider) GetSymbols(
	ctx context.Context,
) ([]domain.Symbol, error) {
	reqURL := fmt.Sprintf(
		"%s%s",
		p.baseURL,
		exchangeInfoPath,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"binance returned status %d",
			resp.StatusCode,
		)
	}

	var dto exchangeInfoDTO

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf(
			"decode exchange info response: %w",
			err,
		)
	}

	result := make([]domain.Symbol, 0, len(dto.Symbols))

	for _, symbolDTO := range dto.Symbols {
		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbol, err := mapSymbol(symbolDTO)
		if err != nil {
			return nil, fmt.Errorf(
				"map symbol %q: %w",
				symbolDTO.Symbol,
				err,
			)
		}

		result = append(result, symbol)
	}

	return result, nil
}
