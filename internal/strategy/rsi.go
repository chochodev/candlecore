package strategy

import (
	"fmt"
)

// RSIStrategy implements an RSI-based trading strategy
type RSIStrategy struct {
	BaseStrategy
	Period         int
	OverboughtLevel float64
	OversoldLevel   float64
}

// NewRSI creates a new RSI strategy
func NewRSI() IStrategy {
	return &RSIStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "RSI Strategy",
			Version: "1.0.0",
			Config: StrategyConfig{
				Stoploss:      -0.04,  // -4% stoploss
				TrailingStop:  true,   // Enable trailing stop
				TrailingDelta: 0.015,  // 1.5% trailing distance
				MinimalROI: map[int]float64{
					0:   0.08, // 8% immediate
					30:  0.04, // 4% after 30 min
					60:  0.02, // 2% after 1 hour
				},
				StakeAmount:   100.0,
				MaxOpenTrades: 3,
				Timeframe:     "5m",
				CustomParams: map[string]interface{}{
					"rsi_period":     14,
					"overbought":     70.0,
					"oversold":       30.0,
				},
			},
		},
		Period:         14,
		OverboughtLevel: 70.0,
		OversoldLevel:   30.0,
	}
}

func init() {
	Register("rsi", NewRSI)
}

// PopulateIndicators calculates RSI
func (s *RSIStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < s.Period+1 {
		return fmt.Errorf("not enough candles for RSI calculation: need %d, have %d", s.Period+1, len(df.Candles))
	}
	
	if df.Indicators == nil {
		df.Indicators = make(map[string][]float64)
	}
	
	// Calculate RSI
	rsi := calculateRSI(df.Candles, s.Period)
	df.Indicators["rsi"] = rsi
	
	return nil
}

// PopulateEntrySignal generates buy signal when RSI crosses above oversold level
func (s *RSIStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	rsi := df.Indicators["rsi"]
	
	if len(rsi) < 2 {
		return Signal{Action: "hold", Confidence: 0, Reason: "Insufficient RSI data"}
	}
	
	rsiCurrent := rsi[len(rsi)-1]
	rsiPrevious := rsi[len(rsi)-2]
	
	// Buy signal: RSI crossing above oversold level (bouncing from oversold)
	if rsiPrevious <= s.OversoldLevel && rsiCurrent > s.OversoldLevel {
		// Confidence based on how oversold it was
		oversoldDepth := s.OversoldLevel - rsiPrevious
		confidence := int(min(oversoldDepth*2+50, 95))
		
		return Signal{
			Action:     "buy",
			Confidence: confidence,
			Reason:     fmt.Sprintf("RSI (%.1f) crossed above oversold level (%.1f), rebounding from %.1f", rsiCurrent, s.OversoldLevel, rsiPrevious),
			Price:      current.Close,
		}
	}
	
	// Strong buy: Deep oversold
	if rsiCurrent < 25 {
		return Signal{
			Action:     "buy",
			Confidence: 80,
			Reason:     fmt.Sprintf("RSI extremely oversold at %.1f", rsiCurrent),
			Price:      current.Close,
		}
	}
	
	return Signal{Action: "hold", Confidence: 0, Reason: fmt.Sprintf("RSI at %.1f, waiting for oversold bounce", rsiCurrent)}
}

// PopulateExitSignal generates sell signal when RSI crosses below overbought level
func (s *RSIStrategy) PopulateExitSignal(df *DataFrame, current Candle, position Position) Signal {
	rsi := df.Indicators["rsi"]
	
	if len(rsi) < 2 {
		return Signal{Action: "hold", Confidence: 0, Reason: "Insufficient RSI data"}
	}
	
	rsiCurrent := rsi[len(rsi)-1]
	rsiPrevious := rsi[len(rsi)-2]
	
	// Sell signal: RSI crossing below overbought level (falling from overbought)
	if rsiPrevious >= s.OverboughtLevel && rsiCurrent < s.OverboughtLevel {
		overboughtDepth := rsiPrevious - s.OverboughtLevel
		confidence := int(min(overboughtDepth*2+50, 95))
		
		return Signal{
			Action:     "sell",
			Confidence: confidence,
			Reason:     fmt.Sprintf("RSI (%.1f) crossed below overbought level (%.1f), falling from %.1f", rsiCurrent, s.OverboughtLevel, rsiPrevious),
			Price:      current.Close,
		}
	}
	
	// Strong sell: Deep overbought
	if rsiCurrent > 75 {
		return Signal{
			Action:     "sell",
			Confidence: 80,
			Reason:     fmt.Sprintf("RSI extremely overbought at %.1f", rsiCurrent),
			Price:      current.Close,
		}
	}
	
	return Signal{Action: "hold", Confidence: 0, Reason: fmt.Sprintf("RSI at %.1f, holding position", rsiCurrent)}
}

// Helper: Calculate RSI
func calculateRSI(candles []Candle, period int) []float64 {
	result := make([]float64, len(candles))
	
	if len(candles) < period+1 {
		return result
	}
	
	// Calculate price changes
	gains := make([]float64, len(candles))
	losses := make([]float64, len(candles))
	
	for i := 1; i < len(candles); i++ {
		change := candles[i].Close - candles[i-1].Close
		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = -change
		}
	}
	
	// Calculate initial average gain and loss
	avgGain := 0.0
	avgLoss := 0.0
	for i := 1; i <= period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)
	
	// Calculate RSI using Wilder's smoothing
	for i := period; i < len(candles); i++ {
		if i > period {
			avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
		}
		
		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - (100 / (1 + rs))
		}
	}
	
	return result
}
