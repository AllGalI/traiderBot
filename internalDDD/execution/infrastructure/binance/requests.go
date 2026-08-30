package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (c *Client) signedRequest(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}

	params.Set(
		"timestamp",
		strconv.FormatInt(time.Now().UnixMilli(), 10),
	)

	query := params.Encode()

	signature := sign(
		query,
		c.apiSecret,
	)

	query += "&signature=" + signature

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		c.baseURL+path+"?"+query,
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"X-MBX-APIKEY",
		c.apiKey,
	)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"binance api error: status=%d body=%s",
			resp.StatusCode,
			string(body),
		)
	}

	return body, nil
}

func sign(
	payload string,
	secret string,
) string {
	mac := hmac.New(
		sha256.New,
		[]byte(secret),
	)

	_, _ = mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}
