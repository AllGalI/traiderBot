package binance

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

	"github.com/shopspring/decimal"

	simulationApplication "traiderBot/internalDDD/simulation/application"
)

const accountCommissionPath = "/api/v3/account/commission"

type FeeProvider struct {
	client  *http.Client
	baseURL string
	apiKey  string
	secret  string
}

func NewFeeProvider() *FeeProvider {
	return &FeeProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: restBaseURL,
		apiKey:  os.Getenv("API_KEY"),
		secret:  os.Getenv("SECRET_KEY"),
	}
}

type commissionResponse struct {
	Symbol string `json:"symbol"`

	StandardCommission struct {
		Maker string `json:"maker"`
		Taker string `json:"taker"`
	} `json:"standardCommission"`
}

func (p *FeeProvider) Get(
	ctx context.Context,
	symbol string,
) (simulationApplication.FeeConfig, error) {
	if p.apiKey == "" {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("binance api key is not configured")
	}

	if p.secret == "" {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("binance secret key is not configured")
	}

	params := url.Values{}

	params.Set("symbol", symbol)
	params.Set(
		"timestamp",
		strconv.FormatInt(time.Now().UnixMilli(), 10),
	)

	query := params.Encode()
	signature := p.sign(query)

	reqURL := fmt.Sprintf(
		"%s%s?%s&signature=%s",
		p.baseURL,
		accountCommissionPath,
		query,
		signature,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL,
		nil,
	)
	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("create commission request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("request commission: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return simulationApplication.FeeConfig{},
			fmt.Errorf(
				"binance commission request returned status %d",
				resp.StatusCode,
			)
	}

	var dto commissionResponse

	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("decode commission response: %w", err)
	}

	taker, err := decimal.NewFromString(
		dto.StandardCommission.Taker,
	)
	if err != nil {
		return simulationApplication.FeeConfig{},
			fmt.Errorf("parse taker commission: %w", err)
	}

	return simulationApplication.FeeConfig{
		Rate: taker,
	}, nil
}

func (p *FeeProvider) sign(query string) string {
	mac := hmac.New(
		sha256.New,
		[]byte(p.secret),
	)

	_, _ = mac.Write([]byte(query))

	return hex.EncodeToString(mac.Sum(nil))
}