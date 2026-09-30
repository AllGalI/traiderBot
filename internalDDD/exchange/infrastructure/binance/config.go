package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey     string
	APISecret  string
	RESTBaseURL string
	WSBaseURL   string
}

func LoadConfig(useTestnet bool) (Config, error) {
	_ = godotenv.Load()

	config := Config{}

	if useTestnet {
		config.APIKey = os.Getenv("TESTNET_API_KEY")
		config.APISecret = os.Getenv("TESTNET_SECRET_KEY")
		config.RESTBaseURL = "https://testnet.binance.vision"
		config.WSBaseURL = "wss://stream.testnet.binance.vision:9443"
	} else {
		config.APIKey = os.Getenv("API_KEY")
		config.APISecret = os.Getenv("SECRET_KEY")
		config.RESTBaseURL = "https://api.binance.com"
		config.WSBaseURL = "wss://stream.binance.com:9443"
	}

	if config.APIKey == "" {
		return Config{}, fmt.Errorf(
			"binance api key is not configured221",
		)
	}

	if config.APISecret == "" {
		return Config{}, fmt.Errorf(
			"binance secret key is not configured",
		)
	}

	return config, nil
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func DecodeJSON[T any](
	body []byte,
) (T, error) {
	var result T

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf(
			"decode binance response: %w",
			err,
		)
	}

	return result, nil
}

var ErrClientNotConfigured = errors.New(
	"binance client is not configured",
)