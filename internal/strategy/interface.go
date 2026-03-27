package strategy

import (
	"time"
)

// Candle represents OHLCV data
type Candle struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// DataFrame holds candles and calculated indicators
type DataFrame struct {
	Candles    []Candle
	Indicators map[string][]float64 // e.g., "sma_20", "rsi_14"
}

// Signal represents entry or exit decision
type Signal struct {
	Action     string  // "buy", "sell", "hold"
	Confidence int     // 0-100%
	Reason     string  // Human-readable explanation
	Price      float64 // Suggested entry/exit price
}

// Position represents current open trade
type Position struct {
	Side       string    // "long" or "short"
	EntryPrice float64   // Price when position opened
	EntryTime  time.Time // When position opened
	Size       float64   // Position size
	StopLoss   float64   // Current stoploss price
	TakeProfit float64   // Current take profit price
}

// StrategyConfig holds strategy parameters
type StrategyConfig struct {
	// Risk Management
	Stoploss      float64            // Fixed stoploss percentage (e.g., -0.05 = -5%)
	TrailingStop  bool               // Enable trailing stoploss
	TrailingDelta float64            // Trailing stop distance (e.g., 0.02 = 2%)
	MinimalROI    map[int]float64    // Time-based minimal ROI {minutes: roi_percentage}
	
	// Position Sizing
	StakeAmount   float64 // Amount to stake per trade
	MaxOpenTrades int     // Maximum concurrent open trades
	
	// Timeframe
	Timeframe string // e.g., "5m", "1h", "1d"
	
	// Strategy-specific parameters
	CustomParams map[string]interface{}
}

// IStrategy defines the interface all strategies must implement
type IStrategy interface {
	// GetName returns the strategy name
	GetName() string
	
	// GetVersion returns the strategy version
	GetVersion() string
	
	// GetConfig returns the strategy configuration
	GetConfig() StrategyConfig
	
	// PopulateIndicators calculates technical indicators
	// Called once per candle update
	PopulateIndicators(df *DataFrame) error
	
	// PopulateEntrySignal determines if we should enter a trade
	// Called when no position is open
	PopulateEntrySignal(df *DataFrame, current Candle) Signal
	
	// PopulateExitSignal determines if we should exit a trade
	// Called when a position is open
	PopulateExitSignal(df *DataFrame, current Candle, position Position) Signal
	
	// CustomStoploss allows dynamic stoploss calculation
	// Optional: return nil to use fixed stoploss from config
	CustomStoploss(current Candle, position Position, currentProfit float64) *float64
}

// BaseStrategy provides default implementations
type BaseStrategy struct {
	Name    string
	Version string
	Config  StrategyConfig
}

func (s *BaseStrategy) GetName() string {
	return s.Name
}

func (s *BaseStrategy) GetVersion() string {
	return s.Version
}

func (s *BaseStrategy) GetConfig() StrategyConfig {
	return s.Config
}

// CustomStoploss default implementation returns nil (use fixed stoploss)
func (s *BaseStrategy) CustomStoploss(current Candle, position Position, currentProfit float64) *float64 {
	return nil
}
