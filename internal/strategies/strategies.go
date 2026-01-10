package strategies

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/indicators"
	"fmt"
)

// SimpleMAStrategy is a moving average crossover strategy
type SimpleMAStrategy struct {
	fastPeriod int
	slowPeriod int
}

// NewSimpleMAStrategy creates a new MA crossover strategy
func NewSimpleMAStrategy(fastPeriod, slowPeriod int) *SimpleMAStrategy {
	return &SimpleMAStrategy{
		fastPeriod: fastPeriod,
		slowPeriod: slowPeriod,
	}
}

// Name returns the strategy name
func (s *SimpleMAStrategy) Name() string {
	return fmt.Sprintf("MA Crossover (%d/%d)", s.fastPeriod, s.slowPeriod)
}

// Analyze analyzes candles using MA crossover
func (s *SimpleMAStrategy) Analyze(candles []exchange.Candle) (*bot.Decision, error) {
	if len(candles) < s.slowPeriod {
		return &bot.Decision{
			Signal: bot.SignalHold,
			Reasoning: "Insufficient data for analysis",
		}, nil
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate MAs
	fastMA, err := indicators.SMA(closes, s.fastPeriod)
	if err != nil {
		return nil, err
	}

	slowMA, err := indicators.SMA(closes, s.slowPeriod)
	if err != nil {
		return nil, err
	}

	// Get latest values
	lastFast := fastMA[len(fastMA)-1]
	lastSlow := slowMA[len(slowMA)-1]
	prevFast := fastMA[len(fastMA)-2]
	prevSlow := slowMA[len(slowMA)-2]

	lastCandle := candles[len(candles)-1]
	decision := &bot.Decision{
		Timestamp: lastCandle.Timestamp,
		Symbol:    "BTCUSDT", // TODO: get from context
		Price:     lastCandle.Close,
		Indicators: map[string]float64{
			"fast_ma": lastFast,
			"slow_ma": lastSlow,
		},
	}

	// Detect crossover
	if prevFast <= prevSlow && lastFast > lastSlow {
		// Bullish crossover
		decision.Signal = bot.SignalBuy
		decision.Confidence = 75
		decision.Reasoning = fmt.Sprintf("MA crossover: Fast MA (%.2f) crossed above Slow MA (%.2f)", lastFast, lastSlow)
	} else if prevFast >= prevSlow && lastFast < lastSlow {
		// Bearish crossover
		decision.Signal = bot.SignalSell
		decision.Confidence = 75
		decision.Reasoning = fmt.Sprintf("MA crossover: Fast MA (%.2f) crossed below Slow MA (%.2f)", lastFast, lastSlow)
	} else {
		decision.Signal = bot.SignalHold
		decision.Confidence = 50
		decision.Reasoning = fmt.Sprintf("No crossover detected. Fast MA: %.2f, Slow MA: %.2f", lastFast, lastSlow)
	}

	return decision, nil
}

// Configure updates strategy parameters
func (s *SimpleMAStrategy) Configure(params map[string]interface{}) error {
	if fast, ok := params["fast_period"].(int); ok {
		s.fastPeriod = fast
	}
	if slow, ok := params["slow_period"].(int); ok {
		s.slowPeriod = slow
	}
	return nil
}

// RSIStrategy is an RSI-based strategy
type RSIStrategy struct {
	period    int
	oversold  float64
	overbought float64
}

// NewRSIStrategy creates a new RSI strategy
func NewRSIStrategy(period int, oversold, overbought float64) *RSIStrategy {
	return &RSIStrategy{
		period:     period,
		oversold:   oversold,
		overbought: overbought,
	}
}

// Name returns the strategy name
func (s *RSIStrategy) Name() string {
	return fmt.Sprintf("RSI (%d)", s.period)
}

// Analyze analyzes candles using RSI
func (s *RSIStrategy) Analyze(candles []exchange.Candle) (*bot.Decision, error) {
	if len(candles) < s.period+1 {
		return &bot.Decision{
			Signal: bot.SignalHold,
			Reasoning: "Insufficient data",
		}, nil
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate RSI
	rsi, err := indicators.RSI(closes, s.period)
	if err != nil {
		return nil, err
	}

	lastRSI := rsi[len(rsi)-1]
	lastCandle := candles[len(candles)-1]

	decision := &bot.Decision{
		Timestamp: lastCandle.Timestamp,
		Symbol:    "BTCUSDT",
		Price:     lastCandle.Close,
		Indicators: map[string]float64{
			"rsi": lastRSI,
		},
	}

	if lastRSI < s.oversold {
		decision.Signal = bot.SignalBuy
		decision.Confidence = 80
		decision.Reasoning = fmt.Sprintf("RSI oversold: %.2f < %.2f", lastRSI, s.oversold)
	} else if lastRSI > s.overbought {
		decision.Signal = bot.SignalSell
		decision.Confidence = 80
		decision.Reasoning = fmt.Sprintf("RSI overbought: %.2f > %.2f", lastRSI, s.overbought)
	} else {
		decision.Signal = bot.SignalHold
		decision.Confidence = 50
		decision.Reasoning = fmt.Sprintf("RSI neutral: %.2f", lastRSI)
	}

	return decision, nil
}

// Configure updates strategy parameters
func (s *RSIStrategy) Configure(params map[string]interface{}) error {
	if period, ok := params["period"].(int); ok {
		s.period = period
	}
	if oversold, ok := params["oversold"].(float64); ok {
		s.oversold = oversold
	}
	if overbought, ok := params["overbought"].(float64); ok {
		s.overbought = overbought
	}
	return nil
}
