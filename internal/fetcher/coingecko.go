package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"candlecore/internal/engine"
)

const (
	coingeckoBaseURL = "https://api.coingecko.com/api/v3"
	cgMaxRetries     = 3
	cgRetryDelay     = time.Second * 3
)

// CoinGeckoFetcher fetches live candle data from CoinGecko public API
type CoinGeckoFetcher struct {
	client  *http.Client
	baseURL string
}

// NewCoinGeckoFetcher creates a new CoinGecko data fetcher
func NewCoinGeckoFetcher() *CoinGeckoFetcher {
	return &CoinGeckoFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: coingeckoBaseURL,
	}
}

// coingeckoOHLC represents CoinGecko OHLC response format
// Returns: [timestamp, open, high, low, close]
type coingeckoOHLC []float64

// FetchCandles fetches historical OHLC data from CoinGecko
// coinID: "bitcoin", "ethereum"
// days: number of days of historical data (1, 7, 14, 30, 90, 180, 365, max)
func (f *CoinGeckoFetcher) FetchCandles(ctx context.Context, coinID string, days int) ([]engine.Candle, error) {
	params := url.Values{}
	params.Add("vs_currency", "usd")
	params.Add("days", strconv.Itoa(days))

	endpoint := fmt.Sprintf("%s/coins/%s/ohlc?%s", f.baseURL, coinID, params.Encode())

	var ohlcData []coingeckoOHLC
	var err error

	for attempt := 0; attempt < cgMaxRetries; attempt++ {
		ohlcData, err = f.fetchWithRetry(ctx, endpoint)
		if err == nil {
			break
		}

		if attempt < cgMaxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(cgRetryDelay):
				continue
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles after %d attempts: %w", cgMaxRetries, err)
	}

	candles := make([]engine.Candle, 0, len(ohlcData))
	for _, ohlc := range ohlcData {
		candle, err := f.parseOHLC(ohlc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OHLC: %w", err)
		}
		candles = append(candles, candle)
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no candle data returned from CoinGecko")
	}

	return candles, nil
}

// FetchLatestCandles fetches recent candles (last 1 day)
func (f *CoinGeckoFetcher) FetchLatestCandles(ctx context.Context, coinID string) ([]engine.Candle, error) {
	return f.FetchCandles(ctx, coinID, 1)
}

// FetchCandlesSince fetches candles since a specific timestamp
func (f *CoinGeckoFetcher) FetchCandlesSince(ctx context.Context, coinID string, since time.Time) ([]engine.Candle, error) {
	daysSince := int(time.Since(since).Hours() / 24)
	if daysSince < 1 {
		daysSince = 1
	}
	if daysSince > 365 {
		daysSince = 365
	}

	candles, err := f.FetchCandles(ctx, coinID, daysSince)
	if err != nil {
		return nil, err
	}

	filtered := make([]engine.Candle, 0)
	for _, candle := range candles {
		if candle.Timestamp.After(since) || candle.Timestamp.Equal(since) {
			filtered = append(filtered, candle)
		}
	}

	return filtered, nil
}

// fetchWithRetry performs HTTP request with error handling
func (f *CoinGeckoFetcher) fetchWithRetry(ctx context.Context, endpoint string) ([]coingeckoOHLC, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Candlecore/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limit exceeded, retry after some time")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var ohlcData []coingeckoOHLC
	if err := json.NewDecoder(resp.Body).Decode(&ohlcData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return ohlcData, nil
}

// parseOHLC converts CoinGecko OHLC format to engine.Candle
// Format: [timestamp_ms, open, high, low, close]
func (f *CoinGeckoFetcher) parseOHLC(ohlc coingeckoOHLC) (engine.Candle, error) {
	if len(ohlc) < 5 {
		return engine.Candle{}, fmt.Errorf("invalid OHLC format: expected 5 fields, got %d", len(ohlc))
	}

	timestamp := time.UnixMilli(int64(ohlc[0]))
	open := ohlc[1]
	high := ohlc[2]
	low := ohlc[3]
	close := ohlc[4]

	if high < low {
		return engine.Candle{}, fmt.Errorf("invalid candle: high (%.2f) < low (%.2f)", high, low)
	}
	if open < low || open > high {
		return engine.Candle{}, fmt.Errorf("invalid candle: open (%.2f) outside [low, high] range", open)
	}
	if close < low || close > high {
		return engine.Candle{}, fmt.Errorf("invalid candle: close (%.2f) outside [low, high] range", close)
	}

	return engine.Candle{
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    0,
	}, nil
}

// ValidateCoinID checks if a coin ID is supported
func ValidateCoinID(coinID string) bool {
	supportedCoins := map[string]bool{
		"bitcoin":  true,
		"ethereum": true,
	}
	return supportedCoins[coinID]
}

// CoinIDFromSymbol converts trading symbol to CoinGecko coin ID
func CoinIDFromSymbol(symbol string) string {
	symbolToCoinID := map[string]string{
		"BTCUSDT": "bitcoin",
		"ETHUSDT": "ethereum",
		"BTC/USD": "bitcoin",
		"ETH/USD": "ethereum",
	}
	
	if coinID, exists := symbolToCoinID[symbol]; exists {
		return coinID
	}
	
	return ""
}
