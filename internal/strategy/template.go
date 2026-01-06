package strategy

import (
	"candlecore/internal/engine"
)

// TemplateStrategy is a template for creating new trading strategies
// Copy this file and implement your own logic in OnCandle
type TemplateStrategy struct {
	// Configuration parameters
	// Add your strategy-specific parameters here

	// Internal state
	// Add any state your strategy needs to maintain
	// For example: price history, indicators, signals, etc.
}

// NewTemplateStrategy creates a new instance of your strategy
// Add any configuration parameters you need
func NewTemplateStrategy( /* your params */ ) *TemplateStrategy {
	return &TemplateStrategy{
		// Initialize your parameters and state
	}
}

// Name returns the strategy name for logging
func (s *TemplateStrategy) Name() string {
	return "TemplateStrategy" // Change this to your strategy name
}

// OnCandle is called for each new candle
// This is where your trading logic goes
//
// Parameters:
//   - candle: The current candle data (OHLCV + timestamp)
//   - account: Current account state (balance, positions, open orders)
//
// Returns:
//   - Signal: A trading signal (buy/sell/hold)
func (s *TemplateStrategy) OnCandle(candle engine.Candle, account *engine.Account) engine.Signal {
	// Step 1: Update your internal state
	// Example: Add current price to a rolling window
	// s.prices = append(s.prices, candle.Close)

	// Step 2: Check if you have enough data
	// Example: Need at least 30 candles for your indicator
	// if len(s.prices) < 30 {
	//     return engine.Signal{
	//         Action: engine.SignalActionHold,
	//         Reason: "insufficient data",
	//     }
	// }

	// Step 3: Calculate your indicators
	// Example: Calculate SMA, RSI, Bollinger Bands, etc.
	// indicator := s.calculateYourIndicator()

	// Step 4: Check current position status
	// hasPosition := false
	// var currentPosition *engine.Position
	// for _, pos := range account.Positions {
	// 	if pos.Symbol == "BTC/USD" && pos.Quantity > 0 {
	// 		hasPosition = true
	// 		currentPosition = pos
	// 		break
	// 	}
	// }

	// Step 5: Generate trading signals based on your logic

	// BUY SIGNAL EXAMPLE:
	// if !hasPosition && yourBuyCondition {
	//     return engine.Signal{
	//         Action:   engine.SignalActionBuy,
	//         Symbol:   "BTC/USD",
	//         Quantity: 0.1, // or calculate based on account.Balance
	//         Reason:   "your buy reason for logging",
	//     }
	// }

	// SELL SIGNAL EXAMPLE:
	// if hasPosition && yourSellCondition {
	//     return engine.Signal{
	//         Action:   engine.SignalActionSell,
	//         Symbol:   "BTC/USD",
	//         Quantity: currentPosition.Quantity, // Sell entire position
	//         Reason:   "your sell reason for logging",
	//     }
	// }

	// HOLD (do nothing)
	return engine.Signal{
		Action: engine.SignalActionHold,
		Reason: "no signal",
	}
}

// OnTrade is called after a trade is completed
// Use this for tracking strategy performance metrics
func (s *TemplateStrategy) OnTrade(trade *engine.Trade) {
	// Optional: Track your strategy's performance
	// Example: Update win/loss ratio, average P&L, etc.
	
	// You can log trade results here
	// log.Info("Trade completed", "pnl", trade.NetPnL)
}

// Helper methods for your strategy
// Add any helper functions you need below

// Example: calculateSMA calculates simple moving average
// func (s *TemplateStrategy) calculateSMA(period int) float64 {
//     if len(s.prices) < period {
//         return 0
//     }
//     
//     sum := 0.0
//     for i := len(s.prices) - period; i < len(s.prices); i++ {
//         sum += s.prices[i]
//     }
//     
//     return sum / float64(period)
// }

// Example: calculateRSI calculates relative strength index
// func (s *TemplateStrategy) calculateRSI(period int) float64 {
//     // Implement RSI calculation
//     return 0
// }
