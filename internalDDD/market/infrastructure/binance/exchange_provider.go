package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"traiderBot/internalDDD/market/domain"
)


type ExchangeProvider struct {
	client  *http.Client
	baseURL string
}

func NewExchangeProvider() *ExchangeProvider {
	return &ExchangeProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
	}
}

func (p *ExchangeProvider) GetSymbols(
	ctx context.Context,
) ([]domain.Symbol, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		p.baseURL+exchangeInfoPath,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create exchange info request: %w", err)
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
		return nil, fmt.Errorf("decode exchange info response: %w", err)
	}

	symbols := make([]domain.Symbol, 0, len(dto.Symbols))

	for _, symbolDTO := range dto.Symbols {
		if symbolDTO.Status != "TRADING" {
			continue
		}

		symbol, err := mapSymbol(symbolDTO)
		if err != nil {
			// Некорректная отдельная пара не должна ломать
			// загрузку всего рынка.
			continue
		}

		symbols = append(symbols, symbol)
	}

	return symbols, nil
}
