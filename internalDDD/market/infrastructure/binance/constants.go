package binance

import "time"

const (
	// restBaseURL = "https://api.binance.com"
	// wsBaseURL   = "wss://stream.binance.com:9443"
	
	restBaseURL = "https://testnet.binance.vision"
	wsBaseURL = "wss://stream.testnet.binance.vision:9443"

	exchangeInfoPath = "/api/v3/exchangeInfo"
	orderBookPath    = "/api/v3/depth"

	defaultHTTPTimeout        = 10 * time.Second
	defaultWSHandshakeTimeout = 10 * time.Second
)
