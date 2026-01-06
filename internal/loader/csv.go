package loader

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"candlecore/internal/engine"
)

// CSVLoader loads candle data from CSV files
type CSVLoader struct {
	filePath string
}

// NewCSVLoader creates a new CSV loader
func NewCSVLoader(filePath string) *CSVLoader {
	return &CSVLoader{
		filePath: filePath,
	}
}

// Load reads candle data from a CSV file
// Expected CSV format: timestamp,open,high,low,close,volume
// timestamp should be in RFC3339 format (e.g., 2024-01-01T00:00:00Z)
func (l *CSVLoader) Load() ([]engine.Candle, error) {
	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
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
		return nil, fmt.Errorf("invalid CSV header: expected %v, got %v", expectedHeader, header)
	}

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records: %w", err)
	}

	// Parse candles
	candles := make([]engine.Candle, 0, len(records))
	for i, record := range records {
		if len(record) != 6 {
			return nil, fmt.Errorf("invalid record at line %d: expected 6 fields, got %d", i+2, len(record))
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp at line %d: %w", i+2, err)
		}

		// Parse OHLCV values
		open, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid open price at line %d: %w", i+2, err)
		}

		high, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid high price at line %d: %w", i+2, err)
		}

		low, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid low price at line %d: %w", i+2, err)
		}

		close, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid close price at line %d: %w", i+2, err)
		}

		volume, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid volume at line %d: %w", i+2, err)
		}

		// Validate candle data
		if high < low {
			return nil, fmt.Errorf("invalid candle at line %d: high (%.2f) < low (%.2f)", i+2, high, low)
		}
		if open < low || open > high {
			return nil, fmt.Errorf("invalid candle at line %d: open (%.2f) outside [low, high] range", i+2, open)
		}
		if close < low || close > high {
			return nil, fmt.Errorf("invalid candle at line %d: close (%.2f) outside [low, high] range", i+2, close)
		}

		candles = append(candles, engine.Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	return candles, nil
}
