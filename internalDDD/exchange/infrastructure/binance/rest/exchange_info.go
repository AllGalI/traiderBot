package rest

import (
	"context"
	"net/url"
)


func (c *Client) GetExchangeInfo(
	ctx context.Context,
	symbols []string,
) ([]byte, error) {

	params := url.Values{}

	if len(symbols) > 0 {
		params.Set(
			"symbols",
			stringArrayToJSON(symbols),
		)
	}


	return c.Request(
		ctx,
		"GET",
		"/api/v3/exchangeInfo",
		params,
	)
}