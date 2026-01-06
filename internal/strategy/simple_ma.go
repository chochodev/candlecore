package strategy

import (
	"candlecore/internal/engine"
)

// SimpleMAStrategy implements a simple moving average crossover strategy
// This is an example implementation showing how to use the Strategy interface
type SimpleMAStrategy struct {
	fastPeriod   int
	slowPeriod   int
	positionSize float64

	// Internal state for MA calculation
	fastMA []float64
	slowMA []float64
	prices []float64
}

// NewSimpleMAStrategy creates a new moving average crossover strategy
func NewSimpleMAStrategy(fastPeriod, slowPeriod int, positionSize float64) *SimpleMAStrategy {
	return &SimpleMAStrategy{
		fastPeriod:   fastPeriod,
		slowPeriod:   slowPeriod,
		positionSize: positionSize,
		prices:       make([]float64, 0, slowPeriod),
	}
}

// Name returns the strategy name
func (s *SimpleMAStrategy) Name() string {
	return "SimpleMAStrategy"
}

// OnCandle processes a new candle and returns a trading signal
func (s *SimpleMAStrategy) OnCandle(candle engine.Candle, account *engine.Account) engine.Signal {
	// Add current price to history
	s.prices = append(s.prices, candle.Close)

	// Keep only the required number of prices
	if len(s.prices) > s.slowPeriod {
		s.prices = s.prices[1:]
	}

	// Need enough data for slow MA
	if len(s.prices) < s.slowPeriod {
		return engine.Signal{
			Action: engine.SignalActionHold,
			Reason: "insufficient data for moving averages",
		}
	}

	// Calculate moving averages
	fastMA := s.calculateMA(s.fastPeriod)
	slowMA := s.calculateMA(s.slowPeriod)

	// Get previous MAs for crossover detection
	prevFastMA := 0.0
	prevSlowMA := 0.0
	if len(s.fastMA) > 0 {
		prevFastMA = s.fastMA[len(s.fastMA)-1]
		prevSlowMA = s.slowMA[len(s.slowMA)-1]
	}

	// Store current MAs
	s.fastMA = append(s.fastMA, fastMA)
	s.slowMA = append(s.slowMA, slowMA)

	// Keep MA history limited
	if len(s.fastMA) > 100 {
		s.fastMA = s.fastMA[1:]
		s.slowMA = s.slowMA[1:]
	}

	// Check for position
	hasPosition := false
	for _, pos := range account.Positions {
		if pos.Symbol == "BTC/USD" && pos.Quantity > 0 {
			hasPosition = true
			break
		}
	}

	// Crossover logic
	// Buy signal: fast MA crosses above slow MA
	if !hasPosition && prevFastMA <= prevSlowMA && fastMA > slowMA {
		quantity := s.positionSize / candle.Close // Calculate quantity based on position size
		return engine.Signal{
			Action:   engine.SignalActionBuy,
			Symbol:   "BTC/USD",
			Quantity: quantity,
			Reason:   "fast MA crossed above slow MA (golden cross)",
		}
	}

	// Sell signal: fast MA crosses below slow MA
	if hasPosition && prevFastMA >= prevSlowMA && fastMA < slowMA {
		// Sell entire position
		var quantity float64
		for _, pos := range account.Positions {
			if pos.Symbol == "BTC/USD" {
				quantity = pos.Quantity
				break
			}
		}

		return engine.Signal{
			Action:   engine.SignalActionSell,
			Symbol:   "BTC/USD",
			Quantity: quantity,
			Reason:   "fast MA crossed below slow MA (death cross)",
		}
	}

	// No signal
	return engine.Signal{
		Action: engine.SignalActionHold,
		Reason: "no crossover detected",
	}
}

// OnTrade is called after a trade is executed
func (s *SimpleMAStrategy) OnTrade(trade *engine.Trade) {
	// This can be used to track strategy performance
	// For now, it's a no-op
}

// calculateMA calculates simple moving average for the given period
func (s *SimpleMAStrategy) calculateMA(period int) float64 {
	if len(s.prices) < period {
		return 0
	}

	sum := 0.0
	startIdx := len(s.prices) - period

	for i := startIdx; i < len(s.prices); i++ {
		sum += s.prices[i]
	}

	return sum / float64(period)
}
