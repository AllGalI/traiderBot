package binance

import "errors"

var ErrClientNotConfigured = errors.New(
	"binance client is not configured",
)
