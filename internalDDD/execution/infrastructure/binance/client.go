package binance

import (
	"fmt"
	"net/http"
)

const defaultBaseURL = "https://api.binance.com"

type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client
}

func NewClient(
	cfg Config,
	httpClient *http.Client,
) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("binance api key is empty")
	}

	if cfg.APISecret == "" {
		return nil, fmt.Errorf("binance api secret is empty")
	}

	if httpClient == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	return &Client{
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		baseURL:   cfg.BaseURL,
		http:      httpClient,
	}, nil
}
