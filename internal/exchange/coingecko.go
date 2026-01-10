package exchange

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// CoinGeckoProvider fetches real market data from CoinGecko API
type CoinGeckoProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewCoinGeckoProvider creates a new CoinGecko data provider
func NewCoinGeckoProvider() *CoinGeckoProvider {
	apiKey := os.Getenv("COINGECKO_API_KEY")
	baseURL := os.Getenv("COINGECKO_API_URL")
	if baseURL == "" {
		baseURL = "https://api.coingecko.com/api/v3"
	}

	return &CoinGeckoProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CoinGeckoOHLC represents OHLC data from CoinGecko
type CoinGeckoOHLC struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
}

// GetCandles fetches candles from CoinGecko
func (p *CoinGeckoProvider) GetCandles(symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	// Map symbols to CoinGecko IDs
	coinID := mapSymbolToCoinGeckoID(symbol)
	
	// Map timeframe to days
	days := mapTimeframeToDays(timeframe)
	
	url := fmt.Sprintf("%s/coins/%s/ohlc?vs_currency=usd&days=%d", p.baseURL, coinID, days)
	
	// Add API key if present
	if p.apiKey != "" {
		url += "&x_cg_demo_api_key=" + p.apiKey
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ohlcData [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&ohlcData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Candle format
	candles := make([]Candle, 0, len(ohlcData))
	for _, data := range ohlcData {
		if len(data) != 5 {
			continue
		}

		timestamp := time.Unix(int64(data[0])/1000, 0)
		candles = append(candles, Candle{
			Timestamp: timestamp,
			Open:      data[1],
			High:      data[2],
			Low:       data[3],
			Close:     data[4],
			Volume:    0, // CoinGecko OHLC doesn't include volume
		})
	}

	// Apply limit
	if limit > 0 && limit < len(candles) {
		candles = candles[len(candles)-limit:]
	}

	return candles, nil
}

// StreamCandles streams candles (for live mode, periodically fetch)
func (p *CoinGeckoProvider) StreamCandles(symbol string, timeframe Timeframe) (<-chan Candle, error) {
	ch := make(chan Candle, 100)
	
	go func() {
		defer close(ch)
		
		ticker := time.NewTicker(timeframe.ToDuration())
		defer ticker.Stop()

		for range ticker.C {
			candles, err := p.GetCandles(symbol, timeframe, 1)
			if err != nil {
				continue
			}
			if len(candles) > 0 {
				ch <- candles[0]
			}
		}
	}()

	return ch, nil
}

// GetSupportedTimeframes returns available timeframes
func (p *CoinGeckoProvider) GetSupportedTimeframes() []Timeframe {
	return []Timeframe{
		Timeframe1h,
		Timeframe4h,
		Timeframe1d,
	}
}

// GetSupportedSymbols returns popular symbols
func (p *CoinGeckoProvider) GetSupportedSymbols() []string {
	return []string{
		"bitcoin",
		"ethereum",
		"binancecoin",
		"cardano",
		"solana",
		"polkadot",
		"dogecoin",
		"avalanche",
		"polygon",
		"chainlink",
	}
}

// mapSymbolToCoinGeckoID maps symbol names to CoinGecko IDs
func mapSymbolToCoinGeckoID(symbol string) string {
	mapping := map[string]string{
		"bitcoin":     "bitcoin",
		"btc":         "bitcoin",
		"ethereum":    "ethereum",
		"eth":         "ethereum",
		"binancecoin": "binancecoin",
		"bnb":         "binancecoin",
		"cardano":     "cardano",
		"ada":         "cardano",
		"solana":      "solana",
		"sol":         "solana",
		"polkadot":    "polkadot",
		"dot":         "polkadot",
		"dogecoin":    "dogecoin",
		"doge":        "dogecoin",
	}

	if id, ok := mapping[symbol]; ok {
		return id
	}
	return symbol
}

// mapTimeframeToDays maps timeframe to CoinGecko days parameter
func mapTimeframeToDays(timeframe Timeframe) int {
	switch timeframe {
	case Timeframe1h:
		return 7 // 1 week for hourly
	case Timeframe4h:
		return 30 // 1 month for 4h
	case Timeframe1d:
		return 90 // 3 months for daily
	default:
		return 7
	}
}
