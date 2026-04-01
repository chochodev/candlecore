package strategies

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/indicators"
	"fmt"
)

// PulseScalperStrategy (v1.1.0)
type PulseScalperStrategy struct {
	tp1Pct    float64
	tp2Pct    float64
	stage1Hit bool
}

func NewPulseScalperStrategy() *PulseScalperStrategy {
	return &PulseScalperStrategy{
		tp1Pct:    0.008, // 0.8%
		tp2Pct:    0.025, // 2.5%
		stage1Hit: false,
	}
}

func (s *PulseScalperStrategy) Name() string {
	return "Pulse Scalper v1.1.0"
}

func (s *PulseScalperStrategy) Analyze(candles []exchange.Candle, currentPos *bot.Position) (*bot.Decision, error) {
	if len(candles) < 30 {
		return &bot.Decision{Signal: bot.SignalHold}, nil
	}

	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// 1. Indicators
	ema9, _ := indicators.EMA(closes, 9)
	ema21, _ := indicators.EMA(closes, 21)
	rsi14, _ := indicators.RSI(closes, 14)

	lastEMA9 := ema9[len(ema9)-1]
	prevEMA9 := ema9[len(ema9)-2]
	lastEMA21 := ema21[len(ema21)-1]
	prevEMA21 := ema21[len(ema21)-2]
	lastRSI := rsi14[len(rsi14)-1]
	lastClose := candles[len(candles)-1].Close

	decision := &bot.Decision{
		Timestamp: candles[len(candles)-1].Timestamp,
		Price:     lastClose,
		Indicators: map[string]float64{
			"ema_9":   lastEMA9,
			"ema_21":  lastEMA21,
			"rsi_14":  lastRSI,
		},
	}

	// 2. Logic for NO POSITION (Entry)
	if currentPos == nil {
		s.stage1Hit = false
		// Pulse Condition: EMA 9 crosses above EMA 21 + RSI Healthy
		if prevEMA9 <= prevEMA21 && lastEMA9 > lastEMA21 && lastRSI < 65 {
			decision.Signal = bot.SignalBuy
			decision.Confidence = 85
			decision.Reasoning = "Pulse: EMA 9/21 Crossover + RSI Health"
			return decision, nil
		}
		decision.Signal = bot.SignalHold
		return decision, nil
	}

	// 3. Logic for ACTIVE POSITION (Multi-Stage Exit)
	pnlPct := (lastClose - currentPos.EntryPrice) / currentPos.EntryPrice

	// STAGE 1: Partial Exit (0.8% Profit)
	if !s.stage1Hit && pnlPct >= s.tp1Pct {
		s.stage1Hit = true
		decision.Signal = bot.SignalSell
		decision.Quantity = currentPos.Quantity * 0.5 // Sell half
		decision.Reasoning = fmt.Sprintf("Stage 1 TP (+%.1f%%). Scaling out 50%%.", pnlPct*100)
		return decision, nil
	}

	// STAGE 2: Safety Lock (After Stage 1)
	if s.stage1Hit {
		// Break-even Protection
		if lastClose <= currentPos.EntryPrice {
			decision.Signal = bot.SignalSell
			decision.Reasoning = "Safety Lock: Exit at Break-even after Stage 1"
			return decision, nil
		}
		
		// Final Target
		if pnlPct >= s.tp2Pct {
			decision.Signal = bot.SignalSell
			decision.Reasoning = fmt.Sprintf("Stage 2 TP Hit (+%.1f%%)", pnlPct*100)
			return decision, nil
		}
	}

	// EMERGENCY: Trend Reversal
	if lastEMA9 < lastEMA21 {
		decision.Signal = bot.SignalSell
		decision.Reasoning = "Pulse Died: EMA 9/21 Bearish Cross"
		return decision, nil
	}

	decision.Signal = bot.SignalHold
	return decision, nil
}

func (s *PulseScalperStrategy) Configure(params map[string]interface{}) error {
	return nil
}
