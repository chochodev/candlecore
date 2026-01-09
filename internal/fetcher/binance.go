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
	binanceBaseURL = "https://api.binance.com"
	maxRetries     = 3
	retryDelay     = time.Second * 2
)

// BinanceFetcher fetches live candle data from Binance public API
type BinanceFetcher struct {
	client  *http.Client
	baseURL string
}

// NewBinanceFetcher creates a new Binance data fetcher
func NewBinanceFetcher() *BinanceFetcher {
	return &BinanceFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: binanceBaseURL,
	}
}

// binanceKline represents Binance API kline response
type binanceKline []interface{}

// FetchCandles fetches historical candles from Binance
// symbol: e.g., "BTCUSDT", "ETHUSDT"
// interval: "1m", "5m", "15m", "1h", "4h", "1d"
// limit: number of candles to fetch (max 1000)
func (f *BinanceFetcher) FetchCandles(ctx context.Context, symbol, interval string, limit int) ([]engine.Candle, error) {
	if limit > 1000 {
		limit = 1000
	}

	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("interval", interval)
	params.Add("limit", strconv.Itoa(limit))

	endpoint := fmt.Sprintf("%s/api/v3/klines?%s", f.baseURL, params.Encode())

	var klines []binanceKline
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		klines, err = f.fetchWithRetry(ctx, endpoint)
		if err == nil {
			break
		}

		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles after %d attempts: %w", maxRetries, err)
	}

	candles := make([]engine.Candle, 0, len(klines))
	for _, k := range klines {
		candle, err := f.parseKline(k)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kline: %w", err)
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// FetchLatestCandle fetches the most recent completed candle
func (f *BinanceFetcher) FetchLatestCandle(ctx context.Context, symbol, interval string) (*engine.Candle, error) {
	candles, err := f.FetchCandles(ctx, symbol, interval, 2)
	if err != nil {
		return nil, err
	}

	if len(candles) < 2 {
		return nil, fmt.Errorf("insufficient candles returned")
	}

	return &candles[len(candles)-2], nil
}

// FetchCandlesSince fetches candles since a specific timestamp
func (f *BinanceFetcher) FetchCandlesSince(ctx context.Context, symbol, interval string, since time.Time) ([]engine.Candle, error) {
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("interval", interval)
	params.Add("startTime", strconv.FormatInt(since.UnixMilli(), 10))
	params.Add("limit", "1000")

	endpoint := fmt.Sprintf("%s/api/v3/klines?%s", f.baseURL, params.Encode())

	var klines []binanceKline
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		klines, err = f.fetchWithRetry(ctx, endpoint)
		if err == nil {
			break
		}

		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles: %w", err)
	}

	candles := make([]engine.Candle, 0, len(klines))
	for _, k := range klines {
		candle, err := f.parseKline(k)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kline: %w", err)
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// StreamCandles creates a channel that continuously fetches new candles
func (f *BinanceFetcher) StreamCandles(ctx context.Context, symbol, interval string, pollInterval time.Duration) (<-chan engine.Candle, <-chan error) {
	candleChan := make(chan engine.Candle, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(candleChan)
		defer close(errChan)

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		var lastTimestamp time.Time

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				candle, err := f.FetchLatestCandle(ctx, symbol, interval)
				if err != nil {
					errChan <- err
					continue
				}

				if candle.Timestamp.After(lastTimestamp) {
					lastTimestamp = candle.Timestamp
					candleChan <- *candle
				}
			}
		}
	}()

	return candleChan, errChan
}

// fetchWithRetry performs HTTP request with error handling
func (f *BinanceFetcher) fetchWithRetry(ctx context.Context, endpoint string) ([]binanceKline, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Candlecore/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var klines []binanceKline
	if err := json.NewDecoder(resp.Body).Decode(&klines); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return klines, nil
}

// parseKline converts Binance kline format to engine.Candle
func (f *BinanceFetcher) parseKline(k binanceKline) (engine.Candle, error) {
	if len(k) < 11 {
		return engine.Candle{}, fmt.Errorf("invalid kline format: expected 11+ fields, got %d", len(k))
	}

	openTime, ok := k[0].(float64)
	if !ok {
		return engine.Candle{}, fmt.Errorf("invalid open time format")
	}

	open, err := parseFloat(k[1])
	if err != nil {
		return engine.Candle{}, fmt.Errorf("invalid open price: %w", err)
	}

	high, err := parseFloat(k[2])
	if err != nil {
		return engine.Candle{}, fmt.Errorf("invalid high price: %w", err)
	}

	low, err := parseFloat(k[3])
	if err != nil {
		return engine.Candle{}, fmt.Errorf("invalid low price: %w", err)
	}

	close, err := parseFloat(k[4])
	if err != nil {
		return engine.Candle{}, fmt.Errorf("invalid close price: %w", err)
	}

	volume, err := parseFloat(k[5])
	if err != nil {
		return engine.Candle{}, fmt.Errorf("invalid volume: %w", err)
	}

	return engine.Candle{
		Timestamp: time.UnixMilli(int64(openTime)),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}, nil
}

// parseFloat safely converts interface{} to float64
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// ValidateSymbol checks if a symbol is supported
func ValidateSymbol(symbol string) bool {
	supportedSymbols := map[string]bool{
		"BTCUSDT": true,
		"ETHUSDT": true,
	}
	return supportedSymbols[symbol]
}

// ValidateInterval checks if an interval is supported
func ValidateInterval(interval string) bool {
	supportedIntervals := map[string]bool{
		"1m":  true,
		"5m":  true,
		"15m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	return supportedIntervals[interval]
}
