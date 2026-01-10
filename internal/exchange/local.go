package exchange

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// LocalFileProvider reads candle data from local CSV files
type LocalFileProvider struct {
	dataDir string
	mu      sync.RWMutex
	cache   map[string][]Candle // symbol_timeframe -> candles
}

// NewLocalFileProvider creates a provider that reads from local files
func NewLocalFileProvider(dataDir string) *LocalFileProvider {
	return &LocalFileProvider{
		dataDir: dataDir,
		cache:   make(map[string][]Candle),
	}
}

// GetCandles retrieves candles from CSV file
func (p *LocalFileProvider) GetCandles(symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	if !timeframe.IsValid() {
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}

	cacheKey := fmt.Sprintf("%s_%s", symbol, timeframe)
	
	// Check cache first
	p.mu.RLock()
	if candles, ok := p.cache[cacheKey]; ok {
		p.mu.RUnlock()
		return p.limitCandles(candles, limit), nil
	}
	p.mu.RUnlock()

	// Load from file
	candles, err := p.loadFromFile(symbol, timeframe)
	if err != nil {
		return nil, err
	}

	// Cache the result
	p.mu.Lock()
	p.cache[cacheKey] = candles
	p.mu.Unlock()

	return p.limitCandles(candles, limit), nil
}

// StreamCandles streams candles one by one (for replay/backtesting)
func (p *LocalFileProvider) StreamCandles(symbol string, timeframe Timeframe) (<-chan Candle, error) {
	candles, err := p.GetCandles(symbol, timeframe, 0)
	if err != nil {
		return nil, err
	}

	ch := make(chan Candle, 100)
	
	go func() {
		defer close(ch)
		for _, candle := range candles {
			ch <- candle
		}
	}()

	return ch, nil
}

// GetSupportedTimeframes returns available timeframes
func (p *LocalFileProvider) GetSupportedTimeframes() []Timeframe {
	return []Timeframe{
		Timeframe1m,
		Timeframe5m,
		Timeframe15m,
		Timeframe1h,
		Timeframe4h,
		Timeframe1d,
	}
}

// GetSupportedSymbols returns symbols by scanning data directory
func (p *LocalFileProvider) GetSupportedSymbols() []string {
	symbols := make(map[string]bool)
	
	files, err := filepath.Glob(filepath.Join(p.dataDir, "*_*.csv"))
	if err != nil {
		return []string{}
	}

	for _, file := range files {
		base := filepath.Base(file)
		// Extract symbol from filename (e.g., bitcoin_1h.csv -> bitcoin)
		if len(base) > 0 {
			// Remove .csv extension
			name := base[:len(base)-4]
			// Split by last underscore
			for i := len(name) - 1; i >= 0; i-- {
				if name[i] == '_' {
					symbol := name[:i]
					symbols[symbol] = true
					break
				}
			}
		}
	}

	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}
	return result
}

// loadFromFile reads candles from CSV file
func (p *LocalFileProvider) loadFromFile(symbol string, timeframe Timeframe) ([]Candle, error) {
	filename := fmt.Sprintf("%s_%s.csv", symbol, timeframe)
	filePath := filepath.Join(p.dataDir, filename)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header
	expectedHeader := []string{"timestamp", "open", "high", "low", "close", "volume"}
	if len(header) != len(expectedHeader) {
		return nil, fmt.Errorf("invalid CSV header in %s", filename)
	}

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records: %w", err)
	}

	// Parse candles
	candles := make([]Candle, 0, len(records))
	for i, record := range records {
		if len(record) != 6 {
			continue // Skip malformed records
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp at line %d: %w", i+2, err)
		}

		// Parse OHLCV
		open, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}

		high, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}

		low, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			continue
		}

		close, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			continue
		}

		volume, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			continue
		}

		candles = append(candles, Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	if len(candles) == 0 {
		return nil, fmt.Errorf("no valid candles found in %s", filename)
	}

	return candles, nil
}

// limitCandles returns the last N candles (most recent)
func (p *LocalFileProvider) limitCandles(candles []Candle, limit int) []Candle {
	if limit <= 0 || limit >= len(candles) {
		return candles
	}
	return candles[len(candles)-limit:]
}

// ClearCache clears the internal candle cache
func (p *LocalFileProvider) ClearCache() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = make(map[string][]Candle)
}
