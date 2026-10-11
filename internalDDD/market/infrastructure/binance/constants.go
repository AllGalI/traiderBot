package binance

import "time"

const (
	exchangeInfoPath = "/api/v3/exchangeInfo"
	orderBookPath    = "/api/v3/depth"

	defaultHTTPTimeout        = 10 * time.Second
	defaultWSHandshakeTimeout = 10 * time.Second
)
