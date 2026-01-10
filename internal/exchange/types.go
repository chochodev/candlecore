package exchange

import (
	"time"
)

// Timeframe represents a candle interval
type Timeframe string

const (
	Timeframe1m  Timeframe = "1m"
	Timeframe5m  Timeframe = "5m"
	Timeframe15m Timeframe = "15m"
	Timeframe1h  Timeframe = "1h"
	Timeframe4h  Timeframe = "4h"
	Timeframe1d  Timeframe = "1d"
)

// Candle represents a single OHLCV candlestick
type Candle struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// DataProvider defines the interface for candle data sources
type DataProvider interface {
	// GetCandles retrieves candles for a symbol and timeframe
	GetCandles(symbol string, timeframe Timeframe, limit int) ([]Candle, error)
	
	// StreamCandles streams candles in real-time or replay mode
	StreamCandles(symbol string, timeframe Timeframe) (<-chan Candle, error)
	
	// GetSupportedTimeframes returns available timeframes
	GetSupportedTimeframes() []Timeframe
	
	// GetSupportedSymbols returns available trading pairs
	GetSupportedSymbols() []string
}

// ToMinutes converts timeframe to minutes
func (t Timeframe) ToMinutes() int {
	switch t {
	case Timeframe1m:
		return 1
	case Timeframe5m:
		return 5
	case Timeframe15m:
		return 15
	case Timeframe1h:
		return 60
	case Timeframe4h:
		return 240
	case Timeframe1d:
		return 1440
	default:
		return 0
	}
}

// ToDuration converts timeframe to time.Duration
func (t Timeframe) ToDuration() time.Duration {
	return time.Duration(t.ToMinutes()) * time.Minute
}

// IsValid checks if timeframe is supported
func (t Timeframe) IsValid() bool {
	switch t {
	case Timeframe1m, Timeframe5m, Timeframe15m, Timeframe1h, Timeframe4h, Timeframe1d:
		return true
	default:
		return false
	}
}
