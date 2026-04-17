package strategies

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/indicators"
)

// PulseScalperStrategy (v1.1.0)
type PulseScalperStrategy struct {
	tp1Pct    float64
	tp2Pct    float64
	stage1Hit bool
}

func NewPulseScalperStrategy() *PulseScalperStrategy {
	return &PulseScalperStrategy{
		tp1Pct:    0.015, // 1.5% (Lowered priority)
		tp2Pct:    0.050, // 5.0% (Main Target)
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

	// 1. Indicators (Optimized: Avoid full slice copies)
	size := len(candles)
	if size < 30 {
		return &bot.Decision{Signal: bot.SignalHold}, nil
	}

	// We still need slices for standard TA-Lib like functions, 
	// but we can use a pooled or cached approach eventually.
	// For now, let's at least avoid the manual loop if possible.
	closes := make([]float64, size)
	highs := make([]float64, size)
	lows := make([]float64, size)
	
	for i := 0; i < size; i++ {
		closes[i] = candles[i].Close
		highs[i] = candles[i].High
		lows[i] = candles[i].Low
	}

	// 1. Indicators
	ema9, _ := indicators.EMA(closes, 9)
	ema21, _ := indicators.EMA(closes, 21)
	rsi14, _ := indicators.RSI(closes, 14)
	adx14, _ := indicators.ADX(highs, lows, closes, 14)

	lastEMA9 := ema9[len(ema9)-1]
	prevEMA9 := ema9[len(ema9)-2]
	lastEMA21 := ema21[len(ema21)-1]
	prevEMA21 := ema21[len(ema21)-2]
	lastRSI := rsi14[len(rsi14)-1]
	lastADX := adx14[len(adx14)-1]
	lastClose := candles[len(candles)-1].Close

	decision := &bot.Decision{
		Timestamp: candles[len(candles)-1].Timestamp,
		Price:     lastClose,
		Indicators: map[string]float64{
			"ema_9":  lastEMA9,
			"ema_21": lastEMA21,
			"rsi_14": lastRSI,
			"adx_14": lastADX,
		},
	}

	// ─── PRODUCTION FILTER: ADX (Trend Strength) ─────────────────────────────
	// If ADX < 18, the market is sideways "chopping". Do not enter.
	isTrending := lastADX > 18.0

	// 2. Logic for NO POSITION (Entry)
	if currentPos == nil {
		s.stage1Hit = false
		if !isTrending {
			return &bot.Decision{Signal: bot.SignalHold}, nil
		}

		// LONG: EMA Cross + RSI not overbought
		if prevEMA9 <= prevEMA21 && lastEMA9 > lastEMA21 && lastRSI < 60 {
			decision.Signal = bot.SignalBuy
			decision.Reasoning = "Pulse: Trending Bullish Cross"
			return decision, nil
		}
		// SHORT: EMA Cross + RSI not oversold
		if prevEMA9 >= prevEMA21 && lastEMA9 < lastEMA21 && lastRSI > 40 {
			decision.Signal = bot.SignalSell
			decision.Reasoning = "Pulse: Trending Bearish Cross"
			return decision, nil
		}
		return &bot.Decision{Signal: bot.SignalHold}, nil
	}

	// 3. Logic for ACTIVE POSITION (Adaptive Management)
	entryPrice := currentPos.EntryPrice
	var pnlPct float64
	if currentPos.Side == "long" {
		pnlPct = (lastClose - entryPrice) / entryPrice
	} else {
		pnlPct = (entryPrice - lastClose) / entryPrice
	}

	// ─── SHIELD: Protect at +0.8% profit ───────────────────────────
	if pnlPct >= 0.008 {
		s.stage1Hit = true
	}

	if s.stage1Hit {
		// Exit immediately if profit drops back towards friction zone (0.3% approx)
		if pnlPct < 0.004 {
			decision.Signal = (map[string]bot.Signal{"long": bot.SignalSell, "short": bot.SignalBuy})[currentPos.Side]
			decision.Reasoning = "Safety Lock: Fee Preservation Exit"
			return decision, nil
		}

		// Exit at Target 1 (1.8% profit)
		if pnlPct >= 0.018 {
			decision.Signal = (map[string]bot.Signal{"long": bot.SignalSell, "short": bot.SignalBuy})[currentPos.Side]
			decision.Reasoning = "Pulse Hit: Target 1 (1.8%)"
			return decision, nil
		}
	}

	// EMERGENCY: Pulse Deflated (Trend Reversal)
	isReversed := (currentPos.Side == "long" && lastEMA9 < lastEMA21) || (currentPos.Side == "short" && lastEMA9 > lastEMA21)
	if isReversed {
		decision.Signal = (map[string]bot.Signal{"long": bot.SignalSell, "short": bot.SignalBuy})[currentPos.Side]
		decision.Reasoning = "Pulse Died: Technical Reversal"
		return decision, nil
	}

	return &bot.Decision{Signal: bot.SignalHold}, nil
}

func (s *PulseScalperStrategy) Configure(params map[string]interface{}) error {
	return nil
}
