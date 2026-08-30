package binance

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

func LoadConfig(useTestnet bool) (Config, error) {
	_ = godotenv.Load()

	config := Config{}

	if useTestnet {
		config.APIKey = os.Getenv("TESTNET_API_KEY")
		config.APISecret = os.Getenv("TESTNET_SECRET_KEY")
		config.BaseURL = "https://testnet.binance.vision"
	} else {
		config.APIKey = os.Getenv("API_KEY")
		config.APISecret = os.Getenv("SECRET_KEY")
		config.BaseURL = "https://api.binance.com"
	}

	if config.APIKey == "" {
		return Config{}, fmt.Errorf(
			"binance api key is not configured",
		)
	}

	if config.APISecret == "" {
		return Config{}, fmt.Errorf(
			"binance secret key is not configured",
		)
	}

	return config, nil
}
