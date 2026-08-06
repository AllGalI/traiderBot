package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	apiKey    string
	secretKey string

	baseURL string

	http *http.Client
}

func NewClient() *Client {

	return &Client{
		apiKey:    os.Getenv("API_KEY"),
		secretKey: os.Getenv("SECRET_KEY"),
		// apiKey:    os.Getenv("TESTNET_API_KEY"),
		// secretKey: os.Getenv("TESTNET_SECRET_KEY"),

		baseURL: "https://api.binance.com",
		// baseURL: "https://testnet.binance.vision",

		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	values url.Values,
	signed bool,
	dst any,
) error {

	if values == nil {
		values = url.Values{}
	}

	if signed {
		values.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		values.Set("recvWindow", "5000")

		signature := c.sign(values.Encode())
		values.Set("signature", signature)
	}

	endpoint := c.baseURL + path

	if method == http.MethodGet || method == http.MethodDelete {
		endpoint += "?" + values.Encode()
	}

	var body io.Reader

	if method == http.MethodPost || method == http.MethodPut {
		body = strings.NewReader(values.Encode())
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		endpoint,
		body,
	)
	if err != nil {
		return err
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {

		var apiErr APIError

		if err := json.Unmarshal(data, &apiErr); err == nil {
			return apiErr
		}

		return fmt.Errorf(
			"binance http %d: %s",
			resp.StatusCode,
			string(data),
		)
	}

	if dst == nil {
		return nil
	}

	return json.Unmarshal(data, dst)
}

func (c *Client) sign(payload string) string {

	mac := hmac.New(
		sha256.New,
		[]byte(c.secretKey),
	)

	mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}
