package strategies

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/strategy"
)

// StrategyAdapter adapts new strategy.IStrategy to old bot.Strategy interface
type StrategyAdapter struct {
	strategy strategy.IStrategy
}

// NewStrategyAdapter wraps a new strategy for use with old bot
func NewStrategyAdapter(s strategy.IStrategy) bot.Strategy {
	return &StrategyAdapter{
		strategy: s,
	}
}

// Name returns the strategy name
func (a *StrategyAdapter) Name() string {
	return a.strategy.GetName()
}

// Analyze analyzes candles and produces a decision
func (a *StrategyAdapter) Analyze(candles []exchange.Candle, currentPos *bot.Position) (*bot.Decision, error) {
	// Convert exchange.Candle to strategy.Candle
	stratCandles := make([]strategy.Candle, len(candles))
	for i, c := range candles {
		stratCandles[i] = strategy.Candle{
			Timestamp: c.Timestamp,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
		}
	}

	// Create DataFrame
	df := &strategy.DataFrame{
		Candles:    stratCandles,
		Indicators: make(map[string][]float64),
	}

	// Populate indicators
	if err := a.strategy.PopulateIndicators(df); err != nil {
		return nil, err
	}

	// Get current candle
	current := stratCandles[len(stratCandles)-1]

	// Determine entry or exit signal
	var signal strategy.Signal
	if currentPos != nil {
		// Convert bot.Position to strategy.Position
		stratPos := strategy.Position{
			Side:       currentPos.Side,
			EntryPrice: currentPos.EntryPrice,
			EntryTime:  currentPos.OpenedAt,
			Size:       currentPos.Quantity,
			TrailingSL: currentPos.TrailingSL, // 🚀 Sync engine state back to strategy
		}
		signal = a.strategy.PopulateExitSignal(df, current, stratPos)
	} else {
		signal = a.strategy.PopulateEntrySignal(df, current)
	}
	
	// Convert strategy.Signal to bot.Decision
	decision := &bot.Decision{
		Timestamp:  current.Timestamp,
		Signal:     bot.Signal(signal.Action),
		Symbol:     "", // Will be filled by bot
		Price:      signal.Price,
		Quantity:   signal.Quantity,
		Confidence: float64(signal.Confidence),
		Reasoning:  signal.Reason,
		Indicators: make(map[string]float64),
		TrailingSL: signal.TrailingSL, // 🚀 Sync Profit Lock
	}

	// Copy indicators to decision
	for name, values := range df.Indicators {
		if len(values) > 0 {
			decision.Indicators[name] = values[len(values)-1]
		}
	}

	return decision, nil
}

// Configure updates strategy parameters (placeholder for now)
func (a *StrategyAdapter) Configure(params map[string]interface{}) error {
	// TODO: Allow runtime configuration
	return nil
}
