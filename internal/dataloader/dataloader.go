package dataloader

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const exchangeInfoURL = "https://api.binance.com/api/v3/exchangeInfo"

type FeeRate struct {
	Maker float64
	Taker float64
}

type SymbolFilters struct {
	MinQty      float64
	MaxQty      float64
	StepSize    float64
	MinNotional float64
	TickSize    float64
}

type ExchangeInfo struct {
	Symbols []Symbol `json:"symbols"`
}

type Symbol struct {
	Symbol  string   `json:"symbol"`
	Status  string   `json:"status"`
	Filters []Filter `json:"filters"`
}

type Filter struct {
	FilterType string `json:"filterType"`

	MinQty   string `json:"minQty,omitempty"`
	MaxQty   string `json:"maxQty,omitempty"`
	StepSize string `json:"stepSize,omitempty"`

	MinNotional string `json:"minNotional,omitempty"`

	MinPrice string `json:"minPrice,omitempty"`
	MaxPrice string `json:"maxPrice,omitempty"`
	TickSize string `json:"tickSize,omitempty"`
}

func LoadExchangeInfo(ctx context.Context, count int) (
	[]string,
	map[string]SymbolFilters,
	error,
) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		exchangeInfoURL,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("status=%d", resp.StatusCode)
	}

	var info ExchangeInfo

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, nil, err
	}

	var pairs []string
	filters := make(map[string]SymbolFilters)
	i := 0
	for _, symbol := range info.Symbols {
		if i == count {
			break
		}
		if symbol.Status != "TRADING" {
			continue
		}

		pairs = append(pairs, symbol.Symbol)

		var sf SymbolFilters

		for _, f := range symbol.Filters {

			switch f.FilterType {

			case "LOT_SIZE":
				sf.MinQty = parseFloat(f.MinQty)
				sf.MaxQty = parseFloat(f.MaxQty)
				sf.StepSize = parseFloat(f.StepSize)

			case "PRICE_FILTER":
				sf.TickSize = parseFloat(f.TickSize)

			case "MIN_NOTIONAL":
				sf.MinNotional = parseFloat(f.MinNotional)

			case "NOTIONAL":
				// На новых символах Binance использует NOTIONAL
				if sf.MinNotional == 0 {
					sf.MinNotional = parseFloat(f.MinNotional)
				}
			}
		}

		filters[symbol.Symbol] = sf
		i++
	}

	return pairs, filters, nil
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

const commissionURL = "https://api.binance.com/api/v3/account/commission"

type Client struct {
	apiKey string
	secret string
	http   *http.Client
}

func NewClient() *Client {
	return &Client{
		apiKey: os.Getenv("API_KEY"),
		secret: os.Getenv("SECRET_KEY"),
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type commissionResponse struct {
	Symbol             string `json:"symbol"`
	StandardCommission struct {
		Maker string `json:"maker"`
		Taker string `json:"taker"`
	} `json:"standardCommission"`
}

func (c *Client) LoadFees(
	ctx context.Context,
	pairs []string,
) (map[string]FeeRate, error) {

	fees := make(map[string]FeeRate, len(pairs))

	for _, symbol := range pairs {

		fee, err := c.GetCommission(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", symbol, err)
		}

		fees[symbol] = fee
	}

	return fees, nil
}

func (c *Client) sign(query string) string {
	mac := hmac.New(sha256.New, []byte(c.secret))
	mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) GetCommission(
	ctx context.Context,
	symbol string,
) (FeeRate, error) {

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	query := params.Encode()
	signature := c.sign(query)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		commissionURL+"?"+query+"&signature="+signature,
		nil,
	)
	if err != nil {
		return FeeRate{}, err
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return FeeRate{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FeeRate{}, fmt.Errorf("binance returned %d", resp.StatusCode)
	}

	var r commissionResponse

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return FeeRate{}, err
	}

	maker, err := strconv.ParseFloat(r.StandardCommission.Maker, 64)
	if err != nil {
		return FeeRate{}, err
	}

	taker, err := strconv.ParseFloat(r.StandardCommission.Taker, 64)
	if err != nil {
		return FeeRate{}, err
	}

	return FeeRate{
		Maker: maker,
		Taker: taker,
	}, nil
}
